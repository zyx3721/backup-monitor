import { cn } from '@/lib/utils';
import type { AgentBinaryMeta, AgentBinaryTargets } from '@/features/backup/types';
import { findSlotMeta, platforms, slotsByPlatform, type Platform } from './agent-update-platforms';

// PlatformTabs 平台切换卡片，展示托管槽位数与目标主机统计
export function PlatformTabs({
  platform,
  items,
  targets,
  disabled,
  onSelect,
}: {
  platform: Platform;
  items: AgentBinaryMeta[];
  targets: AgentBinaryTargets | null;
  disabled: boolean;
  onSelect: (platform: Platform) => void;
}) {
  return (
    <div className="grid gap-2 rounded-xl border border-[var(--bm-border)] bg-[var(--bm-control-bg-soft)] p-1.5 sm:grid-cols-2">
      {platforms.map(item => {
        const itemHosted = slotsByPlatform[item.id].filter(slot =>
          findSlotMeta(items, item.id, slot)
        ).length;
        const itemTarget = targets?.[item.id];
        const active = item.id === platform;
        const Icon = item.icon;
        return (
          <button
            key={item.id}
            type="button"
            aria-pressed={active}
            disabled={disabled}
            onClick={() => onSelect(item.id)}
            className={cn(
              'flex items-center gap-3 rounded-lg px-3 py-2.5 text-left transition-colors disabled:opacity-60',
              active
                ? 'bg-blue-500/10 shadow-[inset_0_0_0_1px_rgba(96,165,250,0.45)]'
                : 'hover:bg-[var(--bm-control-bg)]'
            )}
          >
            <span
              className={cn(
                'grid size-8 shrink-0 place-items-center rounded-lg border',
                active
                  ? 'border-blue-400/40 bg-blue-500/15 text-[var(--bm-accent-text)]'
                  : 'border-[var(--bm-border)] bg-[var(--bm-control-bg)] text-[var(--bm-text-muted)]'
              )}
            >
              <Icon size={15} />
            </span>
            <span className="min-w-0 flex-1">
              <span className="flex flex-wrap items-center gap-2">
                <span className="text-sm font-medium text-[var(--bm-text)]">{item.label}</span>
                {itemHosted > 0 ? (
                  <span className="rounded-full border border-emerald-500/25 bg-emerald-500/10 px-2 py-0.5 text-[10px] text-[var(--bm-status-green-text)]">
                    已托管 {itemHosted} 个槽位
                  </span>
                ) : (
                  <span className="rounded-full border border-[var(--bm-border)] px-2 py-0.5 text-[10px] text-[var(--bm-text-muted)]">
                    未上传
                  </span>
                )}
              </span>
              <span className="mt-1 block text-[11px] text-[var(--bm-text-muted)]">
                {itemTarget
                  ? `登记 ${itemTarget.total} 台 · 在线 ${itemTarget.online} 台`
                  : '目标统计加载中'}
              </span>
            </span>
            <span
              className={cn(
                'size-2 shrink-0 rounded-full',
                active ? 'bg-blue-400' : 'bg-transparent'
              )}
            />
          </button>
        );
      })}
    </div>
  );
}
