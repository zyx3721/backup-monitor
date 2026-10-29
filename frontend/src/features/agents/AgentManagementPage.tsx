import { KeyRound, Radar, RefreshCw, type LucideIcon } from 'lucide-react';
import { useEffect, useState } from 'react';
import { PermissionNotice } from '@/components/feedback/permission-notice';
import { SlidingTabs } from '@/components/layout/sliding-tabs';
import { cn } from '@/lib/utils';
import { PERM, userHasAnyPermission, userHasPermission } from '@/lib/permissions';
import { resetContentScroll } from '@/lib/scroll';
import { getStoredUser } from '@/lib/session';
import { AgentTokensPanel } from '@/features/settings/panels/AgentTokensPanel';
import { AgentUpdatePanel } from '@/features/settings/panels/AgentUpdatePanel';
import { CollectProfilesPanel } from '@/features/settings/panels/CollectProfilesPanel';
import { panelStyle } from '@/features/backup/components';

type AgentTab = 'profiles' | 'tokens' | 'update';

const tabs: Array<{ id: AgentTab; label: string; description: string; icon: LucideIcon }> = [
  {
    id: 'profiles',
    label: '采集配置',
    description: '按主机名下发采集规则，决定 Agent 采集哪些任务与产物',
    icon: Radar,
  },
  {
    id: 'tokens',
    label: 'Agent 令牌',
    description: 'Agent 上报、配置拉取与二进制下载使用的访问令牌',
    icon: KeyRound,
  },
  {
    id: 'update',
    label: 'Agent 更新',
    description: '托管 Linux / Windows 二进制并向在线主机推送自更新',
    icon: RefreshCw,
  },
];

export function AgentManagementPage() {
  const [tab, setTab] = useState<AgentTab>('profiles');
  const user = getStoredUser();
  const allowed = userHasAnyPermission(user, [PERM.agentsRead]);
  const canManage = userHasPermission(user, PERM.agentsManage);

  useEffect(() => {
    resetContentScroll();
  }, [tab]);

  if (!allowed) {
    return <PermissionNotice section="Agent 管理" />;
  }

  return (
    <div className={cn('bm-page-fill gap-5', tab === 'update' && 'h-auto min-h-full')}>
      <div className="shrink-0">
        <h1 className="text-lg font-semibold text-[var(--bm-text)]">Agent 管理</h1>
        <p className="mt-1 text-sm text-[var(--bm-text-muted)]">
          维护采集规则、访问令牌与 Agent 版本分发
        </p>
      </div>
      <SlidingTabs items={tabs} active={tab} onChange={id => setTab(id as AgentTab)} />
      {tab === 'update' ? (
        <section className="bm-card-hover flex-1 rounded-xl p-5" style={panelStyle}>
          <AgentUpdatePanel canManage={canManage} />
        </section>
      ) : (
        <div className="flex min-h-0 flex-1 flex-col">
          {tab === 'profiles' ? <CollectProfilesPanel canManage={canManage} /> : null}
          {tab === 'tokens' ? <AgentTokensPanel canManage={canManage} /> : null}
        </div>
      )}
    </div>
  );
}
