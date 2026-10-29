import type { ReactNode } from 'react';
import { Link } from '@tanstack/react-router';
import { ChevronRight, FileArchive, Server } from 'lucide-react';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { AppTooltip } from '@/components/feedback/app-tooltip';
import { useNow } from '@/lib/clock';
import { cn } from '@/lib/utils';
import type {
  Alert,
  BackupExecution,
  BackupMethod,
  BackupTask,
  HostStatus,
  TaskStatus,
} from './types';

export const panelStyle = {
  background: 'var(--bm-card)',
  border: '1px solid var(--bm-border)',
  boxShadow: 'var(--shadow-card)',
} as const;

export function PageHeader({
  title,
  description,
  actions,
  className,
}: {
  title: string;
  description: string;
  actions?: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        'mb-5 flex shrink-0 flex-col gap-3 sm:flex-row sm:items-end sm:justify-between',
        className
      )}
    >
      <div>
        <h1 className="text-lg font-semibold text-[var(--bm-text)]">{title}</h1>
        <p className="mt-1 text-sm text-[var(--bm-text-muted)]">{description}</p>
      </div>
      {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
    </div>
  );
}

export function HostNameLink({
  hostID,
  hostname,
  className,
}: {
  hostID: string;
  hostname: string;
  className?: string;
}) {
  return (
    <Link
      to="/hosts/$hostID"
      params={{ hostID }}
      className={cn(
        'group inline-flex items-center gap-1.5 rounded-md bg-[var(--bm-control-bg-soft)] px-2.5 py-1.5 text-sm font-medium text-[var(--bm-accent-text)] transition hover:bg-[var(--bm-control-bg)]',
        className
      )}
    >
      <Server size={14} className="opacity-70" />
      <span>{hostname}</span>
      <ChevronRight
        size={13}
        className="-translate-x-1 opacity-0 transition group-hover:translate-x-0 group-hover:opacity-100"
      />
    </Link>
  );
}

export function TaskNameLink({
  taskID,
  name,
  className,
}: {
  taskID: string;
  name: string;
  className?: string;
}) {
  return (
    <Link
      to="/tasks/$taskID"
      params={{ taskID }}
      search={{}}
      className={cn(
        'group inline-flex items-center gap-1.5 rounded-md bg-[var(--bm-control-bg-soft)] px-2.5 py-1.5 text-sm font-medium text-[var(--bm-accent-text)] transition hover:bg-[var(--bm-control-bg)]',
        className
      )}
    >
      <FileArchive size={14} className="opacity-70" />
      <span>{name}</span>
      <ChevronRight
        size={13}
        className="-translate-x-1 opacity-0 transition group-hover:translate-x-0 group-hover:opacity-100"
      />
    </Link>
  );
}

export function Panel({
  title,
  extra,
  children,
  className = '',
  bodyClassName = '',
}: {
  title?: ReactNode;
  extra?: ReactNode;
  children: ReactNode;
  className?: string;
  bodyClassName?: string;
}) {
  return (
    <section
      className={`bm-card-hover flex min-h-0 flex-col overflow-hidden rounded-xl ${className}`}
      style={panelStyle}
    >
      {title || extra ? (
        <div className="flex shrink-0 items-center justify-between gap-3 border-b border-[var(--bm-border)] px-4 py-3">
          <div className="text-sm font-semibold text-[var(--bm-text)]">{title}</div>
          {extra}
        </div>
      ) : null}
      <div className={`min-h-0 flex-1 ${bodyClassName}`}>{children}</div>
    </section>
  );
}

const hostStatusMap: Record<HostStatus, { label: string; color: string; background: string }> = {
  normal: { label: '正常', color: '#34d399', background: 'rgba(16,185,129,0.12)' },
  abnormal: { label: '异常', color: '#fb923c', background: 'rgba(249,115,22,0.14)' },
  offline: { label: '离线', color: '#f87171', background: 'rgba(239,68,68,0.14)' },
};

const taskStatusMap: Record<TaskStatus, { label: string; color: string; background: string }> = {
  normal: { label: '正常', color: '#34d399', background: 'rgba(16,185,129,0.12)' },
  running: { label: '运行中', color: '#60a5fa', background: 'rgba(59,130,246,0.14)' },
  failed: { label: '失败', color: '#f87171', background: 'rgba(239,68,68,0.14)' },
  not_run: { label: '未执行', color: '#fbbf24', background: 'rgba(245,158,11,0.14)' },
  no_data: { label: '无数据', color: '#94a3b8', background: 'rgba(148,163,184,0.14)' },
};

