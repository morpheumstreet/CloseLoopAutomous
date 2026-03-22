package domain

import "testing"

func TestConvoySubtaskDagLayers(t *testing.T) {
	a := SubtaskID("a")
	b := SubtaskID("b")
	c := SubtaskID("c")
	subs := []Subtask{
		{ID: a, AgentRole: "x"},
		{ID: b, AgentRole: "y", DependsOn: []SubtaskID{a}},
		{ID: c, AgentRole: "z", DependsOn: []SubtaskID{a, b}},
	}
	layers := ConvoySubtaskDagLayers(subs)
	if layers[a] != 0 || layers[b] != 1 || layers[c] != 2 {
		t.Fatalf("layers: %#v", layers)
	}
}
