package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/nlp/tfidftags"
)

type postTfidfSuggestTagsReq struct {
	Corpus       []string `json:"corpus"`
	Text         string   `json:"text"`
	TopK         int      `json:"top_k"`
	MinTokenLen  int      `json:"min_token_len"`
}

type postProductTfidfSuggestTagsReq struct {
	Text         string   `json:"text"`
	IdeaID       string   `json:"idea_id"`
	ExtraCorpus  []string `json:"extra_corpus"`
	TopK         int      `json:"top_k"`
	MinTokenLen  int      `json:"min_token_len"`
}

func (h *Handlers) postTfidfSuggestTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}
	var req postTfidfSuggestTagsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "text is required")
		return
	}
	tags := tfidftags.Suggest(req.Corpus, req.Text, req.TopK, req.MinTokenLen)
	writeJSON(w, http.StatusOK, map[string]any{
		"tags":               tags,
		"method":             tfidfMethodFromCorpusCount(countNonEmptyCorpus(req.Corpus)),
		"corpus_documents":   countNonEmptyCorpus(req.Corpus),
	})
}

func (h *Handlers) postProductTfidfSuggestTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}
	pid := domain.ProductID(r.PathValue("id"))
	if _, err := h.Autopilot.Products.ByID(r.Context(), pid); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	var req postProductTfidfSuggestTagsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	ideas, err := h.Autopilot.Ideas.ListByProduct(r.Context(), pid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}

	var target string
	var exclude domain.IdeaID
	if id := strings.TrimSpace(req.IdeaID); id != "" {
		exclude = domain.IdeaID(id)
		idea, ierr := h.Autopilot.Ideas.ByID(r.Context(), exclude)
		if ierr != nil {
			if mapDomainErr(w, ierr) {
				return
			}
			writeError(w, http.StatusInternalServerError, "internal", ierr.Error())
			return
		}
		if idea.ProductID != pid {
			writeError(w, http.StatusBadRequest, "bad_request", "idea does not belong to this product")
			return
		}
		target = ideaText(idea)
	} else {
		target = strings.TrimSpace(req.Text)
	}
	if target == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "text or idea_id is required")
		return
	}

	corpus := make([]string, 0, len(ideas)+len(req.ExtraCorpus))
	for i := range ideas {
		if exclude != "" && ideas[i].ID == exclude {
			continue
		}
		corpus = append(corpus, ideaText(&ideas[i]))
	}
	for _, s := range req.ExtraCorpus {
		if t := strings.TrimSpace(s); t != "" {
			corpus = append(corpus, t)
		}
	}
	tags := tfidftags.Suggest(corpus, target, req.TopK, req.MinTokenLen)
	writeJSON(w, http.StatusOK, map[string]any{
		"tags":             tags,
		"method":           tfidfMethodFromCorpusCount(len(corpus)),
		"corpus_documents": len(corpus),
		"product_id":       string(pid),
		"idea_id":          strings.TrimSpace(req.IdeaID),
	})
}

func ideaText(idea *domain.Idea) string {
	var b strings.Builder
	b.WriteString(idea.Title)
	b.WriteByte(' ')
	b.WriteString(idea.Description)
	b.WriteByte(' ')
	b.WriteString(idea.Reasoning)
	b.WriteByte(' ')
	b.WriteString(strings.Join(idea.Tags, " "))
	return strings.TrimSpace(b.String())
}

func countNonEmptyCorpus(c []string) int {
	n := 0
	for _, s := range c {
		if strings.TrimSpace(s) != "" {
			n++
		}
	}
	return n
}

func tfidfMethodFromCorpusCount(usableCorpusDocs int) string {
	if usableCorpusDocs == 0 {
		return "frequency"
	}
	return "tfidf"
}

type postProductSuggestIdeaIDReq struct {
	Spec           string   `json:"spec"`
	Statement      string   `json:"statement"`
	ExtraCorpus    []string `json:"extra_corpus,omitempty"`
	TopK           int      `json:"top_k,omitempty"`
	MinTokenLen    int      `json:"min_token_len,omitempty"`
	MaxSlugTokens  int      `json:"max_slug_tokens,omitempty"`
}

// postProductSuggestIdeaID returns a globally unique idea id string derived from TF-IDF salient
// tokens over spec+statement, with a numeric suffix chosen so the id is not already used in ideas.
func (h *Handlers) postProductSuggestIdeaID(w http.ResponseWriter, r *http.Request) {
	var req postProductSuggestIdeaIDReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	pid := domain.ProductID(r.PathValue("id"))
	if _, err := h.Autopilot.Products.ByID(r.Context(), pid); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	spec := strings.TrimSpace(req.Spec)
	stmt := strings.TrimSpace(req.Statement)
	if spec == "" && stmt == "" {
		writeError(w, http.StatusBadRequest, "validation", "at least one of spec or statement is required")
		return
	}
	target := strings.TrimSpace(spec + "\n" + stmt)

	ideas, err := h.Autopilot.Ideas.ListByProduct(r.Context(), pid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	corpus := make([]string, 0, len(ideas)+len(req.ExtraCorpus))
	for i := range ideas {
		corpus = append(corpus, ideaText(&ideas[i]))
	}
	for _, s := range req.ExtraCorpus {
		if t := strings.TrimSpace(s); t != "" {
			corpus = append(corpus, t)
		}
	}
	tags := tfidftags.Suggest(corpus, target, req.TopK, req.MinTokenLen)
	base := tfidftags.SlugPrefixFromTags(tags, req.MaxSlugTokens, 48)
	if base == "" {
		base = "idea"
	}

	ctx := r.Context()
	const maxTry = 10000
	for n := 1; n <= maxTry; n++ {
		candidate := fmt.Sprintf("%s-%d", base, n)
		if len(candidate) > 200 {
			candidate = fmt.Sprintf("idea-%d", n)
		}
		_, ierr := h.Autopilot.Ideas.ByID(ctx, domain.IdeaID(candidate))
		if errors.Is(ierr, domain.ErrNotFound) {
			writeJSON(w, http.StatusOK, map[string]any{
				"idea_id":          candidate,
				"base_slug":        base,
				"suffix":           n,
				"tags":             tags,
				"method":           tfidfMethodFromCorpusCount(len(corpus)),
				"corpus_documents": len(corpus),
				"product_id":       string(pid),
			})
			return
		}
		if ierr != nil {
			writeError(w, http.StatusInternalServerError, "internal", ierr.Error())
			return
		}
	}
	writeError(w, http.StatusConflict, "conflict", "could not allocate a unique idea_id")
}