const alertTypeMap: Record<Alert['type'], string> = {
  backup_failed: '备份失败',
  not_run: '未按时执行',
  report_timeout: '主机异常/离线',
};

const methodMap: Record<BackupMethod, string> = {
  veeam: 'Veeam',
  shell: 'Shell',
  powershell: 'PowerShell',
  batch: '批处理',
  exe: '程序',
};

const roleMap: Record<string, string> = {
  database: '数据库',
  web: 'Web',
  application: '应用',
  file: '文件',
  virtualization: '虚拟化',
};

export function Tag({
  tone,
  children,
}: {
  tone: 'success' | 'warning' | 'danger' | 'info' | 'muted' | 'violet' | 'orange' | 'test';
  children: ReactNode;
}) {
  const palette = {
    success: 'text-emerald-300 bg-emerald-500/10 border-emerald-500/25',
    warning: 'text-amber-300 bg-amber-500/10 border-amber-500/25',
    danger: 'text-red-300 bg-red-500/10 border-red-500/25',
    info: 'text-sky-300 bg-sky-500/10 border-sky-500/25',
    muted: 'text-slate-300 bg-slate-500/10 border-slate-500/25',
    violet: 'text-violet-300 bg-violet-500/10 border-violet-500/25',
    test: 'bm-tag-test',
    orange: 'text-orange-400 bg-orange-500/10 border-orange-500/30',
  }[tone];
  return (
    <span
      className={`inline-flex items-center gap-1 whitespace-nowrap rounded-md border px-2 py-0.5 text-xs ${palette}`}
    >
      {children}
    </span>
  );
}

export function HostStatusTag({ status }: { status: HostStatus }) {
  const meta = hostStatusMap[status] ?? hostStatusMap.offline;
  return (
    <span
      className="inline-flex min-w-[64px] items-center justify-center gap-1.5 whitespace-nowrap rounded-md px-2 py-0.5 text-xs"
      style={{ color: meta.color, background: meta.background }}
    >
      <span className="h-1.5 w-1.5 rounded-full" style={{ background: meta.color }} />
      {meta.label}
    </span>
  );
}

export function TaskStatusTag({
  status,
  showDot = true,
}: {
  status: TaskStatus;
  showDot?: boolean;
}) {
  const meta = taskStatusMap[status] ?? taskStatusMap.no_data;
  return (
    <span
      className="inline-flex min-w-[64px] items-center justify-center gap-1.5 whitespace-nowrap rounded-md px-2 py-0.5 text-xs"
      style={{ color: meta.color, background: meta.background }}
    >
      {showDot ? (
        <span className="h-1.5 w-1.5 rounded-full" style={{ background: meta.color }} />
      ) : null}
      {meta.label}
    </span>
  );
}

export function ExecutionStateTag({ state }: { state: BackupExecution['state'] }) {
  const map = {
    success: { label: '成功', tone: 'success' as const, color: '#34d399' },
    running: { label: '运行中', tone: 'info' as const, color: '#60a5fa' },
    failed: { label: '失败', tone: 'danger' as const, color: '#f87171' },
  }[state];
  return (
    <Tag tone={map.tone}>
      <span className="h-1.5 w-1.5 rounded-full" style={{ background: map.color }} />
      {map.label}
    </Tag>
  );
}

export function AlertStatusTag({ status }: { status: Alert['status'] }) {
  return status === 'active' ? <Tag tone="danger">未处理</Tag> : <Tag tone="success">已恢复</Tag>;
}

const methodToneMap: Record<BackupMethod, string> = {
  veeam: '#fbbf24',
  shell: '#38bdf8',
  powershell: '#a78bfa',
  batch: '#fb923c',
  exe: '#34d399',
};

export function MethodTag({ method }: { method: BackupMethod }) {
  const color = methodToneMap[method] ?? '#94a3b8';
  return (
    <span
      className="inline-flex items-center gap-1.5 whitespace-nowrap rounded-full border px-2 py-0.5 text-xs font-medium"
      style={{ color, borderColor: `${color}40`, background: `${color}14` }}
    >
      <span className="h-1.5 w-1.5 rounded-full bg-current" />
      {methodMap[method] ?? method}
    </span>
  );
}

