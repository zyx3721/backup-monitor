import { AlertCircle, CheckCircle2, Copy, Info, Loader2, Timer } from 'lucide-react';
import { cn } from '@/lib/utils';
import { AppTooltip } from '@/components/feedback/app-tooltip';
import { formatBytes, formatTime } from '@/features/backup/components';
import type { AgentBinaryMeta, PushResult } from '@/features/backup/types';
import { useEffect, useState } from 'react';
import {
  findSlotMeta,
  platformLabel,
  slotLabel,
  slotsByPlatform,
  windowsRequirementLabel,
  type Platform,
} from './agent-update-platforms';

export function SectionHeading({
  step,
  title,
  description,
}: {
  step: string;
  title: string;
  description: string;
}) {
  return (
    <div className="flex items-start gap-3">
      <span className="mt-0.5 grid h-6 w-6 shrink-0 place-items-center rounded-md border border-blue-400/30 bg-blue-500/10 font-mono text-[11px] text-[var(--bm-accent-text)]">
        {step}
      </span>
      <div>
        <h3 className="text-sm font-semibold text-[var(--bm-text)]">{title}</h3>
        <p className="mt-0.5 text-[11px] leading-5 text-[var(--bm-text-muted)]">{description}</p>
      </div>
    </div>
  );
}

export function PushModeCard({
  active,
  title,
  description,
  onSelect,
}: {
  active: boolean;
  title: string;
  description: string;
  onSelect: () => void;
}) {
  return (
    <label
      className={cn(
        'flex cursor-pointer items-start gap-3 rounded-lg border px-3 py-2.5 transition-colors',
        active
          ? 'border-blue-400/50 bg-blue-500/10'
          : 'border-[var(--bm-border)] bg-[var(--bm-control-bg-soft)] hover:border-blue-400/30'
      )}
    >
      <input
        type="radio"
        name="agent-push-mode"
        className="sr-only"
        checked={active}
        onChange={onSelect}
      />
      <span
        className={cn(
          'mt-0.5 grid size-4 shrink-0 place-items-center rounded-full border',
          active ? 'border-blue-400' : 'border-[var(--bm-border)]'
        )}
      >
        {active ? <span className="size-1.5 rounded-full bg-blue-400" /> : null}
      </span>
      <span>
        <span className="block text-sm font-medium text-[var(--bm-text)]">{title}</span>
        <span className="mt-0.5 block text-[11px] leading-5 text-[var(--bm-text-muted)]">
          {description}
        </span>
      </span>
    </label>
  );
}

// HostedSlots 展示当前平台各槽位的托管状态
export function HostedSlots({
  platform,
  items,
  onCopy,
}: {
  platform: Platform;
  items: AgentBinaryMeta[];
  onCopy: (value: string) => void;
}) {
  return (
    <div className="space-y-1.5">
      {slotsByPlatform[platform].map(slot => {
        const meta = findSlotMeta(items, platform, slot);
        return (
          <div
            key={`${slot.arch}-${slot.variant}`}
            className="flex flex-wrap items-center gap-x-3 gap-y-1.5 rounded-lg border border-[var(--bm-border)]/60 bg-[var(--bm-control-bg)] px-3 py-2 text-[11px]"
          >
            <span className="w-24 shrink-0 font-mono text-[var(--bm-text)]">{slotLabel(slot)}</span>
            {meta ? (
              <>
                <span className="rounded-full border border-emerald-500/25 bg-emerald-500/10 px-2 py-0.5 text-[var(--bm-status-green-text)]">
                  v{meta.version}
                </span>
                <span className="font-mono text-[var(--bm-text-muted)]">
                  {meta.goVersion || '-'}
                </span>
                <span className="text-[var(--bm-text-muted)]">{formatBytes(meta.size)}</span>
                <span className="text-[var(--bm-text-muted)]">{formatTime(meta.uploadedAt)}</span>
                <span className="min-w-0 flex-1 truncate font-mono text-[10px] text-[var(--bm-text-muted)]">
                  {meta.sha256}
                </span>
                <button
                  type="button"
                  onClick={() => onCopy(meta.sha256)}
                  className="inline-flex shrink-0 items-center gap-1 rounded-md border border-[var(--bm-border)] px-2 py-0.5 text-[10px] text-[var(--bm-text-muted)] transition-colors hover:border-blue-400/40 hover:text-[var(--bm-accent-text)]"
                >
                  <Copy size={10} />
                  复制
                </button>
              </>
            ) : (
              <span className="text-[var(--bm-text-muted)]">未上传</span>
            )}
          </div>
        );
      })}
    </div>
  );
}

