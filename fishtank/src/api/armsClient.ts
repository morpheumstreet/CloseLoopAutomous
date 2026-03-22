import type { ArmsEnv } from '../config/armsEnv';
import type {
  ApiAgentHealthItem,
  ApiOperationLogEntry,
  ApiProduct,
  ApiProductDetail,
  ApiTask,
  ApiVersion,
} from './armsTypes';

export class ArmsHttpError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly code?: string,
  ) {
    super(message);
    this.name = 'ArmsHttpError';
  }
}

function joinUrl(base: string, path: string): string {
  if (path.startsWith('http')) return path;
  const b = base.endsWith('/') ? base.slice(0, -1) : base;
  const p = path.startsWith('/') ? path : `/${path}`;
  return `${b}${p}`;
}

export type CreateProductBody = {
  name: string;
  workspace_id: string;
};

export type OperationsLogQuery = {
  limit?: number;
  product_id?: string;
  action?: string;
  resource_type?: string;
  since?: string;
};

/**
 * HTTP client for arms — auth, JSON, task mutations, and error mapping.
 */
export class ArmsClient {
  constructor(private readonly env: ArmsEnv) {}

  async health(): Promise<{ status: string }> {
    return this.getJson<{ status: string }>('/api/health');
  }

  async version(): Promise<ApiVersion> {
    return this.getJson<ApiVersion>('/api/version');
  }

  async listProducts(): Promise<ApiProduct[]> {
    const body = await this.getJson<{ products: ApiProduct[] }>('/api/products');
    return body.products ?? [];
  }

  async getProduct(id: string): Promise<ApiProductDetail> {
    return this.getJson<ApiProductDetail>(`/api/products/${encodeURIComponent(id)}`);
  }

  async listProductTasks(productId: string): Promise<ApiTask[]> {
    const body = await this.getJson<{ tasks: ApiTask[] }>(
      `/api/products/${encodeURIComponent(productId)}/tasks`,
    );
    return body.tasks ?? [];
  }

  async getTask(taskId: string): Promise<ApiTask> {
    return this.getJson<ApiTask>(`/api/tasks/${encodeURIComponent(taskId)}`);
  }

  async createTask(body: { idea_id: string; spec: string }): Promise<ApiTask> {
    return this.postJson<ApiTask>('/api/tasks', body);
  }

  async patchTask(
    taskId: string,
    body: { status?: string; status_reason?: string; clarifications_json?: string },
  ): Promise<ApiTask> {
    return this.patchJson<ApiTask>(`/api/tasks/${encodeURIComponent(taskId)}`, body);
  }

  async approvePlan(taskId: string, body: Record<string, unknown> = {}): Promise<ApiTask> {
    return this.postJson<ApiTask>(`/api/tasks/${encodeURIComponent(taskId)}/plan/approve`, body);
  }

  async rejectPlan(taskId: string, body: Record<string, unknown> = {}): Promise<ApiTask> {
    return this.postJson<ApiTask>(`/api/tasks/${encodeURIComponent(taskId)}/plan/reject`, body);
  }

  async dispatchTask(taskId: string, estimatedCost = 0): Promise<ApiTask> {
    return this.postJson<ApiTask>(`/api/tasks/${encodeURIComponent(taskId)}/dispatch`, {
      estimated_cost: estimatedCost,
    });
  }

  async completeTask(taskId: string): Promise<ApiTask> {
    return this.postJson<ApiTask>(`/api/tasks/${encodeURIComponent(taskId)}/complete`, {});
  }

  async stallNudge(taskId: string, note?: string): Promise<ApiTask> {
    const body = note?.trim() ? { note: note.trim() } : {};
    return this.postJson<ApiTask>(`/api/tasks/${encodeURIComponent(taskId)}/stall-nudge`, body);
  }

  async openPullRequest(
    taskId: string,
    body: { head_branch: string; title?: string; body?: string },
  ): Promise<{ pr_url: string; pr_number?: number }> {
    return this.postJson<{ pr_url: string; pr_number?: number }>(
      `/api/tasks/${encodeURIComponent(taskId)}/pull-request`,
      body,
    );
  }

  /** Empty list on 503 (agent health not configured). */
  async listStalledTasks(productId: string): Promise<Record<string, unknown>[]> {
    const res = await this.raw(
      'GET',
      `/api/products/${encodeURIComponent(productId)}/stalled-tasks`,
    );
    if (res.status === 503) return [];
    if (!res.ok) {
      const err = await readErrorBody(res);
      throw new ArmsHttpError(err.message, res.status, err.code);
    }
    const data = (await res.json()) as { stalled?: Record<string, unknown>[] };
    return data.stalled ?? [];
  }

