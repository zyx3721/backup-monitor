import { KeySquare, Network, SlidersHorizontal, UsersRound, type LucideIcon } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { PermissionNotice } from '@/components/feedback/permission-notice';
import { SlidingTabs } from '@/components/layout/sliding-tabs';
import { userHasAnyPermission, userHasPermission } from '@/lib/permissions';
import { resetContentScroll } from '@/lib/scroll';
import { getStoredUser } from '@/lib/session';
import type { SettingsTab } from './types';
import { AuthSettingsPanel } from './panels/AuthSettingsPanel';
import { BaseSettingsPanel } from './panels/BaseSettingsPanel';
import { EmailSettingsPanel } from './panels/EmailSettingsPanel';
import { UserSettingsPanel } from './panels/UserSettingsPanel';

const tabs: Array<{
  id: SettingsTab;
  icon: LucideIcon;
  label: string;
  reads: string[];
  manage: string;
}> = [
  {
    id: 'base',
    icon: SlidersHorizontal,
    label: '基础配置',
    reads: ['settings.base.read'],
    manage: 'settings.base.manage',
  },
  {
    id: 'users',
    icon: UsersRound,
    label: '用户配置',
    reads: ['settings.users.read'],
    manage: 'settings.users.manage',
  },
  {
    id: 'auth',
    icon: Network,
    label: '认证配置',
    reads: ['settings.auth.read'],
    manage: 'settings.auth.manage',
  },
  {
    id: 'findpwd',
    icon: KeySquare,
    label: '找密配置',
    reads: ['settings.notifications.read'],
    manage: 'settings.notifications.manage',
  },
];

export function SystemSettingsPage() {
  const [tab, setTab] = useState<SettingsTab>('base');
  const user = getStoredUser();
  const visibleTabs = useMemo(
    () => tabs.filter(item => userHasAnyPermission(user, item.reads)),
    [user]
  );
  const active = visibleTabs.find(item => item.id === tab) ?? visibleTabs[0];

  useEffect(() => {
    resetContentScroll();
  }, [active?.id]);

  if (!active) {
    return <PermissionNotice section="系统配置" />;
  }

  const ActiveIcon = active.icon;
  const canManage = userHasPermission(user, active.manage);
  return (
    <div className="flex h-full min-h-0 flex-col gap-5">
      <div className="shrink-0">
        <h1 className="text-lg font-semibold text-[var(--bm-text)]">系统配置</h1>
        <p className="mt-1 text-sm text-[var(--bm-text-muted)]">
          集中管理基础与备份参数、用户与权限、身份认证、找回密码媒介
        </p>
      </div>
      <SlidingTabs
        items={visibleTabs}
        active={active.id}
        onChange={id => setTab(id as SettingsTab)}
      />

      <section
        className="bm-card-hover flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl p-5"
        style={{
          background: 'var(--bm-card)',
          border: '1px solid var(--bm-border)',
          boxShadow: 'var(--shadow-card)',
        }}
      >
        <div className="mb-4 flex shrink-0 flex-col gap-2 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h2 className="text-sm font-semibold text-[var(--bm-text)]">{active.label}</h2>
            <p className="mt-1 text-xs text-[var(--bm-text-muted)]">
              {sectionDescription(active.id)}
            </p>
          </div>
          <div className="flex items-center gap-2 rounded-lg border border-blue-400/25 bg-blue-500/10 px-3 py-2 text-xs text-[var(--bm-accent-text)]">
            <ActiveIcon size={14} />
            {sectionBadge(active.id)}
          </div>
        </div>
        {active.id === 'base' ? <BaseSettingsPanel canManage={canManage} /> : null}
        {active.id === 'users' ? <UserSettingsPanel canManage={canManage} /> : null}
        {active.id === 'auth' ? <AuthSettingsPanel canManage={canManage} /> : null}
        {active.id === 'findpwd' ? <EmailSettingsPanel canManage={canManage} /> : null}
      </section>
    </div>
  );
}

function sectionDescription(tab: SettingsTab) {
  if (tab === 'base') return '维护品牌标识、登录展示、安全时效与备份参数。';
  if (tab === 'users')
    return '维护平台用户、用户群组和角色权限。启用 AD/LDAP 后，外部账号也必须先在用户中创建并启用。';
  if (tab === 'auth') return '启用外部认证后，登录界面会显示对应登录方式。';
  return '用于找回密码时所使用的发送媒介。';
}

function sectionBadge(tab: SettingsTab) {
  if (tab === 'base') return '基础与备份参数';
  if (tab === 'users') return '用户与权限';
  if (tab === 'auth') return '身份认证';
  return '找回密码媒介';
}
