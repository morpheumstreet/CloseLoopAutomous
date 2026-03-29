import { ARMS_LS_API_KEY, ARMS_LS_BASE_URL } from './armsLocalStorage';

/** Browser env for the arms HTTP + SSE surface (Bun/Vite-style `import.meta.env`). */
export type ArmsEnv = {
  readonly baseUrl: string;
  /** MC_API_TOKEN (Bearer) when set on the server. */
  readonly token: string;
  /** ARMS_ACL HTTP Basic user + password (omit if using token only). */
  readonly basicUser: string;
  readonly basicPassword: string;
};

export function trimBase(url: string): string {
  return url.replace(/\/+$/, '');
}

/** Bun --hot can expose `import.meta.env` as undefined during HMR; never read properties on it directly. */
function viteEnv(): ImportMetaEnv {
  const env = (import.meta as ImportMeta & { env?: ImportMetaEnv }).env;
  return env ?? ({} as ImportMetaEnv);
}

export function readArmsEnv(): ArmsEnv {
  const e = viteEnv();
  const baseUrl = trimBase(e.VITE_ARMS_URL?.trim() || 'http://localhost:8080');
  const token = e.VITE_ARMS_TOKEN?.trim() || '';
  const basicUser = e.VITE_ARMS_BASIC_USER?.trim() || '';
  const basicPassword = e.VITE_ARMS_BASIC_PASSWORD?.trim() || '';
  return { baseUrl, token, basicUser, basicPassword };
}

/**
 * Vite env merged with optional `localStorage` overrides (see `armsLocalStorage`).
 * When `ft_arms_base_url` is set, base URL and bearer token come from storage; Basic fields stay from Vite.
 */
export function resolveArmsEnv(): ArmsEnv {
  const base = readArmsEnv();
  if (typeof localStorage === 'undefined') return base;
  const lsUrl = localStorage.getItem(ARMS_LS_BASE_URL)?.trim();
  if (!lsUrl) return base;
  const lsKey = localStorage.getItem(ARMS_LS_API_KEY);
  const token = lsKey !== null ? lsKey.trim() : base.token;
  return {
    baseUrl: trimBase(lsUrl),
    token,
    basicUser: base.basicUser,
    basicPassword: base.basicPassword,
  };
}

/** True only in dev / HMR; safe when `import.meta.env` is missing (Bun `bun build` in the browser). */
export function isDevBuild(): boolean {
  return viteEnv().DEV === true;
}
