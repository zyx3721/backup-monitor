/* 企业微信认证：直连扫码与统一认证中心两种模式的登录、绑定接口与事件常量 */

import { api } from '@/lib/api';
import { persistLoginResponse, type AuthLoginResponse } from '@/lib/session';

// WECOM_BIND_MESSAGE 绑定弹窗通过 postMessage 通知主窗口绑定结果的事件类型
export const WECOM_BIND_MESSAGE = 'bm:wecom-bind';

// WECOM_PROVIDERS_CHANGED_EVENT 企业微信认证启用状态变更事件，通知布局刷新绑定入口
export const WECOM_PROVIDERS_CHANGED_EVENT = 'bm:wecom-providers-changed';

// loginWithWecomCallback 企业微信直连扫码回调换取会话并持久化
export async function loginWithWecomCallback(code: string, state: string) {
  const response = await api<AuthLoginResponse>('/api/auth/wecom/callback', {
    method: 'POST',
    auth: false,
    body: JSON.stringify({ code, state }),
  });
  return persistLoginResponse(response);
}

// loginWithWecomSSO 统一认证中心回跳 ticket 换取会话并持久化
export async function loginWithWecomSSO(ticket: string) {
  const response = await api<AuthLoginResponse>('/api/auth/wecom/sso/callback', {
    method: 'POST',
    auth: false,
    body: JSON.stringify({ ticket }),
  });
  return persistLoginResponse(response);
}

// WecomAuthorizeEmbed 内嵌二维码登录参数：iframe 地址、回跳路径与直连模式签名 state
export type WecomAuthorizeEmbed = {
  auth_mode: 'direct' | 'sso';
  iframe_url: string;
  state?: string;
  callback_path: string;
};

// WecomAuthorizeResponse 企业微信扫码登录跳转地址与内嵌二维码参数，embed 缺失表示仅支持整页跳转
type WecomAuthorizeResponse = {
  url: string;
  embed?: WecomAuthorizeEmbed;
};

// fetchWecomAuthorize 获取企业微信扫码登录跳转地址与内嵌二维码参数（公开接口）
export async function fetchWecomAuthorize() {
  return api<WecomAuthorizeResponse>('/api/auth/wecom/authorize');
}

// fetchWecomBindUrl 获取当前用户的企业微信绑定扫码地址
export async function fetchWecomBindUrl() {
  const response = await api<{ url: string }>('/api/auth/wecom/bind-url');
  return response.url;
}

// bindWecomCallback 完成当前登录用户的企微绑定（直连模式，绑定弹窗内调用；401 不触发会话清理跳转）
export async function bindWecomCallback(code: string, state: string) {
  return api<{ ok: boolean; wecomUserid: string }>('/api/auth/wecom/bind', {
    method: 'POST',
    body: JSON.stringify({ code, state }),
    redirectOn401: false,
  });
}

// bindWecomSSO 完成当前登录用户的企微绑定（统一认证中心模式，绑定弹窗内调用；401 不触发会话清理跳转）
export function bindWecomSSO(ticket: string) {
  return api<{ ok: boolean; wecomUserid: string }>('/api/auth/wecom/sso/bind', {
    method: 'POST',
    body: JSON.stringify({ ticket }),
    redirectOn401: false,
  });
}

// unbindWecom 解除当前登录用户的企微绑定
export async function unbindWecom() {
  return api<{ ok: boolean }>('/api/auth/wecom/bind', { method: 'DELETE' });
}
