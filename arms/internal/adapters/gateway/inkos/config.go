package inkos

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

// Options configures the InkOS CLI subprocess client ([Narcooo/inkos]).
//
// Field mapping from gateway_endpoints matches other CLI-style drivers:
//   - InkOSBin ← gateway_token (optional; default "inkos" on PATH)
//   - Workspace ← gateway_url (optional project root; sets process working directory)
//   - BookID ← device_id (InkOS book id for `write next <id>`)
//
// [Narcooo/inkos]: https://github.com/Narcooo/inkos
type Options struct {
	InkOSBin  string
	Workspace string
	BookID    string
	Timeout   time.Duration
	// KnowledgeForDispatch appends ranked snippets to the --context payload when non-nil (same hook as OpenClaw / nanobot).
	KnowledgeForDispatch func(context.Context, domain.ProductID, string) (string, error)
}

// ExpandPath expands ~ and environment variables in a filesystem path.
func ExpandPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = os.ExpandEnv(p)
	if strings.HasPrefix(p, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			rest := strings.TrimPrefix(p[2:], "/")
			p = filepath.Join(h, rest)
		}
	}
	return p
}
