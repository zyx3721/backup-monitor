import { MessageCircle, Network } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { fetchAuthProvider, fetchWeComProvider } from '@/features/settings/api';
import { SettingsSplitLayout, UsageIndicator } from '../settings-primitives';
import { cardStyle } from '../settings-style';
import { LdapProviderDetail } from './LdapProviderDetail';
import { WeComSettingsPanel } from './WeComSettingsPanel';

type ProviderSelection = 'ldap' | 'wecom';

type ProviderStatus = { name: string; enabled: boolean };

// AuthSettingsPanel 认证配置管理器：侧栏列出 AD/LDAP 与企业微信提供者并切换详情面板
export function AuthSettingsPanel({ canManage }: { canManage: boolean }) {
  const [selected, setSelected] = useState<ProviderSelection>('ldap');
  const [ldapStatus, setLdapStatus] = useState<ProviderStatus>({ name: 'AD/LDAP', enabled: false });
  const [wecomStatus, setWecomStatus] = useState<ProviderStatus>({
    name: '企业微信',
    enabled: false,
  });

  const refresh = useCallback(async () => {
    await Promise.allSettled([
      fetchAuthProvider().then(setting =>
        setLdapStatus({ name: setting.name || 'AD/LDAP', enabled: setting.enabled })
      ),
      fetchWeComProvider().then(setting =>
        setWecomStatus({ name: setting.name || '企业微信', enabled: setting.enabled })
      ),
    ]);
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  return (
    <SettingsSplitLayout
      sidebarLabel="认证配置"
      sidebar={
        <>
          <button
            type="button"
            className="bm-config-side-card flex w-full items-start gap-3 rounded-lg p-3 text-left transition-all duration-200"
            data-active={selected === 'ldap'}
            style={cardStyle(selected === 'ldap')}
            onClick={() => setSelected('ldap')}
          >
            <span className="grid h-10 w-10 shrink-0 place-items-center rounded-lg border border-white/[0.08] bg-white/[0.05] text-sky-400">
              <Network size={19} />
            </span>
            <span className="min-w-0 flex-1">
              <span className="flex items-center justify-between gap-3">
                <span className="truncate text-sm font-semibold">{ldapStatus.name}</span>
                <UsageIndicator enabled={ldapStatus.enabled} />
              </span>
              <span className="mt-1 block text-xs leading-5 text-[var(--bm-text-muted)]">
                通过企业目录服务实现统一身份认证，支持 AD/LDAP 登录
              </span>
            </span>
          </button>
          <button
            type="button"
            className="bm-config-side-card flex w-full items-start gap-3 rounded-lg p-3 text-left transition-all duration-200"
            data-active={selected === 'wecom'}
            style={cardStyle(selected === 'wecom')}
            onClick={() => setSelected('wecom')}
          >
            <span className="grid h-10 w-10 shrink-0 place-items-center rounded-lg border border-white/[0.08] bg-white/[0.05] text-cyan-400">
              <MessageCircle size={19} />
            </span>
            <span className="min-w-0 flex-1">
              <span className="flex items-center justify-between gap-3">
                <span className="truncate text-sm font-semibold">{wecomStatus.name}</span>
                <UsageIndicator enabled={wecomStatus.enabled} />
              </span>
              <span className="mt-1 block text-xs leading-5 text-[var(--bm-text-muted)]">
                企业微信扫码登录，支持在右上角绑定企微账号
              </span>
            </span>
          </button>
        </>
      }
    >
      {selected === 'ldap' ? (
        <LdapProviderDetail canManage={canManage} onChanged={refresh} />
      ) : (
        <WeComSettingsPanel canManage={canManage} onChanged={refresh} />
      )}
    </SettingsSplitLayout>
  );
}
