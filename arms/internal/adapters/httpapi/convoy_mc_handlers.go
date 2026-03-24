package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/application/convoy"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

func decodeJSONLoose(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	return dec.Decode(dst)
}

type mcPostTaskConvoyReq struct {
	Strategy          string               `json:"strategy"`
	Name              string               `json:"name"`
	Subtasks          []mcPostTaskSubtask  `json:"subtasks"`
	DecompositionSpec string               `json:"decomposition_spec"`
}

type mcPostTaskSubtask struct {
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	SuggestedRole string   `json:"suggested_role"`
	AgentID       string   `json:"agent_id"`
	DependsOn     []string `json:"depends_on"`
}

type mcPatchTaskConvoyReq struct {
	Status string `json:"status"`
}

type mcPostTaskConvoySubtasksReq struct {
	Subtasks []mcPostTaskSubtask `json:"subtasks"`
}

func mcSubtasksToDomain(subs []mcPostTaskSubtask) ([]domain.Subtask, error) {
	out := make([]domain.Subtask, 0, len(subs))
	for i := range subs {
		title := strings.TrimSpace(subs[i].Title)
		if title == "" {
			return nil, errors.New("subtask title is required")
		}
		role := strings.TrimSpace(subs[i].SuggestedRole)
		if role == "" {
			role = strings.TrimSpace(subs[i].AgentID)
		}
		if role == "" {
			role = "builder"
		}
		meta := map[string]any{}
		if d := strings.TrimSpace(subs[i].Description); d != "" {
			meta["description"] = d
		}
		if aid := strings.TrimSpace(subs[i].AgentID); aid != "" {
			meta["assigned_agent_id_hint"] = aid
		}
		mj := "{}"
		if len(meta) > 0 {
			b, err := json.Marshal(meta)
			if err != nil {
				return nil, err
			}
			mj = string(b)
		}
		deps := make([]domain.SubtaskID, len(subs[i].DependsOn))
		for j, d := range subs[i].DependsOn {
			deps[j] = domain.SubtaskID(strings.TrimSpace(d))
		}
		out = append(out, domain.Subtask{
			Title:        title,
			AgentRole:    role,
			MetadataJSON: mj,
			DependsOn:    deps,
		})
	}
	return out, nil
}

func (h *Handlers) getTaskConvoyMC(w http.ResponseWriter, r *http.Request) {
	if h.Convoy == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "convoy service not available")
		return
	}
	tid := domain.TaskID(r.PathValue("id"))
	c, err := h.Convoy.GetByParentTask(r.Context(), tid)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "No convoy found for this task")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	parent, err := h.Task.TaskByIDForAPI(r.Context(), tid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, convoyToMCMissionControlJSON(c, parent.Spec))
}

