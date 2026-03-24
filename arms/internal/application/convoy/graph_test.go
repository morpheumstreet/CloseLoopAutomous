package convoy

import (
	"testing"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

func TestStableTopologicalSubtaskOrder_chain(t *testing.T) {
	subs := []domain.Subtask{
		{ID: "a", DependsOn: nil, DagLayer: 0},
		{ID: "b", DependsOn: []domain.SubtaskID{"a"}, DagLayer: 1},
		{ID: "c", DependsOn: []domain.SubtaskID{"b"}, DagLayer: 2},
	}
	order, err := StableTopologicalSubtaskOrder(subs)
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 3 || order[0] != "a" || order[1] != "b" || order[2] != "c" {
		t.Fatalf("order=%v want a,b,c", order)
	}
}

func TestStableTopologicalSubtaskOrder_parallelRoots(t *testing.T) {
	subs := []domain.Subtask{
		{ID: "z", DependsOn: nil},
		{ID: "m", DependsOn: nil},
		{ID: "c", DependsOn: []domain.SubtaskID{"m", "z"}},
	}
	order, err := StableTopologicalSubtaskOrder(subs)
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 3 {
		t.Fatalf("len=%d", len(order))
	}
	// m < z lexicographically → m before z among roots
	if order[0] != "m" || order[1] != "z" || order[2] != "c" {
		t.Fatalf("order=%v want m,z,c", order)
	}
}

func TestStableTopologicalSubtaskOrder_duplicateDependsIgnored(t *testing.T) {
	subs := []domain.Subtask{
		{ID: "a"},
		{ID: "b", DependsOn: []domain.SubtaskID{"a", "a"}},
	}
	_, err := StableTopologicalSubtaskOrder(subs)
	if err != nil {
		t.Fatal(err)
	}
	edges := SubtaskDependencyEdges(subs)
	if len(edges) != 1 {
		t.Fatalf("edges=%v", edges)
	}
}

func TestSubtaskDependencyEdges_sorted(t *testing.T) {
	subs := []domain.Subtask{
		{ID: "c", DependsOn: []domain.SubtaskID{"a"}},
		{ID: "b", DependsOn: []domain.SubtaskID{"a"}},
	}
	e := SubtaskDependencyEdges(subs)
	if len(e) != 2 || e[0].From != "a" || e[0].To != "b" || e[1].To != "c" {
		t.Fatalf("edges=%v", e)
	}
}
