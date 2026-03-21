/** Browser env for the arms HTTP + SSE surface (Bun/Vite-style `import.meta.env`). */
export type ArmsEnv = {
  readonly baseUrl: string;
  /** MC_API_TOKEN (Bearer) when set on the server. */
  readonly token: string;
  /** ARMS_ACL HTTP Basic user + password (omit if using token only). */
  readonly basicUser: string;
  readonly basicPassword: string;
};

function trimBase(url: string): string {
  return url.replace(/\/+$/, '');
}

export function readArmsEnv(): ArmsEnv {
  const baseUrl = trimBase(import.meta.env.VITE_ARMS_URL?.trim() || 'http://127.0.0.1:8080');
  const token = import.meta.env.VITE_ARMS_TOKEN?.trim() || '';
  const basicUser = import.meta.env.VITE_ARMS_BASIC_USER?.trim() || '';
  const basicPassword = import.meta.env.VITE_ARMS_BASIC_PASSWORD?.trim() || '';
  return { baseUrl, token, basicUser, basicPassword };
}
