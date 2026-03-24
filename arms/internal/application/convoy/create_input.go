package convoy

import (
	"encoding/json"
	"strings"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// CreateInput is the payload for Service.Create (convoy + DAG subtasks).
type CreateInput struct {
	ParentTaskID   domain.TaskID
	ProductID      domain.ProductID
	MetadataJSON   string
	Subtasks       []domain.Subtask
}

func normalizeMetadataJSON(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "{}", nil
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return "", err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func applyDagLayers(subtasks []domain.Subtask) {
	layers := domain.ConvoySubtaskDagLayers(subtasks)
	for i := range subtasks {
		subtasks[i].DagLayer = layers[subtasks[i].ID]
	}
}