func (h *Handlers) postTaskConvoyMC(w http.ResponseWriter, r *http.Request) {
	if h.Convoy == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "convoy service not available")
		return
	}
	ctx := r.Context()
	tid := domain.TaskID(r.PathValue("id"))
	var req mcPostTaskConvoyReq
	if err := decodeJSONLoose(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if strings.EqualFold(strings.TrimSpace(req.Strategy), "ai") {
		writeError(w, http.StatusNotImplemented, "not_implemented", "convoy strategy \"ai\" (OpenClaw decomposition) is not implemented; use strategy manual or planning with subtasks")
		return
	}
	if _, err := h.Convoy.GetByParentTask(ctx, tid); err == nil {
		writeError(w, http.StatusConflict, "convoy_exists", domain.ErrConvoyExists.Error())
		return
	} else if !errors.Is(err, domain.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	parent, err := h.Task.TaskByIDForAPI(ctx, tid)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	subs := req.Subtasks
	if subs == nil {
		subs = []mcPostTaskSubtask{}
	}
	dsubs, err := mcSubtasksToDomain(subs)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	strat := strings.TrimSpace(req.Strategy)
	if strat == "" {
		strat = "manual"
	}
	nm := strings.TrimSpace(req.Name)
	if nm == "" {
		nm = strings.TrimSpace(parent.Spec)
		if len(nm) > 120 {
			nm = nm[:120] + "…"
		}
		if nm == "" {
			nm = "convoy"
		}
	}
	patch := convoy.MCCompatFields{
		Name:              nm,
		Strategy:          strat,
		Status:            "active",
		DecompositionSpec: strings.TrimSpace(req.DecompositionSpec),
		UpdatedAt:         time.Now().UTC(),
	}
	meta, err := convoy.MergeMCCompatIntoMetadata("{}", patch)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	c, err := h.Convoy.Create(ctx, convoy.CreateInput{
		ParentTaskID:   tid,
		ProductID:      parent.ProductID,
		MetadataJSON:   meta,
		Subtasks:       dsubs,
	})
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	_ = h.Task.SetKanbanStatus(ctx, tid, domain.StatusConvoyActive, "")
	writeJSON(w, http.StatusCreated, convoyToMCMissionControlJSON(c, parent.Spec))
}

func (h *Handlers) patchTaskConvoyMC(w http.ResponseWriter, r *http.Request) {
	if h.Convoy == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "convoy service not available")
		return
	}
	ctx := r.Context()
	tid := domain.TaskID(r.PathValue("id"))
	var req mcPatchTaskConvoyReq
	if err := decodeJSONLoose(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	st := strings.TrimSpace(req.Status)
	if st == "" {
		writeError(w, http.StatusBadRequest, "validation", "status is required")
		return
	}
	c, err := h.Convoy.GetByParentTask(ctx, tid)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "No convoy found for this task")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out, err := h.Convoy.PatchMCCompat(ctx, c.ID, convoy.MCCompatFields{Status: st})
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	parent, err := h.Task.TaskByIDForAPI(ctx, tid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, convoyToMCMissionControlJSON(out, parent.Spec))
}

