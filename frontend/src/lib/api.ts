/* 纯 HTTP 传输层：统一封装 fetch、认证头注入、错误解析与 401 处理 */

import { clearSession, getAuthToken, markAuthExpired, markAuthRedirecting } from '@/lib/session';

type ApiErrorResponse = {
  error?: string;
  message?: string;
};

export type ApiOptions = RequestInit & {
  auth?: boolean;
  redirectOn401?: boolean;
};

/* ApiError 携带后端 HTTP 状态码，供调用方按状态码区分提示文案 */
export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

export function isApiStatus(error: unknown, status: number) {
  return error instanceof ApiError && error.status === status;
}

export async function readApiError(response: Response, fallback = `请求失败：${response.status}`) {
  try {
    const body = (await response.json()) as ApiErrorResponse;
    return body.message || body.error || fallback;
  } catch {
    return fallback;
  }
}

export async function api<T>(path: string, options: ApiOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  if (options.body && !(options.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json; charset=utf-8');
  }
  const token = getAuthToken();
  if (options.auth !== false && token && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  let response: Response;
  try {
    response = await fetch(path, { ...options, headers });
  } catch {
    throw new Error('无法连接后端服务');
  }
  if (!response.ok) {
    const message = await readApiError(response);
    if (options.auth !== false && options.redirectOn401 !== false && response.status === 401) {
      markAuthRedirecting();
      clearSession();
      if (token) markAuthExpired();
      if (typeof window !== 'undefined' && window.location.pathname !== '/login') {
        window.location.assign('/login');
      }
    }
    throw new ApiError(message, response.status);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return response.json() as Promise<T>;
}

export const recordExportAudit = (module: string, count: number) =>
  api<{ ok: boolean }>('/api/history/export', {
    method: 'POST',
    body: JSON.stringify({ module, count }),
  });
