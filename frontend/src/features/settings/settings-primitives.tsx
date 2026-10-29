import { Eye, EyeOff, ToggleLeft, ToggleRight } from 'lucide-react';
import { useState, type ReactNode } from 'react';
import { AppTooltip } from '@/components/feedback/app-tooltip';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

export type SettingsField = {
  key: string;
  label: string;
  placeholder?: string;
  required?: boolean;
  helper?: string;
  labelHint?: string;
  type?: 'text' | 'password' | 'number' | 'checkbox' | 'textarea' | 'select';
  inputMode?: React.HTMLAttributes<HTMLInputElement>['inputMode'];
  options?: Array<{ label: string; value: string }>;
};

export function SettingsSplitLayout({
  sidebarLabel,
  sidebar,
  children,
}: {
  sidebarLabel: string;
  sidebar: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="grid min-h-0 flex-1 grid-cols-1 items-stretch gap-4 overflow-hidden xl:grid-cols-[430px_minmax(0,1fr)]">
      <nav
        className="bm-hidden-scrollbar max-h-64 min-h-0 space-y-2 overflow-y-auto overscroll-contain rounded-lg border border-[var(--bm-border)] bg-white/[0.035] p-2 xl:max-h-none"
        aria-label={sidebarLabel}
      >
        {sidebar}
      </nav>
      {children}
    </div>
  );
}

export function SettingsDetailPanel({
  header,
  actions,
  children,
}: {
  header: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <aside className="flex min-h-0 flex-col overflow-hidden rounded-lg border border-[var(--bm-border)] bg-white/[0.035] p-4">
      {header}
      <div className="bm-hidden-scrollbar min-h-0 flex-1 overflow-y-auto overscroll-contain pt-1 pr-2">
        {children}
      </div>
      {actions ? (
        <div className="mt-5 flex shrink-0 flex-wrap justify-end gap-2 border-t border-[var(--bm-border)] pt-4">
          {actions}
        </div>
      ) : null}
    </aside>
  );
}

export function SettingsDetailHeader({
  icon: Icon,
  color,
  title,
  subtitle,
  active,
}: {
  icon: React.ElementType;
  color: string;
  title: string;
  subtitle: string;
  active?: boolean;
}) {
  return (
    <div className="mb-4 flex items-center gap-3">
      <div
        className="flex h-10 w-10 items-center justify-center rounded-lg border border-white/[0.08] bg-white/[0.05]"
        style={{ color }}
      >
        <Icon size={19} />
      </div>
      <div className="min-w-0">
        <div className="truncate text-sm font-semibold text-[var(--bm-text)]">{title}</div>
        <div
          className="mt-0.5 truncate text-xs"
          style={{ color: active ? '#86efac' : 'var(--bm-text-muted)' }}
        >
          {subtitle}
        </div>
      </div>
    </div>
  );
}

// UsageIndicator 认证与找密侧栏的启用状态图标，样式与认证配置保持一致
export function UsageIndicator({ enabled }: { enabled: boolean }) {
  return enabled ? (
    <ToggleRight size={18} aria-hidden="true" className="shrink-0 text-emerald-400" />
  ) : (
    <ToggleLeft size={18} aria-hidden="true" className="shrink-0 text-[var(--bm-text-muted)]" />
  );
}

