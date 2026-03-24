package httpapi

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	convoyapp "github.com/morpheumstreet/CloseLoopAutomous/arms/internal/application/convoy"
	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

func subtaskDescriptionFromMeta(metadataJSON string) string {
	s := strings.TrimSpace(metadataJSON)
	if s == "" || s == "{}" {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return ""
	}
	if v, ok := m["description"].(string); ok {
		return v
	}
	return ""
}

func mcTaskStatusFromWorkload(workload string) string {
	switch workload {
	case "completed":
		return "done"
	case "running":
		return "in_progress"
	default:
		return "inbox"
	}
}

func mcEffectiveUpdatedAt(c *domain.Convoy, mcUpdatedAt string) string {
	if strings.TrimSpace(mcUpdatedAt) != "" {
		return mcUpdatedAt
	}
	return c.CreatedAt.UTC().Format(time.RFC3339Nano)
}

// convoyToMCMissionControlJSON maps an ARMS convoy to Mission Control–style task-scoped convoy JSON
// (nested task per subtask, counters, strategy/name/status from mc_compat metadata).
func convoyToMCMissionControlJSON(c *domain.Convoy, parentTaskSpec string) map[string]any {
	done := make(map[domain.SubtaskID]bool, len(c.Subtasks))
	for i := range c.Subtasks {
		done[c.Subtasks[i].ID] = c.Subtasks[i].Completed
	}
	topo, err := convoyapp.StableTopologicalSubtaskOrder(c.Subtasks)
	if err != nil {
		topo = nil
		for i := range c.Subtasks {
			topo = append(topo, string(c.Subtasks[i].ID))
		}
		sort.Strings(topo)
	}
	rank := make(map[string]int, len(topo))
	for i, id := range topo {
		rank[id] = i
	}

	name, strategy, mcStatus, decomp, mcUpdated := convoyapp.MCCompatFromMetadata(c.MetadataJSON)
	if name == "" {
		name = strings.TrimSpace(parentTaskSpec)
		if len(name) > 120 {
			name = name[:120] + "…"
		}
		if name == "" {
			name = "convoy"
		}
	}
	if strategy == "" {
		strategy = "manual"
	}
	if mcStatus == "" {
		mcStatus = "active"
	}
	effStatus := mcStatus
	if mcStatus == "active" && len(c.Subtasks) > 0 {
		allDone := true
		for i := range c.Subtasks {
			if !c.Subtasks[i].Completed {
				allDone = false
				break
			}
		}
		if allDone {
			effStatus = "done"
		}
	}

	nCompleted := 0
	subsOut := make([]map[string]any, len(c.Subtasks))
	for i := range c.Subtasks {
		st := c.Subtasks[i]
		ws := convoySubtaskWorkloadStatus(st, done)
		ts := mcTaskStatusFromWorkload(ws)
		if st.Completed {
			nCompleted++
		}
		deps := make([]string, len(st.DependsOn))
		for j, d := range st.DependsOn {
			deps[j] = string(d)
		}
		subsOut[i] = map[string]any{
			"id":            string(st.ID),
			"convoy_id":     string(c.ID),
			"task_id":       string(st.ID),
			"sort_order":    rank[string(st.ID)],
			"depends_on":    deps,
			"agent_role":    st.AgentRole,
			"title":         st.Title,
			"metadata_json": st.MetadataJSON,
			"dag_layer":     st.DagLayer,
			"dispatched":    st.Dispatched,
			"completed":     st.Completed,
			"description":   subtaskDescriptionFromMeta(st.MetadataJSON),
			"task": map[string]any{
				"id":                string(st.ID),
				"title":             st.Title,
				"status":            ts,
				"assigned_agent_id": nil,
			},
		}
	}

	rawEdges := convoyapp.SubtaskDependencyEdges(c.Subtasks)
	edgeObjs := make([]map[string]any, len(rawEdges))
	for i := range rawEdges {
		edgeObjs[i] = map[string]any{"from": rawEdges[i].From, "to": rawEdges[i].To}
	}
	edgeCount := 0
	maxLayer := 0
	for i := range c.Subtasks {
		edgeCount += len(c.Subtasks[i].DependsOn)
		if c.Subtasks[i].DagLayer > maxLayer {
			maxLayer = c.Subtasks[i].DagLayer
		}
	}

	return map[string]any{
		"id":                     string(c.ID),
		"parent_task_id":         string(c.ParentID),
		"product_id":             string(c.ProductID),
		"name":                   name,
		"strategy":               strategy,
		"decomposition_strategy": strategy,
		"decomposition_spec":     decomp,
		"status":                 effStatus,
		"total_subtasks":         len(c.Subtasks),
		"completed_subtasks":     nCompleted,
		"failed_subtasks":        0,
		"created_at":             c.CreatedAt.UTC().Format(time.RFC3339Nano),
		"updated_at":             mcEffectiveUpdatedAt(c, mcUpdated),
		"subtasks":               subsOut,
		"metadata_json":          c.MetadataJSON,
		"edges":                  edgeObjs,
		"graph": map[string]any{
			"node_count": len(c.Subtasks),
			"edge_count": edgeCount,
			"max_depth":  maxLayer,
		},
	}
}
