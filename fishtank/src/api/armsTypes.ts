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
  /** Optional team mission (PATCH to set or clear). */
  mission_statement?: string;
  /** Optional team vision (PATCH to set or clear). */
  vision_statement?: string;
};

/** Body for `PATCH /api/products/{id}` — include only fields to change. */
export type PatchProductBody = {
  name?: string;
  repo_url?: string;
  repo_clone_path?: string;
  repo_branch?: string;
  description?: string;
  program_document?: string;
  mission_statement?: string;
  vision_statement?: string;
  settings_json?: string;
  icon_url?: string;
  merge_policy_json?: string;
  research_cadence_sec?: number;
  ideation_cadence_sec?: number;
  automation_tier?: string;
  auto_dispatch_enabled?: boolean;
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

/** GET /api/ops/host-metrics — machine running arms (no auth). */
export type ApiHostMetricsLoadAvg = {
  load1: number;
  load5: number;
  load15: number;
};

export type ApiHostMetricsCpu = {
  logical_cores: number;
  physical_cores: number;
  percent_total: number;
  sample_interval: string;
  load_avg?: ApiHostMetricsLoadAvg;
};

export type ApiHostMetricsMemory = {
  total_bytes: number;
  available_bytes: number;
  used_bytes: number;
  used_percent: number;
};

export type ApiHostMetricsDisk = {
  path: string;
  total_bytes: number;
  free_bytes: number;
  used_bytes: number;
  used_percent: number;
  inodes_total: number;
  inodes_used: number;
  inodes_free: number;
  inodes_percent: number;
};

export type ApiHostMetrics = {
  cpu: ApiHostMetricsCpu;
  memory: ApiHostMetricsMemory;
  disk: ApiHostMetricsDisk;
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
  id?: number | string;
  actor?: string;
  action?: string;
  resource_type?: string;
  resource_id?: string;
  detail_json?: string;
  product_id?: string;
  created_at?: string;
};

/** GET /api/products/{id}/research-cycles — append-only research run snapshots. */
export type ApiResearchCycle = {
  id: string;
  product_id: string;
  summary_snapshot: string;
  created_at: string;
};

/** Product knowledge entry — `GET/POST/PATCH …/products/{id}/knowledge`. */
export type ApiKnowledgeEntry = {
  id: number;
  product_id: string;
  content: string;
  created_at: string;
  updated_at: string;
  task_id?: string;
  metadata: Record<string, unknown>;
};

/** `POST /api/products/{id}/nlp/tfidf-suggest-tags` — tag salience; category for Docs is inferred in the client. */
export type ApiTfidfTagScore = { token: string; score: number };

export type ApiTfidfSuggestTagsResponse = {
  tags: ApiTfidfTagScore[];
  method: string;
  corpus_documents: number;
  product_id?: string;
  idea_id?: string;
};

/** GET/PATCH `/api/products/{id}/product-schedule` — autopilot cadence for this product. */
export type ApiProductSchedule = {
  product_id: string;
  enabled: boolean;
  spec_json: string;
  cron_expr?: string;
  delay_seconds?: number;
  asynq_task_id?: string;
  last_enqueued_at?: string;
  next_scheduled_at?: string;
  updated_at?: string;
};

export type PatchProductScheduleBody = {
  enabled?: boolean;
  spec_json?: string;
  cron_expr?: string;
  delay_seconds?: number;
};

/** `GET/POST …/products/{id}/feedback` and `PATCH /api/product-feedback/{id}`. */
export type ApiProductFeedback = {
  id: string;
  product_id: string;
  source: string;
  content: string;
  customer_id?: string;
  category?: string;
  sentiment?: string;
  processed: boolean;
  idea_id?: string;
  created_at: string;
};

/** `GET /api/products/{id}/convoys` — convoy DAG + subtask workload (Mission Control). */
export type ApiConvoySubtask = {
  id: string;
  agent_role: string;
  title: string;
  metadata_json?: string;
  dag_layer: number;
  depends_on: string[];
  /** Server-derived: completed | running | blocked | ready */
  status: string;
  dispatched: boolean;
  completed: boolean;
  external_ref?: string;
  last_checkpoint?: string;
  dispatch_attempts: number;
};

export type ApiConvoyGraphSummary = {
  node_count: number;
  edge_count: number;
  max_depth: number;
};

export type ApiConvoy = {
  id: string;
  product_id: string;
  parent_id: string;
  metadata_json: string;
  graph: ApiConvoyGraphSummary;
  edges: { from: string; to: string }[];
  subtasks: ApiConvoySubtask[];
  created_at: string;
};

/** `POST /api/tasks/{id}/convoy/dispatch` */
export type ApiConvoyDispatchWaveResponse = {
  dispatched: number;
  total: number;
  results?: Array<{ taskId?: string; success?: boolean }>;
};

/** Row from `GET …/maybe-pool` / batch-reeval (idea JSON + optional maybe_* fields). */
export type ApiMaybePoolIdea = Record<string, unknown> & {
  id?: string;
  title?: string;
  status?: string;
  maybe_next_evaluate_at?: string;
  maybe_last_evaluated_at?: string;
  maybe_evaluation_count?: number;
};
