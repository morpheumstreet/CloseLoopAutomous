package httpapi

// RouteEntry documents a public HTTP route (lightweight substitute until OpenAPI).
type RouteEntry struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

func routeCatalog() []RouteEntry {
	return []RouteEntry{
		{"GET", "/api/health", "Liveness check (no auth)"},
		{"GET", "/api/docs/routes", "This route list (JSON)"},
		{"POST", "/api/products", "Register product (optional repo_url, repo_branch, description, program_document, settings_json, icon_url)"},
		{"GET", "/api/products/{id}", "Get product"},
		{"PATCH", "/api/products/{id}", "Patch product metadata (MC-style profile fields)"},
		{"POST", "/api/products/{id}/research", "Run research phase"},
		{"POST", "/api/products/{id}/ideation", "Run ideation phase"},
		{"GET", "/api/products/{id}/ideas", "List ideas for product"},
		{"GET", "/api/products/{id}/maybe-pool", "List ideas in maybe pool for product"},
		{"GET", "/api/products/{id}/tasks", "List tasks for product (Kanban source, newest first)"},
		{"GET", "/api/products/{id}/convoys", "List convoys for product (newest first)"},
		{"POST", "/api/ideas/{id}/swipe", "Submit swipe decision"},
		{"POST", "/api/ideas/{id}/promote-maybe", "Promote a maybe idea to yes (and advance product from swipe when applicable)"},
		{"POST", "/api/tasks", "Create task from approved idea (Kanban planning)"},
		{"GET", "/api/tasks/{id}", "Get task"},
		{"PATCH", "/api/tasks/{id}", "Update Kanban status / status_reason / planning clarifications JSON"},
		{"POST", "/api/tasks/{id}/plan/approve", "Approve plan → inbox (optional body {spec})"},
		{"POST", "/api/tasks/{id}/plan/reject", "Reject / recall plan → planning (inbox or assigned pre-dispatch; optional {status_reason})"},
		{"POST", "/api/tasks/{id}/dispatch", "Dispatch to agent gateway (requires assigned + approved plan)"},
		{"POST", "/api/tasks/{id}/checkpoint", "Record checkpoint"},
		{"POST", "/api/tasks/{id}/complete", "Mark task completed"},
		{"POST", "/api/convoys", "Create convoy"},
		{"GET", "/api/convoys/{id}", "Get convoy by id"},
		{"POST", "/api/convoys/{id}/dispatch-ready", "Dispatch ready convoy subtasks"},
		{"POST", "/api/costs", "Record cost event"},
		{"GET", "/api/agents", "Agent listing (stub)"},
		{"POST", "/api/openclaw/proxy", "Gateway proxy (stub)"},
		{"GET", "/api/workspaces", "Workspaces (stub)"},
		{"GET", "/api/settings", "Settings (stub)"},
		{"POST", "/api/webhooks/agent-completion", "Agent completion (HMAC, not Bearer)"},
		{"GET", "/api/live/events", "SSE activity stream (?token= when auth enabled)"},
	}
}
