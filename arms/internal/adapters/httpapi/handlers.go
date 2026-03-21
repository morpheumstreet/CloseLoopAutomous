package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/application/autopilot"
	"github.com/closeloopautomous/arms/internal/application/convoy"
	"github.com/closeloopautomous/arms/internal/application/cost"
	productapp "github.com/closeloopautomous/arms/internal/application/product"
	"github.com/closeloopautomous/arms/internal/application/task"
	"github.com/closeloopautomous/arms/internal/domain"
	"github.com/closeloopautomous/arms/internal/ports"
)

const webhookSigHeader = "X-Arms-Signature"

// Handlers holds application services for the HTTP adapter.
type Handlers struct {
	Config    Config
	Product   *productapp.Service
	Autopilot *autopilot.Service
	Task      *task.Service
	Convoy    *convoy.Service
	Cost      *cost.Service
	Live      ports.ActivityStream // SSE subscribers; required for live routes
}

func (h *Handlers) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) routesDoc(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"routes": routeCatalog()})
}

func (h *Handlers) createProduct(w http.ResponseWriter, r *http.Request) {
	var req createProductReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	p, err := h.Product.Register(r.Context(), productapp.RegistrationInput{
		Name:                 req.Name,
		WorkspaceID:          req.WorkspaceID,
		RepoURL:              req.RepoURL,
		RepoBranch:           req.RepoBranch,
		Description:          req.Description,
		ProgramDocument:      req.ProgramDocument,
		SettingsJSON:         req.SettingsJSON,
		IconURL:              req.IconURL,
		ResearchCadenceSec:   req.ResearchCadenceSec,
		IdeationCadenceSec:   req.IdeationCadenceSec,
		AutomationTier:       req.AutomationTier,
		AutoDispatchEnabled:  req.AutoDispatchEnabled,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, productToJSON(p))
}

func (h *Handlers) patchProduct(w http.ResponseWriter, r *http.Request) {
	var req patchProductReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	id := domain.ProductID(r.PathValue("id"))
	patch := productapp.MetadataPatch{
		Name:                 req.Name,
		RepoURL:              req.RepoURL,
		RepoBranch:           req.RepoBranch,
		Description:          req.Description,
		ProgramDocument:      req.ProgramDocument,
		SettingsJSON:         req.SettingsJSON,
		IconURL:              req.IconURL,
		ResearchCadenceSec:   req.ResearchCadenceSec,
		IdeationCadenceSec:   req.IdeationCadenceSec,
		AutomationTier:       req.AutomationTier,
		AutoDispatchEnabled:  req.AutoDispatchEnabled,
	}
	p, err := h.Product.PatchMetadata(r.Context(), id, patch)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, productToJSON(p))
}

func (h *Handlers) getProduct(w http.ResponseWriter, r *http.Request) {
	id := domain.ProductID(r.PathValue("id"))
	p, err := h.Autopilot.Products.ByID(r.Context(), id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, productToJSON(p))
}

func (h *Handlers) runResearch(w http.ResponseWriter, r *http.Request) {
	id := domain.ProductID(r.PathValue("id"))
	if err := h.Autopilot.RunResearch(r.Context(), id); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	p, err := h.Autopilot.Products.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, productToJSON(p))
}

func (h *Handlers) runIdeation(w http.ResponseWriter, r *http.Request) {
	id := domain.ProductID(r.PathValue("id"))
	if err := h.Autopilot.RunIdeation(r.Context(), id); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	p, err := h.Autopilot.Products.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, productToJSON(p))
}

func (h *Handlers) listIdeas(w http.ResponseWriter, r *http.Request) {
	id := domain.ProductID(r.PathValue("id"))
	list, err := h.Autopilot.Ideas.ListByProduct(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]any, 0, len(list))
	for i := range list {
		out = append(out, ideaToJSON(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"ideas": out})
}

