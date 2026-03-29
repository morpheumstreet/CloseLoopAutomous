/** Persisted arms connection for static / hosted builds (overrides Vite env when set). */

export const ARMS_LS_BASE_URL = 'ft_arms_base_url';
export const ARMS_LS_API_KEY = 'ft_arms_api_key';

export function hasArmsEndpointConfigured(): boolean {
  if (typeof localStorage === 'undefined') return false;
  const url = localStorage.getItem(ARMS_LS_BASE_URL)?.trim();
  return Boolean(url);
}

export function saveArmsConnection(baseUrl: string, apiKey: string): void {
  localStorage.setItem(ARMS_LS_BASE_URL, baseUrl.trim());
  localStorage.setItem(ARMS_LS_API_KEY, apiKey.trim());
}