// WindowsCompatHint 展示 Windows 槽位的选包规则
export function WindowsCompatHint({ standard }: { standard?: AgentBinaryMeta }) {
  const requirement = standard?.goVersion
    ? `；当前标准包由 ${standard.goVersion} 编译，要求 ${windowsRequirementLabel(standard.goVersion)}`
    : '';
  const text = `推送时按主机架构与 Windows 构建号自动选包：Build ≥ 10240 使用标准包（未上传即跳过），Build < 10240 使用 legacy 包${requirement}`;
  return (
    <div className="flex items-start gap-2 rounded-lg border border-blue-400/25 bg-blue-500/10 px-3 py-2 text-[11px] leading-5 text-[var(--bm-accent-text)]">
      <Info size={14} className="mt-0.5 shrink-0" />
      <span>{text}</span>
    </div>
  );
}

export type PushPollState = {
  active: boolean;
  nextCheckAt: number | null;
  rounds: number;
  checking: boolean;
};

// PushPollBadge 展示推送结果自动复查的倒计时、进行中与最终状态
export function PushPollBadge({
  outcome,
  poll,
}: {
  outcome: 'idle' | 'pending' | 'success' | 'failed';
  poll: PushPollState;
}) {
  const [, setTick] = useState(0);
  useEffect(() => {
    if (outcome !== 'pending' || !poll.active || poll.checking) return;
    const timer = window.setInterval(() => setTick(value => value + 1), 1000);
    return () => window.clearInterval(timer);
  }, [outcome, poll.active, poll.checking]);
  if (outcome === 'idle') return null;
  const base = 'inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px]';
  if (outcome === 'success') {
    return (
      <span
        className={cn(
          base,
          'border-emerald-500/25 bg-emerald-500/10 text-[var(--bm-status-green-text)]'
        )}
      >
        <CheckCircle2 size={12} />
        已获取最终状态
      </span>
    );
  }
  if (outcome === 'failed') {
    return (
      <span className={cn(base, 'border-rose-500/25 bg-rose-500/10 text-rose-500')}>
        <AlertCircle size={12} />
        {poll.rounds > 0 ? '复查超时，已停止等待' : '未触发复查'}
      </span>
    );
  }
  if (poll.checking) {
    return (
      <span
        className={cn(
          base,
          'border-amber-500/25 bg-amber-500/10 text-[var(--bm-status-yellow-text)]'
        )}
      >
        <Loader2 size={12} className="animate-spin" />
        正在复查主机版本
      </span>
    );
  }
  if (!poll.active) return null;
  const secondsLeft = poll.nextCheckAt
    ? Math.min(3, Math.max(1, Math.ceil((poll.nextCheckAt - Date.now()) / 1000)))
    : 1;
  const countdown = `${secondsLeft} 秒后`;
  const text =
    poll.rounds === 0
      ? `等待重启结果 · ${countdown}自动复查`
      : `尚未获取最终状态 · ${countdown}第 ${poll.rounds + 1} 次复查`;
  return (
    <span
      className={cn(
        base,
        'border-amber-500/25 bg-amber-500/10 text-[var(--bm-status-yellow-text)]'
      )}
    >
      <Timer size={12} />
      {text}
    </span>
  );
}