func (h *Handlers) deleteTaskConvoyMC(w http.ResponseWriter, r *http.Request) {
	if h.Convoy == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "convoy service not available")
		return
	}
	ctx := r.Context()
	tid := domain.TaskID(r.PathValue("id"))
	c, err := h.Convoy.GetByParentTask(ctx, tid)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "No convoy found for this task")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if err := h.Convoy.DeleteConvoy(ctx, c.ID); err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	parent, err := h.Task.TaskByIDForAPI(ctx, tid)
	if err == nil && parent.Status == domain.StatusConvoyActive {
		_ = h.Task.SetKanbanStatus(ctx, tid, domain.StatusInbox, "")
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (h *Handlers) postTaskConvoySubtasksMC(w http.ResponseWriter, r *http.Request) {
	if h.Convoy == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "convoy service not available")
		return
	}
	ctx := r.Context()
	tid := domain.TaskID(r.PathValue("id"))
	var req mcPostTaskConvoySubtasksReq
	if err := decodeJSONLoose(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if len(req.Subtasks) == 0 {
		writeError(w, http.StatusBadRequest, "validation", "subtasks array is required")
		return
	}
	dsubs, err := mcSubtasksToDomain(req.Subtasks)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation", err.Error())
		return
	}
	before, err := h.Convoy.GetByParentTask(ctx, tid)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "No convoy found for this task")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	prevN := len(before.Subtasks)
	out, err := h.Convoy.AppendSubtasks(ctx, tid, dsubs)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	added := out.Subtasks[prevN:]
	created := make([]map[string]any, len(added))
	done := make(map[domain.SubtaskID]bool, len(out.Subtasks))
	for i := range out.Subtasks {
		done[out.Subtasks[i].ID] = out.Subtasks[i].Completed
	}
	topo, terr := convoy.StableTopologicalSubtaskOrder(out.Subtasks)
	if terr != nil {
		topo = nil
		for i := range out.Subtasks {
			topo = append(topo, string(out.Subtasks[i].ID))
		}
		sort.Strings(topo)
	}
	rank := make(map[string]int, len(topo))
	for i, id := range topo {
		rank[id] = i
	}
	for i := range added {
		st := added[i]
		ws := convoySubtaskWorkloadStatus(st, done)
		ts := mcTaskStatusFromWorkload(ws)
		deps := make([]string, len(st.DependsOn))
		for j, d := range st.DependsOn {
			deps[j] = string(d)
		}
		created[i] = map[string]any{
			"id":            string(st.ID),
			"convoy_id":     string(out.ID),
			"task_id":       string(st.ID),
			"sort_order":    rank[string(st.ID)],
			"depends_on":    deps,
			"agent_role":    st.AgentRole,
			"title":         st.Title,
			"metadata_json": st.MetadataJSON,
			"dag_layer":     st.DagLayer,
			"description":   subtaskDescriptionFromMeta(st.MetadataJSON),
			"task": map[string]any{
				"id":                string(st.ID),
				"title":             st.Title,
				"status":            ts,
				"assigned_agent_id": nil,
			},
		}
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handlers) postTaskConvoyDispatchMC(w http.ResponseWriter, r *http.Request) {
	if h.Convoy == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "convoy service not available")
		return
	}
	ctx := r.Context()
	tid := domain.TaskID(r.PathValue("id"))
	var req dispatchReq
	if err := decodeJSONLoose(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	c, err := h.Convoy.GetByParentTask(ctx, tid)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "No convoy found for this task")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	_, _, mcSt, _, _ := convoy.MCCompatFromMetadata(c.MetadataJSON)
	if strings.TrimSpace(mcSt) != "" && !convoy.MCConvoyDispatchAllowed(c.MetadataJSON) {
		writeError(w, http.StatusBadRequest, "bad_request", "Convoy is "+mcSt+", cannot dispatch")
		return
	}
	before := make(map[string]bool, len(c.Subtasks))
	for i := range c.Subtasks {
		before[string(c.Subtasks[i].ID)] = c.Subtasks[i].Dispatched
	}
	n, err := h.Convoy.DispatchReady(ctx, c.ID, req.EstimatedCost)
	if err != nil {
		if mapDomainErr(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	results := make([]map[string]any, 0)
	refreshed, _ := h.Convoy.Get(ctx, c.ID)
	if refreshed != nil {
		for i := range refreshed.Subtasks {
			st := refreshed.Subtasks[i]
			sid := string(st.ID)
			if st.Dispatched && !before[sid] {
				results = append(results, map[string]any{
					"taskId":  sid,
					"success": true,
				})
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"dispatched": n,
		"total":      n,
		"results":    results,
	})
}

func (h *Handlers) getTaskConvoyProgressMC(w http.ResponseWriter, r *http.Request) {
	if h.Convoy == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "convoy service not available")
		return
	}
	tid := domain.TaskID(r.PathValue("id"))
	c, err := h.Convoy.GetByParentTask(r.Context(), tid)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "No convoy found for this task")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	_, _, mcSt, _, _ := convoy.MCCompatFromMetadata(c.MetadataJSON)
	if mcSt == "" {
		mcSt = "active"
	}
	done := make(map[domain.SubtaskID]bool, len(c.Subtasks))
	for i := range c.Subtasks {
		done[c.Subtasks[i].ID] = c.Subtasks[i].Completed
	}
	breakdown := map[string]int{}
	summary := make([]map[string]any, len(c.Subtasks))
	for i := range c.Subtasks {
		st := c.Subtasks[i]
		ts := mcTaskStatusFromWorkload(convoySubtaskWorkloadStatus(st, done))
		breakdown[ts]++
		deps := make([]string, len(st.DependsOn))
		for j, d := range st.DependsOn {
			deps[j] = string(d)
		}
		summary[i] = map[string]any{
			"id":                  string(st.ID),
			"task_id":             string(st.ID),
			"title":               st.Title,
			"status":              ts,
			"assigned_agent_id":   nil,
			"sort_order":          i,
			"depends_on":          deps,
		}
	}
	nCompleted := 0
	for i := range c.Subtasks {
		if c.Subtasks[i].Completed {
			nCompleted++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"convoy_id":          string(c.ID),
		"status":             mcSt,
		"total":              len(c.Subtasks),
		"completed":          nCompleted,
		"failed":             0,
		"failed_subtasks":    0,
		"completed_subtasks": nCompleted,
		"breakdown":          breakdown,
		"subtasks":           summary,
	})
}
