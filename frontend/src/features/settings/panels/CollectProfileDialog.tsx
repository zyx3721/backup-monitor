import { CircleHelp } from 'lucide-react';
import { useLayoutEffect, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { AppTooltip } from '@/components/feedback/app-tooltip';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';
import type { AgentProfileConfig } from '@/features/backup/types';
import type { ProfileDraft, ProfileTab } from './profile-draft';

const tabs: Array<{ id: ProfileTab; label: string }> = [
  { id: 'basic', label: '基本信息' },
  { id: 'collect', label: '采集选项' },
  { id: 'filter', label: '过滤规则' },
];

export function CollectProfileDialog({
  draft,
  tab,
  busy,
  onTabChange,
  onDraftChange,
  onClose,
  onSave,
}: {
  draft: ProfileDraft | null;
  tab: ProfileTab;
  busy: boolean;
  onTabChange: (tab: ProfileTab) => void;
  onDraftChange: (draft: ProfileDraft) => void;
  onClose: () => void;
  onSave: () => void;
}) {
  const updateConfig = <K extends keyof AgentProfileConfig>(
    key: K,
    value: AgentProfileConfig[K]
  ) => {
    if (!draft) return;
    onDraftChange({ ...draft, config: { ...draft.config, [key]: value } });
  };

  const tabRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const [indicator, setIndicator] = useState({ left: 0, width: 0 });

  useLayoutEffect(() => {
    if (!draft) {
      setIndicator({ left: 0, width: 0 });
      return;
    }
    let frame = 0;
    const measure = () => {
      const active = tabRefs.current[tabs.findIndex(item => item.id === tab)];
      if (active?.isConnected) {
        setIndicator({ left: active.offsetLeft, width: active.offsetWidth });
      } else {
        frame = requestAnimationFrame(measure);
      }
    };
    measure();
    window.addEventListener('resize', measure);
    return () => {
      cancelAnimationFrame(frame);
      window.removeEventListener('resize', measure);
    };
  }, [tab, draft]);
  return (
    <Dialog open={Boolean(draft)} onOpenChange={open => !open && !busy && onClose()}>
      <DialogContent className="flex h-[min(86vh,680px)] max-w-3xl flex-col gap-0 overflow-hidden p-0">
        <DialogHeader className="flex h-16 shrink-0 flex-row items-center border-b border-[var(--bm-border)] px-6">
          <DialogTitle className="text-xl">
            {draft?.id ? '编辑采集配置' : '新建采集配置'}
          </DialogTitle>
        </DialogHeader>
        <div className="relative flex shrink-0 gap-6 border-b border-[var(--bm-border)] px-6">
          {tabs.map((item, index) => (
            <button
              key={item.id}
              ref={element => {
                tabRefs.current[index] = element;
              }}
              type="button"
              onClick={() => onTabChange(item.id)}
              className="py-3 text-sm font-medium outline-none transition-colors duration-300"
              style={{
                color: tab === item.id ? 'var(--bm-accent-text)' : 'var(--bm-text-muted)',
              }}
            >
              {item.label}
            </button>
          ))}
          <span
            className="absolute bottom-0 h-0.5 rounded-full bg-[var(--bm-accent)] transition-[left,width] duration-500 ease-out"
            style={{ left: indicator.left, width: indicator.width }}
          />
        </div>
        <div className="bm-hidden-scrollbar min-h-0 flex-1 overflow-y-auto px-6 py-5">
          {draft && tab === 'basic' ? (
            <div className="space-y-5">
              <FormRow label="配置名称" required>
                <Input
                  value={draft.name}
                  placeholder="如：数据库主机"
                  onChange={event => onDraftChange({ ...draft, name: event.target.value })}
                />
              </FormRow>
              <FormRow label="匹配主机" required hint="支持 * 与 ? 通配符，如 db-*">
                <Input
                  value={draft.hostname}
                  className="font-mono"
                  onChange={event => onDraftChange({ ...draft, hostname: event.target.value })}
                />
              </FormRow>
              <FormRow label="优先级" hint="数值越大越优先">
                <Input
                  type="number"
                  min={0}
                  max={100}
                  value={draft.priority}
                  onChange={event =>
                    onDraftChange({
                      ...draft,
                      priority: Math.max(0, Number(event.target.value) || 0),
                    })
                  }
                />
              </FormRow>
              <FormRow label="上报间隔（秒）" hint="0 表示使用服务端默认，1–9 秒按 10 秒生效">
                <Input
                  type="number"
                  min={0}
                  max={86400}
                  value={draft.config.reportIntervalSec}
                  onChange={event =>
                    updateConfig('reportIntervalSec', Math.max(0, Number(event.target.value) || 0))
                  }
                />
              </FormRow>
              <FormRow label="启用">
                <Switch
                  checked={draft.enabled}
                  ariaLabel="启用该采集配置"
                  onCheckedChange={value => onDraftChange({ ...draft, enabled: value })}
                />
              </FormRow>
              <FormRow label="描述">
                <Input
                  value={draft.description}
                  placeholder="可选"
                  onChange={event => onDraftChange({ ...draft, description: event.target.value })}
                />
              </FormRow>
            </div>
          ) : null}

          {draft && tab === 'collect' ? (
            <div className="space-y-6">
              <section>
                <SectionLabel>采集来源</SectionLabel>
                <div className="mt-3 grid gap-3 lg:grid-cols-3">
                  <SourceCard
                    label="采集 Crontab"
                    description="识别 crontab 中带「任务标签前缀」的备份任务及其执行状态"
                    checked={draft.config.collectCrontab}
                    onChange={value => updateConfig('collectCrontab', value)}
                  />
                  <SourceCard
                    label="采集 Veeam"
                    description="通过 veeamconfig 采集 Veeam 备份作业与会话"
                    checked={draft.config.collectVeeam}
                    onChange={value => updateConfig('collectVeeam', value)}
                  />
                  <SourceCard
                    label="产物检查"
                    description="检查任务标签 dir= 声明目录中的备份产物是否按时产出"
                    checked={draft.config.artifactCheck}
                    onChange={value => updateConfig('artifactCheck', value)}
                  />
                </div>
              </section>
              <section className="border-t border-[var(--bm-border)] pt-5">
                <SectionLabel>任务识别</SectionLabel>
                <div className="mt-4">
                  <FormRow label="任务标签前缀">
                    <Input
                      value={draft.config.labelPrefix}
                      className="font-mono"
                      onChange={event => updateConfig('labelPrefix', event.target.value)}
                    />
                  </FormRow>
                  <p className="mt-2 pl-[148px] text-[11px] leading-5 text-[var(--bm-text-muted)]">
                    crontab 行中带此标记即识别为备份任务；行尾可追加 name= / key= / dir= / match= /
                    status= 声明属性。
                  </p>
                </div>
              </section>
              <section className="border-t border-[var(--bm-border)] pt-5">
                <SectionLabel>产物检查参数</SectionLabel>
                <div className="mt-3 grid gap-4 sm:grid-cols-2">
                  <NumberField
                    label="产物最大年龄"
                    unit="小时"
                    hint="仅对无执行计划的任务生效；有计划的按计划周期判断产物新旧"
                    value={draft.config.maxArtifactAgeHours}
                    min={1}
                    max={720}
                    onChange={value => updateConfig('maxArtifactAgeHours', value)}
                  />
                  <NumberField
                    label="产物扫描深度"
                    unit="层"
                    hint="查找产物时递归扫描目录的最大层级"
                    value={draft.config.artifactMaxDepth}
                    min={1}
                    max={10}
                    onChange={value => updateConfig('artifactMaxDepth', value)}
                  />
                </div>
              </section>
              <section className="border-t border-[var(--bm-border)] pt-5">
                <SectionLabel>Veeam 参数</SectionLabel>
                <div className="mt-3 grid gap-4 sm:grid-cols-3">
                  <NumberField
                    label="单作业会话上限"
                    unit="条"
                    hint="每个 Veeam 作业最多保留的会话条数，0 表示默认 5"
                    value={draft.config.maxSessionsPerJob}
                    min={0}
                    max={100}
                    onChange={value => updateConfig('maxSessionsPerJob', value)}
                  />
                  <NumberField
                    label="会话详情调用"
                    unit="条"
                    hint="单次采集最多查询的会话详情次数，0 表示默认 10"
                    value={draft.config.maxSessionInfoCalls}
                    min={0}
                    max={100}
                    onChange={value => updateConfig('maxSessionInfoCalls', value)}
                  />
                  <NumberField
                    label="作业详情调用"
                    unit="条"
                    hint="单次采集最多查询的作业详情次数，0 表示默认 20"
                    value={draft.config.maxJobInfoCalls}
                    min={0}
                    max={100}
                    onChange={value => updateConfig('maxJobInfoCalls', value)}
                  />
                </div>
              </section>
              <section className="border-t border-[var(--bm-border)] pt-5">
                <SectionLabel hint="0 表示使用 Agent 内置默认值，一般保持默认即可">
                  进阶参数
                </SectionLabel>
                <div className="mt-3 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
                  <NumberField
                    label="心跳间隔"
                    unit="秒"
                    hint="Agent 向服务端发送心跳的间隔秒数"
                    value={draft.config.heartbeatIntervalSec}
                    min={5}
                    max={600}
                    onChange={value => updateConfig('heartbeatIntervalSec', value)}
                  />
                  <NumberField
                    label="手动执行判定"
                    unit="分"
                    hint="开始时间偏离计划超过该值判为手动执行，0 表示默认 30"
                    value={draft.config.manualTriggerMinutes}
                    min={0}
                    max={1440}
                    onChange={value => updateConfig('manualTriggerMinutes', value)}
                  />
                  <NumberField
                    label="产物时间容差"
                    unit="分"
                    hint="允许产物时间早于计划时刻的分钟数，0 表示默认 10"
                    value={draft.config.artifactPreToleranceMinutes}
                    min={0}
                    max={1440}
                    onChange={value => updateConfig('artifactPreToleranceMinutes', value)}
                  />
                  <NumberField
                    label="自动更新检查"
                    unit="分"
                    hint="Agent 定时查询托管版本并自动升级，0 表示禁用"
                    value={draft.config.autoUpdateMinutes}
                    min={0}
                    max={10080}
                    onChange={value => updateConfig('autoUpdateMinutes', value)}
                  />
                </div>
              </section>
            </div>
          ) : null}

          {draft && tab === 'filter' ? (
            <div className="space-y-5">
              <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                <ListField
                  label="关键字白名单"
                  hint="命中任一关键字才采集"
                  value={draft.config.includeKeywords}
                  onChange={value => updateConfig('includeKeywords', value)}
                />
                <ListField
                  label="路径白名单"
                  hint="脚本或产物路径命中才采集"
                  value={draft.config.includePaths}
                  onChange={value => updateConfig('includePaths', value)}
                />
              </div>
              <ListField
                label="排除关键字"
                hint="命中任一关键字即忽略"
                value={draft.config.excludeKeywords}
                onChange={value => updateConfig('excludeKeywords', value)}
              />
            </div>
          ) : null}
        </div>
        <DialogFooter className="flex shrink-0 flex-row items-center justify-end gap-3 border-t border-[var(--bm-border)] px-6 py-4">
          <Button
            type="button"
            variant="outline"
            className="h-10 rounded-lg px-5"
            disabled={busy}
            onClick={onClose}
          >
            取消
          </Button>
          <Button
            type="button"
            className="bm-primary-soft-button h-10 rounded-lg px-5"
            disabled={busy}
            onClick={onSave}
          >
            {busy ? '保存中…' : '保存'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
function FormRow({
  label,
  hint,
  required,
  children,
}: {
  label: string;
  hint?: string;
  required?: boolean;
  children: ReactNode;
}) {
  return (
    <div className="grid grid-cols-[132px_minmax(0,1fr)] items-center gap-4">
      <div className="flex justify-end">
        <AppTooltip label={hint} placement="top" wrap>
          <span
            className={`flex items-center gap-1 text-right text-sm text-[var(--bm-text-muted)] ${
              hint ? 'cursor-help' : ''
            }`}
          >
            {required ? <span className="text-red-400">*</span> : null}
            {label}
            {hint ? <CircleHelp size={12} className="opacity-60" /> : null}
          </span>
        </AppTooltip>
      </div>
      <div className="min-w-0">{children}</div>
    </div>
  );
}

function SectionLabel({ children, hint }: { children: ReactNode; hint?: string }) {
  return (
    <div className="flex items-center gap-2 text-sm font-semibold text-[var(--bm-text)]">
      <span className="h-4 w-[3px] rounded-full bg-[var(--bm-accent)]" />
      <AppTooltip label={hint} placement="top" wrap>
        <span className={`flex items-center gap-1 ${hint ? 'cursor-help' : ''}`}>
          {children}
          {hint ? <CircleHelp size={12} className="opacity-60" /> : null}
        </span>
      </AppTooltip>
    </div>
  );
}

function SourceCard({
  label,
  description,
  checked,
  onChange,
}: {
  label: string;
  description: string;
  checked: boolean;
  onChange: (value: boolean) => void;
}) {
  return (
    <div
      onClick={() => onChange(!checked)}
      className={`bm-card-hover cursor-pointer rounded-xl border p-4 ${
        checked
          ? 'border-[var(--bm-accent)]/45 bg-[var(--bm-accent)]/[0.08]'
          : 'border-[var(--bm-border)] bg-white/[0.02]'
      }`}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="text-sm font-semibold text-[var(--bm-accent-text)]">{label}</div>
        <span onClick={event => event.stopPropagation()}>
          <Switch checked={checked} ariaLabel={`启用${label}`} onCheckedChange={onChange} />
        </span>
      </div>
      <p className="mt-3 text-xs leading-5 text-[var(--bm-text-muted)]">{description}</p>
    </div>
  );
}

function NumberField({
  label,
  unit,
  hint,
  value,
  min,
  max,
  onChange,
}: {
  label: string;
  unit: string;
  hint?: string;
  value: number;
  min: number;
  max: number;
  onChange: (value: number) => void;
}) {
  return (
    <div>
      <div className="mb-1.5 flex items-center gap-1 text-sm text-[var(--bm-text-muted)]">
        <AppTooltip label={hint} placement="top" wrap>
          <span className={`flex items-center gap-1 ${hint ? 'cursor-help' : ''}`}>
            {label}
            {hint ? <CircleHelp size={12} className="opacity-60" /> : null}
          </span>
        </AppTooltip>
      </div>
      <div className="flex items-center gap-2">
        <Input
          type="number"
          min={min}
          max={max}
          value={value}
          onChange={event =>
            onChange(Math.min(max, Math.max(min, Number(event.target.value) || 0)))
          }
          className="h-10 text-center"
        />
        <span className="shrink-0 text-xs text-[var(--bm-text-muted)]">{unit}</span>
      </div>
    </div>
  );
}

function ListField({
  label,
  hint,
  value,
  onChange,
}: {
  label: string;
  hint?: string;
  value: string[];
  onChange: (value: string[]) => void;
}) {
  return (
    <label className="block">
      <div className="mb-1.5 flex items-center gap-1 text-sm text-[var(--bm-text-muted)]">
        <AppTooltip label={hint} placement="top" wrap>
          <span className={`flex items-center gap-1 ${hint ? 'cursor-help' : ''}`}>
            {label}（每行一个）
            {hint ? <CircleHelp size={12} className="opacity-60" /> : null}
          </span>
        </AppTooltip>
      </div>
      <textarea
        wrap="soft"
        rows={6}
        value={value.join('\n')}
        onMouseMove={event => {
          const textarea = event.currentTarget;
          textarea.style.cursor =
            event.nativeEvent.offsetX > textarea.clientWidth ? 'default' : 'text';
        }}
        onMouseLeave={event => {
          event.currentTarget.style.cursor = '';
        }}
        onChange={event => {
          const textarea = event.currentTarget;
          const atEnd = textarea.selectionStart === textarea.value.length;
          onChange(textarea.value.split('\n').map(item => item.trim()));
          if (atEnd) {
            requestAnimationFrame(() => {
              textarea.scrollTop = textarea.scrollHeight;
            });
          }
        }}
        className="bm-filter-textarea bm-visible-scrollbar w-full resize-none overflow-y-auto whitespace-pre-wrap break-words border px-3 py-2 font-mono text-sm leading-6 text-[var(--bm-text)] outline-none focus:border-[var(--bm-accent)]/50"
      />
    </label>
  );
}
