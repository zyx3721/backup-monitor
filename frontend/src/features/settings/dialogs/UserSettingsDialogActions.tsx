import { Ban, CircleCheck, Save, Trash2 } from 'lucide-react';

export function DangerButton({
  label,
  busy,
  disabled = false,
  onClick,
}: {
  label: string;
  busy: boolean;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      disabled={busy || disabled}
      onClick={() => void onClick()}
      className="bm-action-button flex items-center gap-2 rounded-lg border px-3 py-2 text-sm disabled:cursor-not-allowed disabled:opacity-50"
      style={{
        borderColor: 'rgba(239,68,68,0.34)',
        color: '#f87171',
        background: 'rgba(239,68,68,0.08)',
      }}
    >
      <Trash2 size={14} />
      {label}
    </button>
  );
}

export function PrimaryButton({
  label,
  busy,
  disabled = false,
  onClick,
}: {
  label: string;
  busy: boolean;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      disabled={busy || disabled}
      onClick={() => void onClick()}
      className="bm-action-button flex items-center gap-2 rounded-lg border px-3 py-2 text-sm disabled:cursor-not-allowed disabled:opacity-50"
      style={{
        borderColor: 'rgba(59,130,246,0.38)',
        color: 'var(--bm-accent-text)',
        background: 'rgba(59,130,246,0.1)',
      }}
    >
      <Save size={14} />
      {busy ? '保存中' : label}
    </button>
  );
}

export function StateButton({
  active,
  disabled,
  onClick,
}: {
  active: boolean;
  disabled: boolean;
  onClick: () => void;
}) {
  const Icon = active ? CircleCheck : Ban;
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={() => void onClick()}
      className="bm-action-button flex items-center gap-2 rounded-lg border px-3 py-2 text-sm disabled:cursor-not-allowed disabled:opacity-50"
      style={{
        borderColor: active ? 'rgba(34,197,94,0.42)' : 'rgba(245,158,11,0.36)',
        color: active ? '#86efac' : '#fbbf24',
        background: active ? 'rgba(34,197,94,0.1)' : 'rgba(245,158,11,0.09)',
      }}
    >
      <Icon size={14} />
      {active ? '启用' : '禁用'}
    </button>
  );
}
