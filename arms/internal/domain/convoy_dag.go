package domain

import "fmt"

// ValidateConvoySubtasks checks that depends_on ids exist and the dependency graph is acyclic.
// An empty slice is valid (Mission Control allows creating a convoy with zero subtasks, then POST …/subtasks).
func ValidateConvoySubtasks(subtasks []Subtask) error {
	if len(subtasks) == 0 {
		return nil
	}
	ids := make(map[SubtaskID]struct{}, len(subtasks))
	for i := range subtasks {
		if subtasks[i].ID == "" {
			return fmt.Errorf("%w: subtask id required", ErrInvalidInput)
		}
		ids[subtasks[i].ID] = struct{}{}
	}
	for i := range subtasks {
		for _, d := range subtasks[i].DependsOn {
			if _, ok := ids[d]; !ok {
				return fmt.Errorf("%w: depends_on references unknown subtask %s", ErrInvalidInput, d)
			}
		}
	}
	if convoyDependsCycle(subtasks) {
		return fmt.Errorf("%w: depends_on contains a cycle", ErrInvalidInput)
	}
	return nil
}

// convoyDependsCycle detects a directed cycle. Edge dep -> st when st lists dep in DependsOn (dep must complete before st).
func convoyDependsCycle(subtasks []Subtask) bool {
	adj := make(map[SubtaskID][]SubtaskID)
	for i := range subtasks {
		st := subtasks[i].ID
		for _, d := range subtasks[i].DependsOn {
			adj[d] = append(adj[d], st)
		}
	}
	visited := make(map[SubtaskID]bool)
	stack := make(map[SubtaskID]bool)
	var dfs func(SubtaskID) bool
	dfs = func(u SubtaskID) bool {
		if stack[u] {
			return true
		}
		if visited[u] {
			return false
		}
		visited[u] = true
		stack[u] = true
		for _, v := range adj[u] {
			if dfs(v) {
				return true
			}
		}
		stack[u] = false
		return false
	}
	for i := range subtasks {
		if dfs(subtasks[i].ID) {
			return true
		}
	}
	return false
}
