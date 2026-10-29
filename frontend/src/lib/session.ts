/* 会话层：登录令牌与用户信息的本地存取、当前用户缓存与认证类接口 */

import { api } from '@/lib/api';

const TOKEN_KEY = 'bm.auth.token';
const USER_KEY = 'bm.auth.user';
const EXPIRES_AT_KEY = 'bm.auth.expires_at';
export const AUTH_SESSION_CHANGED_EVENT = 'bm:auth-session-changed';

export type AuthUser = {
  id: string;
  username: string;
  displayName?: string;
  role: string;
  source: 'local' | 'ldap' | 'wecom';
  permissions: string[];
  wecomBound?: boolean;
};

export type AuthSession = {
  token: string;
  expiresAt: string;
  user: AuthUser;
};

export type PublicAuthProvider = {
  id: string;
  type: string;
  name: string;
  enabled: boolean;
};

export type PublicAuthConfiguration = {
  items: PublicAuthProvider[];
  total: number;
  passwordResetEnabled: boolean;
};

type AuthApiUser = {
  id: number;
  username: string;
  userDesc?: string;
  displayName?: string;
  role?: string;
  source?: 'local' | 'ldap' | 'wecom';
  wecomBound?: boolean;
  permissions?: string[];
};

export type AuthLoginResponse = {
  token: string;
  user: AuthApiUser;
};

type LogoutOptions = {
  waitForRemote?: boolean;
};

const CURRENT_USER_CACHE_TTL_MS = 1500;
const PUBLIC_AUTH_PROVIDERS_CACHE_TTL_MS = 1000;

let pendingCurrentUser: Promise<AuthUser> | null = null;
let cachedCurrentUser: { user: AuthUser; expiresAt: number } | null = null;
let pendingPublicAuthProviders: Promise<PublicAuthConfiguration> | null = null;
let cachedPublicAuthProviders: { value: PublicAuthConfiguration; expiresAt: number } | null = null;

// storage 登录态统一存取 localStorage，使同一浏览器内新建标签页与重启浏览器后仍保持登录，登录态仅随服务端会话过期失效
function storage() {
  if (typeof window === 'undefined') return null;
  return window.localStorage;
}

function emitSessionChanged() {
  if (typeof window === 'undefined') return;
  window.dispatchEvent(new Event(AUTH_SESSION_CHANGED_EVENT));
}

export function getAuthToken() {
  return storage()?.getItem(TOKEN_KEY) ?? '';
}

export function getStoredUser(): AuthUser | null {
  const raw = storage()?.getItem(USER_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as AuthUser;
  } catch {
    storage()?.removeItem(USER_KEY);
    return null;
  }
}

export function persistSession(session: AuthSession) {
  const store = storage();
  if (!store) return;
  store.setItem(TOKEN_KEY, session.token);
  store.setItem(USER_KEY, JSON.stringify(session.user));
  store.setItem(EXPIRES_AT_KEY, session.expiresAt);
  emitSessionChanged();
}

export function clearSession() {
  const store = storage();
  if (!store) return;
  store.removeItem(TOKEN_KEY);
  store.removeItem(USER_KEY);
  store.removeItem(EXPIRES_AT_KEY);
  emitSessionChanged();
}

const AUTH_EXPIRED_FLAG_KEY = 'bm.auth.expired';
let authRedirectingToLogin = false;

// markAuthExpired 仅在携带令牌的认证请求收到 401 时打标记，供登录页消费后统一提示会话过期
export function markAuthExpired() {
  if (typeof window === 'undefined') return;
  try {
    sessionStorage.setItem(AUTH_EXPIRED_FLAG_KEY, '1');
  } catch {
    return;
  }
}

// consumeAuthExpired 读取并清除会话失效标记，返回是否存在待提示的过期进入
export function consumeAuthExpired() {
  try {
    const expired = sessionStorage.getItem(AUTH_EXPIRED_FLAG_KEY) === '1';
    if (expired) sessionStorage.removeItem(AUTH_EXPIRED_FLAG_KEY);
    return expired;
  } catch {
    return false;
  }
}

// clearAuthExpiredFlag 清除尚未消费的会话失效标记：登录成功与主动登出后残留标记不再引发误提示
export function clearAuthExpiredFlag() {
  try {
    sessionStorage.removeItem(AUTH_EXPIRED_FLAG_KEY);
  } catch {
    return;
  }
}

// markAuthRedirecting 标记正以整页刷新跳转登录页，RequireAuth 据此跳过客户端跳转，
// 避免"先无动画进入登录页、再整页刷新带加载动画"的重复进入
export function markAuthRedirecting() {
  authRedirectingToLogin = true;
}

// isAuthRedirecting 返回当前文档是否已触发会话失效的整页跳转
export function isAuthRedirecting() {
  return authRedirectingToLogin;
}