export function PushResultsTable({
  platform,
  results,
  outcome,
  poll,
}: {
  platform: Platform;
  results: PushResult[];
  outcome: 'idle' | 'pending' | 'success' | 'failed';
  poll?: PushPollState;
}) {
  const successCount = results.filter(item => item.delivered && item.restarted === true).length;
  const skippedCount = results.filter(item => item.skipped === true).length;
  const pollState = poll ?? { active: false, nextCheckAt: null, rounds: 0, checking: false };
  return (
    <div className="rounded-lg border border-[var(--bm-border)]/60 bg-[var(--bm-control-bg)] p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm font-medium text-[var(--bm-text)]">推送结果</span>
          <span className="text-[11px] text-[var(--bm-text-muted)]">
            {platformLabel(platform)} · 按主机自动选包
          </span>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <PushPollBadge outcome={outcome} poll={pollState} />
          <span
            className={cn(
              'rounded-full border px-2 py-0.5 text-[11px]',
              successCount === results.length
                ? 'border-emerald-500/25 bg-emerald-500/10 text-[var(--bm-status-green-text)]'
                : outcome === 'failed'
                  ? 'border-rose-500/25 bg-rose-500/10 text-rose-500'
                  : 'border-amber-500/25 bg-amber-500/10 text-[var(--bm-status-yellow-text)]'
            )}
          >
            已上报 {successCount} / {results.length}
          </span>
          {skippedCount > 0 ? (
            <span className="rounded-full border border-[var(--bm-border)] px-2 py-0.5 text-[11px] text-[var(--bm-text-muted)]">
              已跳过 {skippedCount}
            </span>
          ) : null}
        </div>
      </div>
      <div className="mt-2 overflow-x-auto rounded-md border border-[var(--bm-border)]/60">
        <table className="w-full min-w-[900px] text-center text-sm">
          <thead className="bg-[var(--bm-table-head-bg)] text-[11px] text-[var(--bm-text-muted)]">
            <tr>
              <th className="px-3 py-2 font-semibold">主机</th>
              <th className="px-3 py-2 font-semibold">IP</th>
              <th className="px-3 py-2 font-semibold">当前版本</th>
              <th className="px-3 py-2 font-semibold">目标版本</th>
              <th className="px-3 py-2 font-semibold">架构</th>
              <th className="px-3 py-2 font-semibold">结果</th>
              <th className="px-3 py-2 font-semibold">说明</th>
            </tr>
          </thead>
          <tbody>
            {results.map(item => {
              const skipped = item.skipped === true;
              const targetVersion = item.version ?? '';
              const matched = targetVersion !== '' && item.agentVersion === targetVersion;
              const succeeded = !skipped && item.delivered && item.restarted === true;
              const failed = !skipped && outcome === 'failed' && item.delivered && !succeeded;
              const awaitingRestart = !skipped && item.delivered && item.restarted !== true;
              let description = '';
              if (skipped) {
                description = item.reason || '未找到匹配的安装包，已跳过';
              } else if (succeeded) {
                description = matched
                  ? `已重启至 v${targetVersion}`
                  : `已重启并上报，当前版本 ${item.agentVersion || '未知'}`;
              } else if (failed) {
                description = '未见重启后的上报，请检查 Agent 日志';
              } else if (awaitingRestart) {
                description = matched
                  ? `已下发，等待重启（当前已是 v${targetVersion}，将原地重启）`
                  : '已下发，等待重启';
              } else {
                description = item.delivered ? '已下发，等待重启' : item.message || '-';
              }
              if (!skipped && item.warning) description += `；${item.warning}`;
              const archText = item.arch
                ? `${item.arch}${item.variant === 'legacy' ? ' legacy' : ''}`
                : '-';
              return (
                <tr
                  key={`${item.hostname}-${item.ip}`}
                  className="border-t border-[var(--bm-border)]/50 transition-colors hover:bg-[var(--bm-control-bg-soft)]"
                >
                  <td className="px-3 py-2 text-xs text-[var(--bm-text)]">{item.hostname}</td>
                  <td className="px-3 py-2 text-xs text-[var(--bm-text)]">{item.ip}</td>
                  <td className="px-3 py-2 text-xs text-[var(--bm-text)]">
                    {item.agentVersion || '-'}
                  </td>
                  <td className="px-3 py-2 text-xs text-[var(--bm-text)]">
                    {targetVersion || '-'}
                  </td>
                  <td className="px-3 py-2 text-xs text-[var(--bm-text)]">{archText}</td>
                  <td className="px-3 py-2">
                    <span
                      className={cn(
                        'inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px]',
                        skipped
                          ? 'border-[var(--bm-border)] bg-transparent text-[var(--bm-text-muted)]'
                          : succeeded
                            ? 'border-emerald-500/25 bg-emerald-500/10 text-[var(--bm-status-green-text)]'
                            : failed
                              ? 'border-rose-500/25 bg-rose-500/10 text-rose-500'
                              : 'border-amber-500/25 bg-amber-500/10 text-[var(--bm-status-yellow-text)]'
                      )}
                    >
                      <span className="size-1.5 rounded-full bg-current" />
                      {skipped
                        ? '已跳过'
                        : succeeded
                          ? '已上报'
                          : failed
                            ? '失败'
                            : item.delivered
                              ? '已下发'
                              : '未送达'}
                    </span>
                  </td>
                  <td className="px-3 py-2 text-xs text-[var(--bm-text)]">
                    <AppTooltip label={description} placement="top" wrap>
                      <span className="mx-auto block max-w-80 truncate">{description}</span>
                    </AppTooltip>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