func (h *Handlers) listProductTasks(w http.ResponseWriter, r *http.Request) {
	id := domain.ProductID(r.PathValue("id"))
	list, err := h.Task.ListByProduct(r.Context(), id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]any, 0, len(list))
	for i := range list {
		out = append(out, taskToJSON(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": out})
}

func (h *Handlers) listProductConvoys(w http.ResponseWriter, r *http.Request) {
	id := domain.ProductID(r.PathValue("id"))
	list, err := h.Convoy.ListByProduct(r.Context(), id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]any, 0, len(list))
	for i := range list {
		out = append(out, convoyToJSON(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"convoys": out})
}

func (h *Handlers) swipe(w http.ResponseWriter, r *http.Request) {
	var req swipeReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	dec, err := parseSwipe(req.Decision)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	id := domain.IdeaID(r.PathValue("id"))
	if err := h.Autopilot.SubmitSwipe(r.Context(), id, dec); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	idea, err := h.Autopilot.Ideas.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ideaToJSON(idea))
}

func (h *Handlers) listMaybePool(w http.ResponseWriter, r *http.Request) {
	pid := domain.ProductID(r.PathValue("id"))
	if _, err := h.Autopilot.Products.ByID(r.Context(), pid); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	ids, err := h.Autopilot.MaybePool.ListIdeaIDsByProduct(r.Context(), pid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]any, 0, len(ids))
	for _, iid := range ids {
		idea, err := h.Autopilot.Ideas.ByID(r.Context(), iid)
		if err != nil {
			if mapDomainErr(w, err) {
				return
			}
			writeError(w, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		out = append(out, ideaToJSON(idea))
	}
	writeJSON(w, http.StatusOK, map[string]any{"ideas": out})
}

func (h *Handlers) promoteMaybe(w http.ResponseWriter, r *http.Request) {
	id := domain.IdeaID(r.PathValue("id"))
	if err := h.Autopilot.PromoteMaybe(r.Context(), id); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	idea, err := h.Autopilot.Ideas.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ideaToJSON(idea))
}

func (h *Handlers) createTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	t, err := h.Task.CreateFromApprovedIdea(r.Context(), domain.IdeaID(req.IdeaID), req.Spec)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, taskToJSON(t))
}

func (h *Handlers) getTask(w http.ResponseWriter, r *http.Request) {
	id := domain.TaskID(r.PathValue("id"))
	t, err := h.Task.Tasks.ByID(r.Context(), id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, taskToJSON(t))
}

func (h *Handlers) patchTask(w http.ResponseWriter, r *http.Request) {
	var req patchTaskReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	ctx := r.Context()
	id := domain.TaskID(r.PathValue("id"))
	if req.ClarificationsJSON != nil {
		if err := h.Task.UpdatePlanningArtifacts(ctx, id, *req.ClarificationsJSON); err != nil {
			if mapDomainErr(w, err) {
				return
			}
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
	}
	if req.Status != nil {
		to, err := domain.ParseTaskStatus(*req.Status)
		if err != nil {
			writeError(w, http.StatusBadRequest, "validation", err.Error())
			return
		}
		reason := ""
		if req.StatusReason != nil {
			reason = *req.StatusReason
		}
		if err := h.Task.SetKanbanStatus(ctx, id, to, reason); err != nil {
			if mapDomainErr(w, err) {
				return
			}
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
	} else if req.StatusReason != nil {
		if err := h.Task.SetStatusReasonOnly(ctx, id, *req.StatusReason); err != nil {
			if mapDomainErr(w, err) {
				return
			}
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
	}
	t, err := h.Task.Tasks.ByID(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, taskToJSON(t))
}

func (h *Handlers) approvePlan(w http.ResponseWriter, r *http.Request) {
	var req approvePlanReq
	if err := decodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	id := domain.TaskID(r.PathValue("id"))
	if err := h.Task.ApprovePlan(r.Context(), id, req.Spec); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	t, err := h.Task.Tasks.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, taskToJSON(t))
}

func (h *Handlers) rejectPlan(w http.ResponseWriter, r *http.Request) {
	var req rejectPlanReq
	if err := decodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	id := domain.TaskID(r.PathValue("id"))
	if err := h.Task.ReturnToPlanning(r.Context(), id, req.StatusReason); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	t, err := h.Task.Tasks.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, taskToJSON(t))
}

func (h *Handlers) dispatchTask(w http.ResponseWriter, r *http.Request) {
	var req dispatchReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	id := domain.TaskID(r.PathValue("id"))
	if err := h.Task.Dispatch(r.Context(), id, req.EstimatedCost); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	t, err := h.Task.Tasks.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, taskToJSON(t))
}