export function alertTypeLabel(type: Alert['type']) {
  return alertTypeMap[type] ?? type;
}

const eventToneMap: Record<string, string> = {
  backup_failed: 'bm-event-tone-danger',
  not_run: 'bm-event-tone-warning',
  report_timeout: 'bm-event-tone-info',
};

export function EventTag({ event }: { event: string }) {
  const tone = eventToneMap[event] ?? 'bm-event-tone-neutral';
  return (
    <span
      className={`inline-flex items-center gap-1 whitespace-nowrap rounded-md border px-2 py-0.5 text-xs ${tone}`}
      style={{
        backgroundColor: 'rgba(var(--bm-event-rgb), 0.14)',
        borderColor: 'rgba(var(--bm-event-rgb), 0.35)',
        color: 'var(--bm-event-text)',
      }}
    >
      {alertTypeLabel(event as never)}
    </span>
  );
}

export function methodLabel(method: BackupMethod) {
  return methodMap[method] ?? method;
}

export function roleLabel(role?: string) {
  if (!role) return '-';
  return roleMap[role] ?? role;
}

export function platformLabel(platform?: string) {
  if (!platform) return '-';
  const normalized = platform.toLowerCase();
  if (normalized === 'linux') return 'Linux';
  if (normalized === 'windows') return 'Windows';
  return platform;
}

export function EmptyState({
  children,
  className = '',
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={`grid min-h-40 place-items-center px-4 text-sm text-[var(--bm-text-muted)] ${className}`}
    >
      {children}
    </div>
  );
}

export function formatTime(value?: string) {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  });
}

export function relativeTime(value?: string, now: number = Date.now()) {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  const diff = now - date.getTime();
  if (diff < 0) return '刚刚';
  const second = 1_000;
  const minute = 60_000;
  const hour = 60 * minute;
  const day = 24 * hour;
  if (diff < minute) return `${Math.floor(diff / second)} 秒前`;
  if (diff < hour) return `${Math.floor(diff / minute)} 分钟前`;
  if (diff < day) return `${Math.floor(diff / hour)} 小时前`;
  if (diff < 7 * day) return `${Math.floor(diff / day)} 天前`;
  return formatTime(value);
}

// RelativeTime 每秒重算相对时间，数据未刷新时文案也会继续推进
export function RelativeTime({ value }: { value?: string }) {
  const now = useNow();
  return <>{relativeTime(value, now)}</>;
}

