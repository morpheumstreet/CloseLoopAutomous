/** JSON shapes returned by arms (subset used by Fishtank). */

export type ApiProduct = {
  id: string;
  name: string;
  workspace_id: string;
  stage: string;
  updated_at: string;
  icon_url?: string;
};

/** GET /api/products/{id} — extra fields when returned by the API. */
export type ApiProductDetail = ApiProduct & {
  merge_queue_pending?: number;
  merge_policy?: { merge_method?: string; merge_backend_override?: string };
  merge_policy_json?: string;
  description?: string;
  repo_url?: string;
  repo_branch?: string;
  automation_tier?: string;
  program_document?: string;
};

export type ApiTask = {
  id: string;
  product_id: string;
  idea_id: string;
  spec: string;
  status: string;
  updated_at: string;
  status_reason?: string;
  plan_approved?: boolean;
  clarifications_json?: string;
  checkpoint?: string;
  external_ref?: string;
  sandbox_path?: string;
  worktree_path?: string;
  pull_request_url?: string;
  pull_request_number?: number;
  pull_request_head_branch?: string;
  current_execution_agent_id?: string;
  created_at?: string;
};

/** GET /api/version (no auth). */
export type ApiVersion = {
  version: string;
  tag: string;
  number: string;
  commits_after_tag: number;
  commit: string;
  dirty: boolean;
};

export type ApiAgentHealthItem = {
  task_id: string;
  product_id: string;
  status: string;
  heartbeat_stale: boolean;
  last_heartbeat_at: string;
  detail: Record<string, unknown>;
};

export type ArmsSsePayload = {
  event?: string;
  type?: string;
  ts?: string;
  product_id?: string;
  task_id?: string;
  data?: Record<string, unknown>;
};

export type ApiOperationLogEntry = {
  id?: string;
  actor?: string;
  action?: string;
  resource_type?: string;
  resource_id?: string;
  detail_json?: string;
  product_id?: string;
  created_at?: string;
};