export function persistUser(user: AuthUser) {
  storage()?.setItem(USER_KEY, JSON.stringify(user));
}

export function setCurrentUserSnapshot(user: AuthUser) {
  cachedCurrentUser = { user, expiresAt: Date.now() + CURRENT_USER_CACHE_TTL_MS };
  persistUser(user);
}

function normalizeAuthUser(user: AuthApiUser): AuthUser {
  return {
    id: String(user.id),
    username: user.username,
    displayName: user.displayName?.trim() || user.userDesc?.trim() || user.username,
    role: user.role || 'viewer',
    source: user.source || 'local',
    permissions: user.permissions ?? [],
    wecomBound: user.wecomBound ?? false,
  };
}

// sessionExpiresAt 解析 JWT 的 exp 声明作为本地会话过期时间，解析失败按后端默认 12 小时估算
function sessionExpiresAt(token: string): string {
  const fallback = new Date(Date.now() + 12 * 60 * 60 * 1000).toISOString();
  try {
    const payloadPart = token.split('.')[1] ?? '';
    const bytes = Uint8Array.from(atob(payloadPart.replace(/-/g, '+').replace(/_/g, '/')), char =>
      char.charCodeAt(0)
    );
    const claims = JSON.parse(new TextDecoder().decode(bytes)) as { exp?: number };
    if (typeof claims.exp === 'number' && claims.exp > 0) {
      return new Date(claims.exp * 1000).toISOString();
    }
  } catch {
    return fallback;
  }
  return fallback;
}

// persistLoginResponse 将登录类接口返回的令牌与用户信息持久化为本地会话
export function persistLoginResponse(response: AuthLoginResponse): AuthSession {
  const session: AuthSession = {
    token: response.token,
    expiresAt: sessionExpiresAt(response.token),
    user: normalizeAuthUser(response.user),
  };
  clearAuthExpiredFlag();
  persistSession(session);
  setCurrentUserSnapshot(session.user);
  return session;
}

export async function login(username: string, password: string, provider = 'local') {
  const response = await api<AuthLoginResponse>('/api/auth/login', {
    method: 'POST',
    auth: false,
    body: JSON.stringify({
      username,
      password,
      mode: provider === 'local' ? 'local' : 'ldap',
    }),
  });
  return persistLoginResponse(response);
}

// fetchCurrentUser 获取当前用户（带 1.5 秒缓存；绑定状态变更等需要立即反映的场景传 force 绕过缓存）
export function fetchCurrentUser(options?: { force?: boolean }) {
  if (!options?.force) {
    if (cachedCurrentUser && cachedCurrentUser.expiresAt > Date.now()) {
      return Promise.resolve(cachedCurrentUser.user);
    }
    if (pendingCurrentUser) return pendingCurrentUser;
  }
  pendingCurrentUser = api<AuthApiUser>('/api/auth/me')
    .then(response => {
      const user = normalizeAuthUser(response);
      cachedCurrentUser = { user, expiresAt: Date.now() + CURRENT_USER_CACHE_TTL_MS };
      persistUser(user);
      return user;
    })
    .finally(() => {
      pendingCurrentUser = null;
    });
  return pendingCurrentUser;
}

export function fetchPublicAuthProviders(options?: { force?: boolean }) {
  if (!options?.force) {
    if (cachedPublicAuthProviders && cachedPublicAuthProviders.expiresAt > Date.now()) {
      return Promise.resolve(cachedPublicAuthProviders.value);
    }
    if (pendingPublicAuthProviders) return pendingPublicAuthProviders;
  }
  pendingPublicAuthProviders = api<PublicAuthConfiguration>('/api/auth/providers', { auth: false })
    .then(value => {
      cachedPublicAuthProviders = {
        value,
        expiresAt: Date.now() + PUBLIC_AUTH_PROVIDERS_CACHE_TTL_MS,
      };
      return value;
    })
    .finally(() => {
      pendingPublicAuthProviders = null;
    });
  return pendingPublicAuthProviders;
}

// logout 先清理本地会话再通知后端：退出引发的组件重挂载若触发重新请求，令牌已不在本地，
// 不会携带"服务端已注销"的旧令牌而误报会话过期；注销请求失败静默忽略，不阻塞退出流程
export async function logout(options: LogoutOptions = {}) {
  const token = getAuthToken();
  pendingCurrentUser = null;
  cachedCurrentUser = null;
  clearAuthExpiredFlag();
  clearSession();
  if (!token) return;

  const request = fetch('/api/auth/logout', {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
    keepalive: true,
  }).catch(() => undefined);

  if (options.waitForRemote === false) {
    void request;
    return;
  }
  await request.then(() => undefined);
}

export function changePassword(body: {
  old_password: string;
  new_password: string;
  confirm_password: string;
}) {
  return api<{ status: string }>('/api/auth/change-password', {
    method: 'POST',
    body: JSON.stringify(body),
  });
}
