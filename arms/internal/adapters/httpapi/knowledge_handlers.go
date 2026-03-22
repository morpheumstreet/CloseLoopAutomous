package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/closeloopautomous/arms/internal/domain"
)

func knowledgeToJSON(e *domain.KnowledgeEntry) map[string]any {
	if e == nil {
		return nil
	}
	out := map[string]any{
		"id":         e.ID,
		"product_id": string(e.ProductID),
		"content":    e.Content,
		"created_at": e.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updated_at": e.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if tid := strings.TrimSpace(string(e.TaskID)); tid != "" {
		out["task_id"] = tid
	}
	var meta any
	if strings.TrimSpace(e.MetadataJSON) != "" && json.Valid([]byte(e.MetadataJSON)) {
		_ = json.Unmarshal([]byte(e.MetadataJSON), &meta)
	}
	if meta == nil {
		meta = map[string]any{}
	}
	out["metadata"] = meta
	return out
}

func (h *Handlers) requireKnowledge(w http.ResponseWriter) bool {
	if h.Knowledge == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "knowledge service not wired")
		return false
	}
	return true
}

func (h *Handlers) postProductKnowledge(w http.ResponseWriter, r *http.Request) {
	if !h.requireKnowledge(w) {
		return
	}
	var req postKnowledgeReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	ctx := r.Context()
	pid := domain.ProductID(r.PathValue("id"))
	var tid domain.TaskID
	if strings.TrimSpace(req.TaskID) != "" {
		tid = domain.TaskID(strings.TrimSpace(req.TaskID))
	}
	e, err := h.Knowledge.Create(ctx, pid, req.Content, tid, req.Metadata)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	detail, _ := json.Marshal(map[string]any{"knowledge_id": e.ID})
	h.logOperation(ctx, "http", "product.knowledge.create", "product", string(pid), string(detail), pid)
	writeJSON(w, http.StatusCreated, knowledgeToJSON(e))
}

func (h *Handlers) listProductKnowledge(w http.ResponseWriter, r *http.Request) {
	if !h.requireKnowledge(w) {
		return
	}
	pid := domain.ProductID(r.PathValue("id"))
	limit := 100
	if q := strings.TrimSpace(r.URL.Query().Get("limit")); q != "" {
		n, err := strconv.Atoi(q)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "validation", "limit must be a positive integer")
			return
		}
		limit = n
	}
	ctx := r.Context()
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	var list []domain.KnowledgeEntry
	var err error
	if q != "" {
		list, err = h.Knowledge.Search(ctx, pid, q, limit)
	} else {
		list, err = h.Knowledge.List(ctx, pid, limit)
	}
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]any, 0, len(list))
	for i := range list {
		out = append(out, knowledgeToJSON(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": out})
}

func (h *Handlers) getProductKnowledgeEntry(w http.ResponseWriter, r *http.Request) {
	if !h.requireKnowledge(w) {
		return
	}
	pid := domain.ProductID(r.PathValue("id"))
	id, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("entryId")), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "validation", "entryId must be a positive integer")
		return
	}
	e, err := h.Knowledge.Get(r.Context(), id, pid)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, knowledgeToJSON(e))
}

func (h *Handlers) patchProductKnowledgeEntry(w http.ResponseWriter, r *http.Request) {
	if !h.requireKnowledge(w) {
		return
	}
	var req patchKnowledgeReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	pid := domain.ProductID(r.PathValue("id"))
	id, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("entryId")), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "validation", "entryId must be a positive integer")
		return
	}
	ctx := r.Context()
	e, err := h.Knowledge.Update(ctx, id, pid, req.Content, req.Metadata)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	detail, _ := json.Marshal(map[string]any{"knowledge_id": e.ID})
	h.logOperation(ctx, "http", "product.knowledge.update", "product", string(pid), string(detail), pid)
	writeJSON(w, http.StatusOK, knowledgeToJSON(e))
}

func (h *Handlers) deleteProductKnowledgeEntry(w http.ResponseWriter, r *http.Request) {
	if !h.requireKnowledge(w) {
		return
	}
	pid := domain.ProductID(r.PathValue("id"))
	id, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("entryId")), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "validation", "entryId must be a positive integer")
		return
	}
	ctx := r.Context()
	if err := h.Knowledge.Delete(ctx, id, pid); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	detail, _ := json.Marshal(map[string]any{"knowledge_id": id})
	h.logOperation(ctx, "http", "product.knowledge.delete", "product", string(pid), string(detail), pid)
	w.WriteHeader(http.StatusNoContent)
}
