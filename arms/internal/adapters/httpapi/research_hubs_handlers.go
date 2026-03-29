package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/ai"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/adapters/researchclaw"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

func researchHubToJSONMasked(h *domain.ResearchHub) map[string]any {
	hasKey := strings.TrimSpace(h.APIKey) != ""
	return map[string]any{
		"id":            h.ID,
		"display_name":  h.DisplayName,
		"base_url":      h.BaseURL,
		"api_key":       "",
		"has_api_key":   hasKey,
		"created_at":    h.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updated_at":    h.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (h *Handlers) listResearchHubs(w http.ResponseWriter, r *http.Request) {
	if h.ResearchHubs == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "research hubs not available")
		return
	}
	list, err := h.ResearchHubs.List(r.Context(), 500)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	out := make([]any, 0, len(list))
	for i := range list {
		out = append(out, researchHubToJSONMasked(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"research_hubs": out})
}

func (h *Handlers) createResearchHub(w http.ResponseWriter, r *http.Request) {
	if h.ResearchHubs == nil || h.IDs == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "research hubs not available")
		return
	}
	var req createResearchHubReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	now := time.Now().UTC()
	hub := &domain.ResearchHub{
		ID:          h.IDs.NewResearchHubID(),
		DisplayName: strings.TrimSpace(req.DisplayName),
		BaseURL:     strings.TrimSpace(req.BaseURL),
		APIKey:      strings.TrimSpace(req.APIKey),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := h.ResearchHubs.Save(r.Context(), hub); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, researchHubToJSONMasked(hub))
}

func (h *Handlers) patchResearchHub(w http.ResponseWriter, r *http.Request) {
	if h.ResearchHubs == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "research hubs not available")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "validation", "id is required")
		return
	}
	var req patchResearchHubReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.empty() {
		writeError(w, http.StatusBadRequest, "validation", "at least one field is required")
		return
	}
	cur, err := h.ResearchHubs.ByID(r.Context(), id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	next := *cur
	if req.DisplayName != nil {
		next.DisplayName = strings.TrimSpace(*req.DisplayName)
	}
	if req.BaseURL != nil {
		next.BaseURL = strings.TrimSpace(*req.BaseURL)
		if err := validateResearchHubBaseURL(next.BaseURL); err != nil {
			writeError(w, http.StatusBadRequest, "validation", err.Error())
			return
		}
	}
	if req.APIKey != nil {
		next.APIKey = strings.TrimSpace(*req.APIKey)
	}
	next.UpdatedAt = time.Now().UTC()
	if err := h.ResearchHubs.Update(r.Context(), &next); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, researchHubToJSONMasked(&next))
}

func (h *Handlers) deleteResearchHub(w http.ResponseWriter, r *http.Request) {
	if h.ResearchHubs == nil || h.ResearchSettings == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "research hubs not available")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "validation", "id is required")
		return
	}
	st, _ := h.ResearchSettings.Get(r.Context())
	if strings.TrimSpace(st.DefaultResearchHubID) == id {
		_ = h.ResearchSettings.Upsert(r.Context(), domain.ResearchSystemSettings{
			AutoResearchClawEnabled: st.AutoResearchClawEnabled,
			DefaultResearchHubID:    "",
		})
	}
	if err := h.ResearchHubs.Delete(r.Context(), id); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func mergeResearchHubTestDraft(base domain.ResearchHub, d *testResearchHubDraft) domain.ResearchHub {
	if d == nil {
		return base
	}
	out := base
	if d.BaseURL != nil {
		out.BaseURL = strings.TrimSpace(*d.BaseURL)
	}
	if d.APIKey != nil {
		out.APIKey = strings.TrimSpace(*d.APIKey)
	}
	return out
}

func (h *Handlers) postResearchHubTest(w http.ResponseWriter, r *http.Request) {
	if h.ResearchHubs == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "research hubs not available")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "validation", "id is required")
		return
	}
	cur, err := h.ResearchHubs.ByID(r.Context(), id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	b, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "read body: "+err.Error())
		return
	}
	var req testResearchHubReq
	if strings.TrimSpace(string(b)) != "" {
		if err := json.Unmarshal(b, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
	}
	ep := mergeResearchHubTestDraft(*cur, req.Draft)
	if err := validateResearchHubBaseURL(ep.BaseURL); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	hc := ai.DefaultHTTPClient()
	health, version, err := researchclaw.Probe(r.Context(), ep.BaseURL, ep.APIKey, hc)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}
	out := map[string]any{"ok": true, "health": health}
	if len(version) > 0 {
		out["version"] = version
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) postResearchHubInvoke(w http.ResponseWriter, r *http.Request) {
	if h.ResearchHubs == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "research hubs not available")
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "validation", "id is required")
		return
	}
	cur, err := h.ResearchHubs.ByID(r.Context(), id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	rb, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "read body: "+err.Error())
		return
	}
	var req invokeResearchHubReq
	if err := json.Unmarshal(rb, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	if !researchclaw.AllowedInvokePath(req.Method, req.Path) {
		writeError(w, http.StatusBadRequest, "validation", "path not allowed for ResearchClaw invoke")
		return
	}
	hc := ai.DefaultHTTPClient()
	status, body, err := researchclaw.InvokeAllowlisted(r.Context(), hc, cur, req.Method, req.Path, req.JSONBody)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	out := map[string]any{"status": status, "body": string(body)}
	if len(bytes.TrimSpace(body)) > 0 && json.Valid(body) {
		var parsed any
		if json.Unmarshal(body, &parsed) == nil {
			out["json"] = parsed
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) getResearchSystemSettings(w http.ResponseWriter, r *http.Request) {
	if h.ResearchSettings == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "research system settings not available")
		return
	}
	st, err := h.ResearchSettings.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"auto_research_claw_enabled": st.AutoResearchClawEnabled,
		"default_research_hub_id":    strings.TrimSpace(st.DefaultResearchHubID),
	})
}

func (h *Handlers) patchResearchSystemSettings(w http.ResponseWriter, r *http.Request) {
	if h.ResearchSettings == nil || h.ResearchHubs == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "research system settings not available")
		return
	}
	var req patchResearchSystemSettingsReq
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.empty() {
		writeError(w, http.StatusBadRequest, "validation", "at least one field is required")
		return
	}
	cur, err := h.ResearchSettings.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	next := cur
	if req.AutoResearchClawEnabled != nil {
		next.AutoResearchClawEnabled = *req.AutoResearchClawEnabled
	}
	if req.DefaultResearchHubID != nil {
		v := strings.TrimSpace(*req.DefaultResearchHubID)
		next.DefaultResearchHubID = v
		if v != "" {
			if _, err := h.ResearchHubs.ByID(r.Context(), v); err != nil {
				if mapDomainErr(w, err) {
					return
				}
				writeError(w, http.StatusBadRequest, "bad_request", err.Error())
				return
			}
		}
	}
	if err := h.ResearchSettings.Upsert(r.Context(), next); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"auto_research_claw_enabled": next.AutoResearchClawEnabled,
		"default_research_hub_id":    strings.TrimSpace(next.DefaultResearchHubID),
	})
}
