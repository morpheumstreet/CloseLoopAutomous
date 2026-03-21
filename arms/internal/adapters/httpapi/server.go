package httpapi

import (
	"log/slog"
	"net/http"
)

// NewRouter returns the full HTTP handler tree: public routes, webhook (HMAC), SSE (query token), and Bearer-protected API.
func NewRouter(cfg Config, h *Handlers) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /api/health", http.HandlerFunc(h.health))
	mux.Handle("GET /api/docs/routes", http.HandlerFunc(h.routesDoc))
	mux.Handle("POST /api/webhooks/agent-completion", http.HandlerFunc(h.agentCompletionWebhook))
	mux.Handle("GET /api/live/events", SSEQueryToken(cfg, http.HandlerFunc(h.liveSSE)))

	sub := http.NewServeMux()
	sub.Handle("POST /api/products", http.HandlerFunc(h.createProduct))
	sub.Handle("GET /api/products/{id}", http.HandlerFunc(h.getProduct))
	sub.Handle("PATCH /api/products/{id}", http.HandlerFunc(h.patchProduct))
	sub.Handle("POST /api/products/{id}/research", http.HandlerFunc(h.runResearch))
	sub.Handle("POST /api/products/{id}/ideation", http.HandlerFunc(h.runIdeation))
	sub.Handle("GET /api/products/{id}/ideas", http.HandlerFunc(h.listIdeas))
	sub.Handle("GET /api/products/{id}/maybe-pool", http.HandlerFunc(h.listMaybePool))
	sub.Handle("GET /api/products/{id}/tasks", http.HandlerFunc(h.listProductTasks))
	sub.Handle("GET /api/products/{id}/convoys", http.HandlerFunc(h.listProductConvoys))
	sub.Handle("POST /api/ideas/{id}/swipe", http.HandlerFunc(h.swipe))
	sub.Handle("POST /api/ideas/{id}/promote-maybe", http.HandlerFunc(h.promoteMaybe))
	sub.Handle("POST /api/tasks", http.HandlerFunc(h.createTask))
	sub.Handle("GET /api/tasks/{id}", http.HandlerFunc(h.getTask))
	sub.Handle("PATCH /api/tasks/{id}", http.HandlerFunc(h.patchTask))
	sub.Handle("POST /api/tasks/{id}/plan/approve", http.HandlerFunc(h.approvePlan))
	sub.Handle("POST /api/tasks/{id}/plan/reject", http.HandlerFunc(h.rejectPlan))
	sub.Handle("POST /api/tasks/{id}/dispatch", http.HandlerFunc(h.dispatchTask))
	sub.Handle("POST /api/tasks/{id}/checkpoint", http.HandlerFunc(h.checkpoint))
	sub.Handle("POST /api/tasks/{id}/complete", http.HandlerFunc(h.completeTask))
	sub.Handle("POST /api/convoys", http.HandlerFunc(h.createConvoy))
	sub.Handle("GET /api/convoys/{id}", http.HandlerFunc(h.getConvoy))
	sub.Handle("POST /api/convoys/{id}/dispatch-ready", http.HandlerFunc(h.dispatchConvoy))
	sub.Handle("POST /api/costs", http.HandlerFunc(h.recordCost))
	sub.Handle("GET /api/agents", http.HandlerFunc(h.agentsStub))
	sub.Handle("POST /api/openclaw/proxy", http.HandlerFunc(h.openclawStub))
	sub.Handle("GET /api/workspaces", http.HandlerFunc(h.workspacesStub))
	sub.Handle("GET /api/settings", http.HandlerFunc(h.settingsStub))

	mux.Handle("/", AuthMiddleware(cfg, sub))

	var handler http.Handler = mux
	if cfg.AccessLog {
		handler = AccessLogMiddleware(slog.Default(), handler)
	}
	handler = RequestIDMiddleware(handler)
	return handler
}