func (h *Handlers) checkpoint(w http.ResponseWriter, r *http.Request) {
	var req checkpointReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	id := domain.TaskID(r.PathValue("id"))
	if err := h.Task.RecordCheckpoint(r.Context(), id, req.Payload); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	t, err := h.Task.Tasks.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, taskToJSON(t))
}

func (h *Handlers) completeTask(w http.ResponseWriter, r *http.Request) {
	id := domain.TaskID(r.PathValue("id"))
	if err := h.Task.Complete(r.Context(), id); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	t, err := h.Task.Tasks.ByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, taskToJSON(t))
}

func (h *Handlers) createConvoy(w http.ResponseWriter, r *http.Request) {
	var req createConvoyReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	subs := make([]domain.Subtask, len(req.Subtasks))
	for i := range req.Subtasks {
		deps := make([]domain.SubtaskID, len(req.Subtasks[i].DependsOn))
		for j, d := range req.Subtasks[i].DependsOn {
			deps[j] = domain.SubtaskID(d)
		}
		subs[i] = domain.Subtask{
			ID:        domain.SubtaskID(req.Subtasks[i].ID),
			DependsOn: deps,
			AgentRole: req.Subtasks[i].AgentRole,
		}
	}
	c, err := h.Convoy.Create(r.Context(), domain.TaskID(req.ParentTaskID), domain.ProductID(req.ProductID), subs)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, convoyToJSON(c))
}

func (h *Handlers) getConvoy(w http.ResponseWriter, r *http.Request) {
	id := domain.ConvoyID(r.PathValue("id"))
	c, err := h.Convoy.Get(r.Context(), id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, convoyToJSON(c))
}

func (h *Handlers) dispatchConvoy(w http.ResponseWriter, r *http.Request) {
	id := domain.ConvoyID(r.PathValue("id"))
	if err := h.Convoy.DispatchReady(r.Context(), id); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	c, err := h.Convoy.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, convoyToJSON(c))
}

func (h *Handlers) recordCost(w http.ResponseWriter, r *http.Request) {
	var req recordCostReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	if err := h.Cost.Record(r.Context(), domain.ProductID(req.ProductID), domain.TaskID(req.TaskID), req.Amount, req.Note); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "recorded"})
}

func (h *Handlers) agentsStub(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": []any{}, "stub": true})
}

func (h *Handlers) openclawStub(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "openclaw proxy not implemented; use AgentGateway adapter")
}

func (h *Handlers) workspacesStub(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": []any{}, "stub": true})
}

func (h *Handlers) settingsStub(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{})
}

func (h *Handlers) agentCompletionWebhook(w http.ResponseWriter, r *http.Request) {
	if h.Config.WebhookSecret == "" {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "WEBHOOK_SECRET is not set")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "body", err.Error())
		return
	}
	sig := r.Header.Get(webhookSigHeader)
	if !hmacSHA256Equal(h.Config.WebhookSecret, body, sig) {
		writeError(w, http.StatusUnauthorized, "invalid_signature", "HMAC verification failed")
		return
	}
	var req agentCompletionReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	if err := h.Task.Complete(r.Context(), domain.TaskID(req.TaskID)); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func hmacSHA256Equal(secret string, body []byte, sigHex string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	want := mac.Sum(nil)
	got, err := hex.DecodeString(strings.TrimSpace(sigHex))
	if err != nil || len(got) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare(want, got) == 1
}

func (h *Handlers) liveSSE(w http.ResponseWriter, r *http.Request) {
	if h.Live == nil {
		writeError(w, http.StatusInternalServerError, "streaming", "live activity stream not configured")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	fl, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming", "response writer does not flush")
		return
	}
	enc := func(v any) {
		b, _ := json.Marshal(v)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
		fl.Flush()
	}
	enc(map[string]any{"event": "hello", "ts": time.Now().UTC().Format(time.RFC3339Nano)})
	ch, unsub := h.Live.Subscribe()
	defer unsub()
	tick := time.NewTicker(25 * time.Second)
	defer tick.Stop()
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case payload := <-ch:
			_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
			fl.Flush()
		case <-tick.C:
			_, _ = fmt.Fprintf(w, ": ping\n\n")
			fl.Flush()
		}
	}
}

