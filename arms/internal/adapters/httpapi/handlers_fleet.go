package httpapi

import (
	"net/http"
	"sort"
	"strconv"
)

func (h *Handlers) postFleetRefresh(w http.ResponseWriter, r *http.Request) {
	if h.AgentIdentity == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "agent identity refresh not available")
		return
	}
	if err := h.AgentIdentity.RefreshAll(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) listFleetIdentities(w http.ResponseWriter, r *http.Request) {
	if h.AgentIdentity == nil || h.AgentIdentity.Profiles == nil {
		writeJSON(w, http.StatusOK, map[string]any{"identities": []any{}})
		return
	}
	limit := 200
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	list, err := h.AgentIdentity.Profiles.List(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]any, 0, len(list))
	for i := range list {
		out = append(out, list[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{"identities": out})
}

func (h *Handlers) getFleetIdentity(w http.ResponseWriter, r *http.Request) {
	if h.AgentIdentity == nil || h.AgentIdentity.Profiles == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "agent identity not available")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "validation", "missing id")
		return
	}
	ident, err := h.AgentIdentity.Profiles.ByID(r.Context(), id)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ident)
}

func (h *Handlers) fleetGeoSummary(w http.ResponseWriter, r *http.Request) {
	if h.AgentIdentity == nil || h.AgentIdentity.Profiles == nil {
		writeJSON(w, http.StatusOK, map[string]any{"countries": []any{}, "total": 0})
		return
	}
	list, err := h.AgentIdentity.Profiles.List(r.Context(), 500)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	counts := make(map[string]int)
	var withGeo int
	for i := range list {
		g := list[i].Geo
		if g == nil || g.Source == "" || g.Source == "none" {
			continue
		}
		withGeo++
		iso := g.CountryISO
		if iso == "" {
			iso = "_unknown"
		}
		counts[iso]++
	}
	type row struct {
		CountryISO string `json:"country_iso"`
		Count      int    `json:"count"`
	}
	var rows []row
	for iso, n := range counts {
		rows = append(rows, row{CountryISO: iso, Count: n})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].CountryISO < rows[j].CountryISO })
	writeJSON(w, http.StatusOK, map[string]any{
		"countries":          rows,
		"total_identities":   len(list),
		"with_geo":           withGeo,
	})
}