  async listOperationsLog(q: OperationsLogQuery = {}): Promise<ApiOperationLogEntry[]> {
    const sp = new URLSearchParams();
    if (q.limit != null) sp.set('limit', String(q.limit));
    if (q.product_id) sp.set('product_id', q.product_id);
    if (q.action) sp.set('action', q.action);
    if (q.resource_type) sp.set('resource_type', q.resource_type);
    if (q.since) sp.set('since', q.since);
    const qs = sp.toString();
    const path = qs ? `/api/operations-log?${qs}` : '/api/operations-log';
    const body = await this.getJson<{ entries?: ApiOperationLogEntry[] }>(path);
    return body.entries ?? [];
  }

  /** Empty when agent health is not configured (503) or on other errors we treat as unavailable. */
  async listProductAgentHealth(productId: string): Promise<ApiAgentHealthItem[]> {
    const res = await this.raw('GET', `/api/products/${encodeURIComponent(productId)}/agent-health`);
    if (res.status === 503) return [];
    if (!res.ok) {
      const err = await readErrorBody(res);
      throw new ArmsHttpError(err.message, res.status, err.code);
    }
    const body = (await res.json()) as { items?: ApiAgentHealthItem[] };
    return body.items ?? [];
  }

  async createProduct(body: CreateProductBody): Promise<ApiProduct> {
    return this.postJson<ApiProduct>('/api/products', body);
  }

  private headers(): HeadersInit {
    const h: Record<string, string> = { Accept: 'application/json' };
    if (this.env.token) {
      h.Authorization = `Bearer ${this.env.token}`;
    } else if (this.env.basicUser) {
      const raw = `${this.env.basicUser}:${this.env.basicPassword}`;
      h.Authorization = `Basic ${btoa(raw)}`;
    }
    return h;
  }

  private async raw(method: string, path: string, body?: unknown): Promise<Response> {
    const init: RequestInit = { method, headers: this.headers() };
    if (body !== undefined) {
      (init.headers as Record<string, string>)['Content-Type'] = 'application/json';
      init.body = JSON.stringify(body);
    }
    return fetch(joinUrl(this.env.baseUrl, path), init);
  }

  private async getJson<T>(path: string): Promise<T> {
    const res = await this.raw('GET', path);
    if (!res.ok) {
      const err = await readErrorBody(res);
      throw new ArmsHttpError(err.message, res.status, err.code);
    }
    return (await res.json()) as T;
  }

  private async postJson<T>(path: string, body: unknown): Promise<T> {
    const res = await this.raw('POST', path, body);
    if (!res.ok) {
      const err = await readErrorBody(res);
      throw new ArmsHttpError(err.message, res.status, err.code);
    }
    return (await res.json()) as T;
  }

  private async patchJson<T>(path: string, body: unknown): Promise<T> {
    const res = await this.raw('PATCH', path, body);
    if (!res.ok) {
      const err = await readErrorBody(res);
      throw new ArmsHttpError(err.message, res.status, err.code);
    }
    return (await res.json()) as T;
  }
}

async function readErrorBody(res: Response): Promise<{ message: string; code?: string }> {
  try {
    const j = (await res.json()) as { error?: string; code?: string };
    if (j?.error && typeof j.error === 'string') return { message: j.error, code: j.code };
  } catch {
    /* ignore */
  }
  return { message: res.statusText || 'request failed' };
}

export function buildLiveEventsUrl(env: ArmsEnv, productId: string): string {
  const u = new URL(joinUrl(env.baseUrl, '/api/live/events'));
  u.searchParams.set('product_id', productId);
  if (env.token) {
    u.searchParams.set('token', env.token);
  } else if (env.basicUser) {
    u.searchParams.set('basic', btoa(`${env.basicUser}:${env.basicPassword}`));
  }
  return u.toString();
}

/** Template URL for docs (replace `product_id`). */
export function buildLiveEventsUrlTemplate(env: ArmsEnv): string {
  const u = new URL(joinUrl(env.baseUrl, '/api/live/events'));
  u.searchParams.set('product_id', '<product_id>');
  if (env.token) {
    u.searchParams.set('token', env.token);
  } else if (env.basicUser) {
    u.searchParams.set('basic', btoa(`${env.basicUser}:${env.basicPassword}`));
  }
  return u.toString();
}