export function formatBytes(size: number) {
  if (!size) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  let value = size;
  let index = 0;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }
  return `${value.toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}

const WEEKDAY_KEYS = ['MON', 'TUE', 'WED', 'THU', 'FRI', 'SAT', 'SUN'];
const WEEKDAY_LABELS = ['一', '二', '三', '四', '五', '六', '日'];
const CRON_WEEKDAY_LABELS = ['日', '一', '二', '三', '四', '五', '六'];

function englishScheduleLabel(value: string) {
  const weekly = value.match(/^Weekly\s+(.+?)\s+(\d{1,2}:\d{2})$/i);
  if (weekly) {
    const indexes = weekly[1]
      .split(',')
      .map(day => WEEKDAY_KEYS.indexOf(day.trim().toUpperCase()))
      .filter(index => index >= 0)
      .sort((left, right) => left - right);
    if (indexes.length === 0) return '';
    const contiguous = indexes.every((day, index) => index === 0 || day === indexes[index - 1] + 1);
    const label =
      contiguous && indexes.length > 1
        ? `${WEEKDAY_LABELS[indexes[0]]}至周${WEEKDAY_LABELS[indexes[indexes.length - 1]]}`
        : indexes.map(index => `周${WEEKDAY_LABELS[index]}`).join('、');
    return `每周${label} ${weekly[2]}`;
  }
  const daily = value.match(/^Daily\s+(\d{1,2}:\d{2})$/i);
  if (daily) return `每天 ${daily[1]}`;
  return '';
}

function cronWeekdayLabel(field: string) {
  if (!/^\d+(?:-\d+)?(?:,\d+(?:-\d+)?)*$/.test(field)) return '';
  const days = new Set<number>();
  for (const part of field.split(',')) {
    const [startText, endText] = part.split('-');
    const start = Number(startText);
    const end = endText === undefined ? start : Number(endText);
    if (!Number.isInteger(start) || !Number.isInteger(end) || start > end || start < 0 || end > 7) {
      return '';
    }
    for (let day = start; day <= end; day += 1) days.add(day === 7 ? 0 : day);
  }
  const indexes = [...days].sort((left, right) => left - right);
  if (indexes.length === 7) return '每天';
  if (indexes.length === 1) return `周${CRON_WEEKDAY_LABELS[indexes[0]]}`;
  const contiguous = indexes.every((day, index) => index === 0 || day === indexes[index - 1] + 1);
  if (contiguous) {
    return `周${CRON_WEEKDAY_LABELS[indexes[0]]}至周${CRON_WEEKDAY_LABELS[indexes[indexes.length - 1]]}`;
  }
  return indexes.map(day => `周${CRON_WEEKDAY_LABELS[day]}`).join('、');
}

export function cronScheduleLabel(value: string) {
  const parts = value.trim().split(/\s+/);
  if (parts.length !== 5) return '';
  const [minuteField, hourField, dayField, monthField, weekdayField] = parts;
  const minute = /^\d+$/.test(minuteField) ? Number(minuteField) : null;
  const hour = /^\d+$/.test(hourField) ? Number(hourField) : null;
  if (minute !== null && hour !== null) {
    const time = `${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}`;
    if (dayField !== '*' || monthField !== '*') {
      if (/^\d+$/.test(dayField) && (monthField === '*' || /^\d+$/.test(monthField))) {
        return `每月 ${Number(dayField)} 日 ${time}`;
      }
      return '';
    }
    if (weekdayField !== '*') {
      const weekday = cronWeekdayLabel(weekdayField);
      return weekday ? `每${weekday} ${time}` : '';
    }
    return `每天 ${time}`;
  }
  if (hourField === '*' && /^\*\/\d+$/.test(minuteField)) {
    return `每 ${Number(minuteField.slice(2))} 分钟`;
  }
  if (/^\*\/\d+$/.test(hourField) && minute !== null) {
    return `每 ${Number(hourField.slice(2))} 小时`;
  }
  return '';
}

export function taskSchedule(task: BackupTask) {
  const schedule = task.schedule?.trim() ?? '';
  if (schedule && /[\u4e00-\u9fa5]/.test(schedule)) return schedule;
  const english = englishScheduleLabel(schedule);
  if (english) return english;
  const cron =
    task.scheduleCron?.trim() || (/^\S+\s+\S+\s+\S+\s+\S+\s+\S+$/.test(schedule) ? schedule : '');
  if (cron) {
    const label = cronScheduleLabel(cron);
    if (label) return label;
  }
  return schedule || '-';
}

export type SelectOption = { value: string; label: string; hint?: string };

// BackupSelect 统一使用平台下拉框样式，避免各页面原生 select 观感不一致。
export function BackupSelect({
  value,
  onValueChange,
  options,
  className,
  placeholder,
  ariaLabel,
  disabled,
  clearable = false,
  footerOption,
  contentClassName,
}: {
  value: string;
  onValueChange: (value: string) => void;
  options: SelectOption[];
  className?: string;
  placeholder?: string;
  ariaLabel?: string;
  disabled?: boolean;
  clearable?: boolean;
  footerOption?: SelectOption;
  contentClassName?: string;
}) {
  return (
    <Select value={value} onValueChange={onValueChange} disabled={disabled}>
      <SelectTrigger
        className={cn(className, value && 'text-[var(--bm-accent-text)]')}
        aria-label={ariaLabel}
        disabled={disabled}
        onClear={clearable && value ? () => onValueChange('') : undefined}
        clearLabel={`清除${ariaLabel ?? '选择'}`}
      >
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent className={contentClassName}>
        {options.map(option => (
          <SelectItem key={option.value} value={option.value}>
            {option.hint ? (
              <AppTooltip label={option.hint} placement="right">
                <span>{option.label}</span>
              </AppTooltip>
            ) : (
              option.label
            )}
          </SelectItem>
        ))}
        {footerOption ? (
          <>
            <SelectSeparator className="my-1 bg-[var(--bm-border)]" />
            <SelectItem value={footerOption.value}>{footerOption.label}</SelectItem>
          </>
        ) : null}
      </SelectContent>
    </Select>
  );
}
