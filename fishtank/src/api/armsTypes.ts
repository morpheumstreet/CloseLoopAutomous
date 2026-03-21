/** JSON shapes returned by arms (subset used by Fishtank). */

export type ApiProduct = {
  id: string;
  name: string;
  workspace_id: string;
  stage: string;
  updated_at: string;
  icon_url?: string;
};

export type ApiTask = {
  id: string;
  product_id: string;
  idea_id: string;
  spec: string;
  status: string;
  updated_at: string;
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