export function EnableToggle({
  enabled,
  disabled,
  onChange,
  label,
  enabledText,
  disabledText,
}: {
  enabled: boolean;
  disabled?: boolean;
  onChange: (value: boolean) => void;
  label: string;
  enabledText: string;
  disabledText: string;
}) {
  return (
    <label
      className={`group flex min-h-14 items-center justify-between gap-3 rounded-lg px-3 py-2 text-sm transition-all duration-200 ${
        disabled
          ? 'cursor-not-allowed opacity-75'
          : 'cursor-pointer hover:-translate-y-0.5 active:translate-y-0 active:scale-[0.99]'
      }`}
      style={{
        background: enabled ? 'rgba(16,185,129,0.08)' : 'rgba(255,255,255,0.035)',
        border: enabled ? '1px solid rgba(16,185,129,0.28)' : '1px solid var(--bm-border)',
        boxShadow: enabled ? '0 10px 24px rgba(16,185,129,0.08)' : 'none',
        color: 'var(--bm-text)',
      }}
    >
      <input
        type="checkbox"
        className="peer sr-only"
        checked={enabled}
        disabled={disabled}
        onChange={event => onChange(event.target.checked)}
      />
      <span className="min-w-0">
        <span className="block font-medium">{label}</span>
        <span className="mt-0.5 block text-xs text-[var(--bm-text-muted)]">
          {enabled ? enabledText : disabledText}
        </span>
      </span>
      <span
        aria-hidden="true"
        className="relative inline-flex h-6 w-11 shrink-0 rounded-full transition-all duration-200 peer-focus-visible:ring-2 peer-focus-visible:ring-emerald-300"
        style={{
          background: enabled ? 'rgba(16,185,129,0.95)' : 'rgba(148,163,184,0.28)',
          border: enabled ? '1px solid rgba(134,239,172,0.4)' : '1px solid rgba(148,163,184,0.32)',
        }}
      >
        <span
          className={`absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white shadow-sm transition-transform duration-200 ${enabled ? 'translate-x-5' : 'translate-x-0'}`}
        />
      </span>
    </label>
  );
}

