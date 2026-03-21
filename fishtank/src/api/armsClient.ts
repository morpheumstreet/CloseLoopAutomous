import type { ArmsEnv } from '../config/armsEnv';
import type { ApiAgentHealthItem, ApiProduct, ApiTask } from './armsTypes';

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

/**
 * Thin HTTP client for arms — single place for auth, JSON, and error mapping (DRY).
 */
export class ArmsClient {
  constructor(private readonly env: ArmsEnv) {}

  async health(): Promise<{ status: string }> {
    return this.getJson<{ status: string }>('/api/health');
  }

  async listProducts(): Promise<ApiProduct[]> {
    const body = await this.getJson<{ products: ApiProduct[] }>('/api/products');
    return body.products ?? [];
  }

  async listProductTasks(productId: string): Promise<ApiTask[]> {
    const body = await this.getJson<{ tasks: ApiTask[] }>(
      `/api/products/${encodeURIComponent(productId)}/tasks`,
    );
    return body.tasks ?? [];
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
