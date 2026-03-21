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
		{"POST", "/api/products", "Register product (optional repo_url, repo_clone_path, repo_branch, description, program_document, settings_json, icon_url)"},
		{"GET", "/api/products", "List all products (newest registration order is store-defined; typically useful for dashboards)"},
		{"GET", "/api/products/{id}", "Get product"},
		{"PATCH", "/api/products/{id}", "Patch product metadata (MC-style profile fields)"},
		{"POST", "/api/products/{id}/research", "Run research phase"},
		{"POST", "/api/products/{id}/ideation", "Run ideation phase"},
		{"GET", "/api/products/{id}/ideas", "List ideas for product"},
		{"GET", "/api/products/{id}/maybe-pool", "List ideas in maybe pool for product"},
		{"GET", "/api/products/{id}/swipe-history", "Swipe audit log for product (?limit= default 100, max 500)"},
		{"GET", "/api/products/{id}/tasks", "List tasks for product (Kanban source, newest first)"},
		{"GET", "/api/products/{id}/costs/breakdown", "Cost breakdown (?from=&to= RFC3339) + by_agent / by_model"},
		{"PATCH", "/api/products/{id}/cost-caps", "Patch daily/monthly/cumulative caps (negative value clears a limit)"},
		{"GET", "/api/products/{id}/convoys", "List convoys for product (newest first)"},
		{"GET", "/api/products/{id}/merge-queue", "List pending merge-queue rows for product (FIFO by id; ?limit= default 50, max 500)"},
		{"GET", "/api/products/{id}/agent-health", "List task agent heartbeats for product (?limit= default 100, max 500)"},
		{"GET", "/api/products/{id}/stalled-tasks", "Tasks expecting agent heartbeats with none or stale heartbeat (?stale_sec= overrides default)"},
		{"POST", "/api/ideas/{id}/swipe", "Submit swipe decision"},
		{"POST", "/api/ideas/{id}/promote-maybe", "Promote a maybe idea to yes (and advance product from swipe when applicable)"},
		{"POST", "/api/tasks", "Create task from approved idea (Kanban planning)"},
		{"GET", "/api/tasks/{id}", "Get task"},
		{"PATCH", "/api/tasks/{id}", "Update Kanban status / status_reason / planning clarifications JSON"},
		{"POST", "/api/tasks/{id}/plan/approve", "Approve plan → inbox (optional body {spec})"},
		{"POST", "/api/tasks/{id}/plan/reject", "Reject / recall plan → planning (inbox or assigned pre-dispatch; optional {status_reason})"},
		{"POST", "/api/tasks/{id}/dispatch", "Dispatch to agent gateway (requires assigned + approved plan)"},
		{"POST", "/api/tasks/{id}/merge-queue", "Enqueue task for serialized merge (409 if already pending)"},
		{"POST", "/api/tasks/{id}/merge-queue/complete", "Mark pending merge-queue row done for this task (404 if none pending; 409 if not FIFO head)"},
		{"POST", "/api/tasks/{id}/workspace/git-worktree", "Optional git worktree add (ARMS_ENABLE_GIT_WORKTREES=1, ARMS_WORKSPACE_ROOT, product.repo_clone_path); body {branch}"},
		{"GET", "/api/tasks/{id}/agent-health", "Agent heartbeat row for task (unknown if never reported)"},
		{"PATCH", "/api/tasks/{id}/agent-health", "Record heartbeat {status, detail?}"},
		{"POST", "/api/tasks/{id}/pull-request", "Open GitHub PR (head_branch, optional title/body; product.repo_url; ARMS_GITHUB_TOKEN or ARMS_GITHUB_PR_BACKEND=gh + gh CLI)"},
		{"GET", "/api/tasks/{id}/checkpoints", "List checkpoint history (?limit=)"},
		{"POST", "/api/tasks/{id}/checkpoint/restore", "Restore checkpoint from history {history_id}"},
		{"POST", "/api/tasks/{id}/checkpoint", "Record checkpoint"},
		{"POST", "/api/tasks/{id}/complete", "Mark task completed"},
		{"POST", "/api/tasks/{id}/stall-nudge", "Operator stall nudge (optional {note}); execution statuses only; SSE task_stall_nudged"},
		{"POST", "/api/convoys", "Create convoy"},
		{"GET", "/api/convoys/{id}", "Get convoy by id"},
		{"POST", "/api/convoys/{id}/dispatch-ready", "Dispatch ready convoy subtasks"},
		{"POST", "/api/convoy", "MC alias: same as POST /api/convoys"},
		{"GET", "/api/convoy/{id}", "MC alias: same as GET /api/convoys/{id}"},
		{"POST", "/api/convoy/{id}/dispatch-ready", "MC alias: same as POST /api/convoys/{id}/dispatch-ready"},
		{"POST", "/api/costs", "Record cost event"},
		{"GET", "/api/agents", "Recent task agent heartbeats (stub:true when persistence disabled)"},
		{"POST", "/api/openclaw/proxy", "Gateway proxy (stub)"},
		{"GET", "/api/workspaces", "Workspace snapshot: allocated ports + merge_queue_pending"},
		{"GET", "/api/settings", "Settings (stub)"},
		{"POST", "/api/webhooks/agent-completion", "Agent completion (HMAC, not Bearer)"},
		{"GET", "/api/live/events", "SSE activity stream (Bearer or ?token= when MC_API_TOKEN; ?basic= for ARMS_ACL; ?product_id= filters)"},
	}
}