export function ConfigField({
  field,
  value,
  onChange,
  secretConfigured = false,
  disabled = false,
}: {
  field: SettingsField;
  value: unknown;
  onChange: (value: unknown) => void;
  secretConfigured?: boolean;
  disabled?: boolean;
}) {
  const [passwordVisible, setPasswordVisible] = useState(false);
  if (field.type === 'checkbox') {
    return (
      <label
        className={`flex items-start gap-2 text-sm ${disabled ? 'cursor-not-allowed opacity-70' : 'cursor-pointer'}`}
        style={{ color: 'var(--bm-text-muted)' }}
      >
        <input
          type="checkbox"
          disabled={disabled}
          className="mt-1 cursor-pointer disabled:cursor-not-allowed"
          checked={Boolean(value)}
          onChange={event => onChange(event.target.checked)}
        />
        <span>
          <span className="block">{field.label}</span>
          {field.helper ? (
            <span className="mt-0.5 block text-[11px] leading-4">{field.helper}</span>
          ) : null}
        </span>
      </label>
    );
  }
  const inputType =
    field.type === 'password'
      ? passwordVisible
        ? 'text'
        : 'password'
      : field.type === 'number'
        ? 'number'
        : 'text';
  const textValue = String(value ?? '');
  const placeholder =
    field.type === 'password' && secretConfigured ? '已配置，留空表示不修改' : field.placeholder;
  return (
    <label className="block space-y-1.5 text-xs" style={{ color: 'var(--bm-text-muted)' }}>
      <span className="flex items-center gap-1">
        {field.labelHint ? (
          <AppTooltip label={field.labelHint} placement="top">
            <span className="cursor-help underline decoration-dotted underline-offset-4">
              {field.label}
            </span>
          </AppTooltip>
        ) : (
          field.label
        )}
        {field.required ? <span style={{ color: '#f87171' }}>*</span> : null}
      </span>
      {field.type === 'select' ? (
        <Select value={textValue} disabled={disabled} onValueChange={onChange}>
          <SelectTrigger aria-label={field.label}>
            <SelectValue placeholder={field.placeholder} />
          </SelectTrigger>
          <SelectContent>
            {(field.options ?? []).map(option => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      ) : field.type === 'textarea' ? (
        <textarea
          value={textValue}
          disabled={disabled}
          onChange={event => onChange(event.target.value)}
          placeholder={field.placeholder}
          rows={3}
          className="bm-hidden-scrollbar w-full resize-y rounded-lg px-3 py-2 text-sm outline-none disabled:opacity-60"
          style={{
            background: 'var(--bm-control-bg)',
            border: '1px solid var(--bm-border)',
            color: 'var(--bm-text)',
          }}
        />
      ) : (
        <span className="relative block">
          <input
            value={textValue}
            disabled={disabled}
            onChange={event =>
              onChange(field.type === 'number' ? Number(event.target.value) : event.target.value)
            }
            placeholder={placeholder}
            type={inputType}
            inputMode={field.inputMode}
            className={`w-full rounded-lg px-3 py-2 text-sm outline-none disabled:opacity-60 ${field.type === 'password' ? 'pr-10' : ''} ${field.type === 'number' ? 'bm-number-input' : ''}`}
            style={{
              background: 'var(--bm-control-bg)',
              border: '1px solid var(--bm-border)',
              color: 'var(--bm-text)',
            }}
          />
          {field.type === 'password' ? (
            <AppTooltip label={passwordVisible ? '隐藏密码' : '显示密码'} placement="top">
              <button
                type="button"
                onClick={event => {
                  event.preventDefault();
                  setPasswordVisible(current => !current);
                }}
                className="bm-action-button absolute right-2 top-1/2 flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-md disabled:opacity-40"
                aria-label={passwordVisible ? '隐藏密码' : '显示密码'}
              >
                {passwordVisible ? <EyeOff size={15} /> : <Eye size={15} />}
              </button>
            </AppTooltip>
          ) : null}
        </span>
      )}
      {field.helper ? <span className="block text-[11px] leading-4">{field.helper}</span> : null}
    </label>
  );
}

export function NumberControl({
  label,
  description,
  unit,
  value,
  min,
  max,
  step = 1,
  disabled,
  onChange,
}: {
  label: string;
  description?: string;
  unit: string;
  value: number;
  min: number;
  max: number;
  step?: number;
  disabled?: boolean;
  onChange: (value: number) => void;
}) {
  const normalized = Math.min(max, Math.max(min, Number(value) || min));
  const progress = max === min ? 100 : ((normalized - min) / (max - min)) * 100;
  return (
    <label
      className="rounded-xl border p-4"
      style={{
        borderColor: 'var(--bm-border)',
        background: 'rgba(255,255,255,0.026)',
        color: 'var(--bm-text-muted)',
      }}
    >
      <span className="flex items-center justify-between gap-3 text-sm font-semibold text-[var(--bm-text)]">
        <span>{label}</span>
        <span className="shrink-0 rounded-md border border-blue-400/30 bg-blue-500/[0.08] px-2 py-0.5 text-xs text-[var(--bm-accent-text)]">
          {normalized} {unit}
        </span>
      </span>
      {description ? <span className="mt-2 block text-xs">{description}</span> : null}
      <input
        type="range"
        min={min}
        max={max}
        step={step}
        value={normalized}
        disabled={disabled}
        onChange={event => onChange(Number(event.target.value))}
        className="bm-flow-range mt-4 w-full disabled:opacity-60"
        style={{ '--bm-range-progress': `${progress}%` } as React.CSSProperties}
      />
      <div className="mt-3 flex items-center gap-3">
        <input
          type="number"
          min={min}
          max={max}
          step={step}
          value={normalized}
          disabled={disabled}
          onChange={event =>
            onChange(Math.min(max, Math.max(min, Number(event.target.value) || min)))
          }
          className="h-9 w-24 rounded-lg border border-[var(--bm-border)] bg-[var(--bm-control-bg)] px-3 text-sm text-[var(--bm-text)] outline-none disabled:opacity-60"
        />
        <span className="text-xs">
          范围 {min}-{max} {unit}
        </span>
      </div>
    </label>
  );
}

export function ActionButton({
  icon,
  label,
  busy,
  disabled,
  tone = 'primary',
  onClick,
}: {
  icon: ReactNode;
  label: string;
  busy?: boolean;
  disabled?: boolean;
  tone?: 'primary' | 'success' | 'danger' | 'muted';
  onClick: () => void | Promise<void>;
}) {
  const palette = {
    primary: ['rgba(59,130,246,0.38)', 'var(--bm-accent-text)', 'rgba(59,130,246,0.1)'],
    success: ['rgba(16,185,129,0.35)', '#34d399', 'rgba(16,185,129,0.08)'],
    danger: ['rgba(239,68,68,0.34)', '#f87171', 'rgba(239,68,68,0.08)'],
    muted: ['var(--bm-border)', 'var(--bm-text-muted)', 'rgba(255,255,255,0.035)'],
  }[tone];
  return (
    <button
      type="button"
      disabled={busy || disabled}
      onClick={() => void onClick()}
      className={`bm-action-button flex items-center gap-2 rounded-lg border px-3 py-2 text-sm disabled:cursor-not-allowed disabled:opacity-50 ${tone === 'danger' ? 'bm-danger-button' : ''}`}
      style={{ borderColor: palette[0], color: palette[1], background: palette[2] }}
    >
      {icon}
      {busy ? '处理中' : label}
    </button>
  );
}