// JSON views

func productToJSON(p *domain.Product) map[string]any {
	m := map[string]any{
		"id":                    string(p.ID),
		"name":                  p.Name,
		"stage":                 p.Stage.String(),
		"research_summary":      p.ResearchSummary,
		"workspace_id":          p.WorkspaceID,
		"repo_url":              p.RepoURL,
		"repo_branch":           p.RepoBranch,
		"description":           p.Description,
		"program_document":      p.ProgramDocument,
		"settings_json":         p.SettingsJSON,
		"icon_url":              p.IconURL,
		"research_cadence_sec":  p.ResearchCadenceSec,
		"ideation_cadence_sec":  p.IdeationCadenceSec,
		"automation_tier":       p.AutomationTier.String(),
		"auto_dispatch_enabled": p.AutoDispatchEnabled,
		"preference_model_json": p.PreferenceModelJSON,
		"updated_at":            p.UpdatedAt.Format(time.RFC3339Nano),
	}
	if !p.LastAutoResearchAt.IsZero() {
		m["last_auto_research_at"] = p.LastAutoResearchAt.Format(time.RFC3339Nano)
	}
	if !p.LastAutoIdeationAt.IsZero() {
		m["last_auto_ideation_at"] = p.LastAutoIdeationAt.Format(time.RFC3339Nano)
	}
	return m
}

func ideaToJSON(i *domain.Idea) map[string]any {
	return map[string]any{
		"id":          string(i.ID),
		"product_id":  string(i.ProductID),
		"title":       i.Title,
		"description": i.Description,
		"impact":      i.Impact,
		"feasibility": i.Feasibility,
		"reasoning":   i.Reasoning,
		"decided":     i.Decided,
		"decision":    swipeString(i.Decision),
		"created_at":  i.CreatedAt.Format(time.RFC3339Nano),
	}
}

func swipeString(d domain.SwipeDecision) string {
	switch d {
	case domain.DecisionPass:
		return "pass"
	case domain.DecisionMaybe:
		return "maybe"
	case domain.DecisionYes:
		return "yes"
	case domain.DecisionNow:
		return "now"
	default:
		return ""
	}
}

func taskToJSON(t *domain.Task) map[string]any {
	return map[string]any{
		"id":                  string(t.ID),
		"product_id":          string(t.ProductID),
		"idea_id":             string(t.IdeaID),
		"spec":                t.Spec,
		"status":              t.Status.String(),
		"status_reason":       t.StatusReason,
		"plan_approved":       t.PlanApproved,
		"clarifications_json": t.ClarificationsJSON,
		"checkpoint":          t.Checkpoint,
		"external_ref":        t.ExternalRef,
		"created_at":          t.CreatedAt.Format(time.RFC3339Nano),
		"updated_at":          t.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func convoyToJSON(c *domain.Convoy) map[string]any {
	subs := make([]map[string]any, len(c.Subtasks))
	for i := range c.Subtasks {
		deps := make([]string, len(c.Subtasks[i].DependsOn))
		for j, d := range c.Subtasks[i].DependsOn {
			deps[j] = string(d)
		}
		subs[i] = map[string]any{
			"id":              string(c.Subtasks[i].ID),
			"agent_role":      c.Subtasks[i].AgentRole,
			"depends_on":      deps,
			"dispatched":      c.Subtasks[i].Dispatched,
			"external_ref":    c.Subtasks[i].ExternalRef,
			"last_checkpoint": c.Subtasks[i].LastCheckpoint,
		}
	}
	return map[string]any{
		"id":         string(c.ID),
		"product_id": string(c.ProductID),
		"parent_id":  string(c.ParentID),
		"subtasks":   subs,
		"created_at": c.CreatedAt.Format(time.RFC3339Nano),
	}
}
