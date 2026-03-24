package convoy

import (
	"errors"
	"fmt"
	"sort"

	"github.com/dominikbraun/graph"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// SubtaskDependencyGraph builds a directed graph with edge dep → st for each entry in st.DependsOn
// (dependency must finish before dependent). Duplicate edges are ignored.
func SubtaskDependencyGraph(subtasks []domain.Subtask) (graph.Graph[string, string], error) {
	g := graph.New(graph.StringHash, graph.Directed())
	for i := range subtasks {
		id := string(subtasks[i].ID)
		if err := g.AddVertex(id); err != nil {
			return nil, fmt.Errorf("vertex %s: %w", id, err)
		}
	}
	for i := range subtasks {
		sid := string(subtasks[i].ID)
		for _, d := range subtasks[i].DependsOn {
			if err := g.AddEdge(string(d), sid); err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
				return nil, fmt.Errorf("edge %s -> %s: %w", d, sid, err)
			}
		}
	}
	return g, nil
}

// StableTopologicalSubtaskOrder returns subtask ids in dependency order (dependencies before dependents).
// Ordering among mutually non-dependent nodes is lexicographic by id (stable, MC-friendly).
func StableTopologicalSubtaskOrder(subtasks []domain.Subtask) ([]string, error) {
	g, err := SubtaskDependencyGraph(subtasks)
	if err != nil {
		return nil, err
	}
	return graph.StableTopologicalSort(g, func(a, b string) bool { return a < b })
}

// SubtaskDependencyEdges returns directed edges dep → dependent from DependsOn fields.
func SubtaskDependencyEdges(subtasks []domain.Subtask) []GraphEdge {
	var out []GraphEdge
	seen := make(map[string]struct{})
	for i := range subtasks {
		sid := string(subtasks[i].ID)
		for _, d := range subtasks[i].DependsOn {
			key := string(d) + "\x00" + sid
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, GraphEdge{From: string(d), To: sid})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		return out[i].To < out[j].To
	})
	return out
}

// SubtaskLayers groups subtask ids by DagLayer (ascending layer, ids sorted lexicographically within each layer).
func SubtaskLayers(subtasks []domain.Subtask) []GraphLayer {
	by := make(map[int][]string)
	maxL := 0
	for i := range subtasks {
		L := subtasks[i].DagLayer
		if L > maxL {
			maxL = L
		}
		id := string(subtasks[i].ID)
		by[L] = append(by[L], id)
	}
	var layers []int
	for L := range by {
		layers = append(layers, L)
	}
	sort.Ints(layers)
	out := make([]GraphLayer, 0, len(layers))
	for _, L := range layers {
		ids := append([]string(nil), by[L]...)
		sort.Strings(ids)
		out = append(out, GraphLayer{Layer: L, SubtaskIDs: ids})
	}
	return out
}
