package domain

// ConvoySubtaskDagLayers assigns each subtask a non-negative depth: roots (no depends_on) are 0;
// otherwise max(dep layer)+1. Requires ValidateConvoySubtasks to have been satisfied (acyclic, ids consistent).
func ConvoySubtaskDagLayers(subtasks []Subtask) map[SubtaskID]int {
	idToIdx := make(map[SubtaskID]int, len(subtasks))
	for i := range subtasks {
		idToIdx[subtasks[i].ID] = i
	}
	memo := make(map[SubtaskID]int)
	var dfs func(SubtaskID) int
	dfs = func(id SubtaskID) int {
		if v, ok := memo[id]; ok {
			return v
		}
		idx, ok := idToIdx[id]
		if !ok {
			memo[id] = 0
			return 0
		}
		st := subtasks[idx]
		if len(st.DependsOn) == 0 {
			memo[id] = 0
			return 0
		}
		max := -1
		for _, d := range st.DependsOn {
			v := dfs(d)
			if v > max {
				max = v
			}
		}
		memo[id] = max + 1
		return memo[id]
	}
	out := make(map[SubtaskID]int, len(subtasks))
	for i := range subtasks {
		out[subtasks[i].ID] = dfs(subtasks[i].ID)
	}
	return out
}
