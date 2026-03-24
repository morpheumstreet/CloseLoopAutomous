package chromemstore

import (
	"fmt"
	"os"
	"strings"

	"github.com/philippgille/chromem-go"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/config"
)

// NewEmbeddingFunc returns a chromem EmbeddingFunc from config (Ollama or OpenAI-compatible).
func NewEmbeddingFunc(cfg config.Config) (chromem.EmbeddingFunc, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.ChromemEmbedder)) {
	case "openai":
		key := strings.TrimSpace(cfg.ChromemOpenAIAPIKey)
		if key == "" {
			key = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
		}
		if key == "" {
			return nil, fmt.Errorf("chromem openai embedder: set ARMS_CHROMEM_OPENAI_API_KEY or OPENAI_API_KEY")
		}
		model := strings.TrimSpace(cfg.ChromemOpenAIModel)
		if model == "" {
			model = string(chromem.EmbeddingModelOpenAI3Small)
		}
		return chromem.NewEmbeddingFuncOpenAI(key, chromem.EmbeddingModelOpenAI(model)), nil
	default:
		// ollama (default)
		model := strings.TrimSpace(cfg.ChromemEmbedderModel)
		if model == "" {
			model = "nomic-embed-text"
		}
		base := strings.TrimSpace(cfg.ChromemOllamaBaseURL)
		return chromem.NewEmbeddingFuncOllama(model, base), nil
	}
}
