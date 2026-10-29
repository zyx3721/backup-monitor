import { useEffect, useState } from 'react';

export type BrandSettings = {
  siteName: string;
  loginName: string;
  appName: string;
  appSubtitle: string;
  iconData: string;
};

export const defaultBrandSettings: BrandSettings = {
  siteName: 'Backup Monitor',
  loginName: 'Backup Monitor',
  appName: 'Backup Monitor',
  appSubtitle: 'Backup Control Plane',
  iconData: '/favicon.svg',
};

let snapshot = defaultBrandSettings;
const listeners = new Set<(value: BrandSettings) => void>();

// applyBrandDocument 把品牌同步到文档标题与 favicon（仅客户端）
function applyBrandDocument(value: BrandSettings) {
  if (typeof document === 'undefined') return;
  document.title = value.siteName;
  const icon = document.querySelector<HTMLLinkElement>("link[rel='icon']");
  if (icon) icon.href = value.iconData;
}

// initializeBrandSettings 注入 root loader 从服务端实时拉取的品牌（SSR 与客户端各执行一次，
// 幂等）。刷新即直出最新配置，不再维护本地缓存副本
export function initializeBrandSettings(value: Partial<BrandSettings>) {
  snapshot = normalizeBrandSettings(value);
  applyBrandDocument(snapshot);
}

// setBrandSettings 保存配置后立即生效：更新内存快照、文档标题与 favicon
export function setBrandSettings(value: Partial<BrandSettings>) {
  snapshot = normalizeBrandSettings(value);
  applyBrandDocument(snapshot);
  listeners.forEach(listener => listener(snapshot));
}

export function useBrandSettings() {
  const [value, setValue] = useState(snapshot);

  useEffect(() => {
    listeners.add(setValue);
    return () => {
      listeners.delete(setValue);
    };
  }, []);

  return value;
}

function normalizeBrandSettings(value: Partial<BrandSettings>): BrandSettings {
  return {
    siteName: text(value.siteName, defaultBrandSettings.siteName),
    loginName: text(value.loginName, defaultBrandSettings.loginName),
    appName: text(value.appName, defaultBrandSettings.appName),
    appSubtitle: text(value.appSubtitle, defaultBrandSettings.appSubtitle),
    iconData: text(value.iconData, defaultBrandSettings.iconData),
  };
}

function text(value: string | undefined, fallback: string) {
  return value?.trim() || fallback;
}
