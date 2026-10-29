import type { ReactNode } from 'react';

export function ExecutionPair({
  label,
  value,
  rightLabel,
  rightValue,
}: {
  label: string;
  value: ReactNode;
  rightLabel: string;
  rightValue: ReactNode;
}) {
  return (
    <div className="grid grid-cols-[110px_minmax(0,1fr)] border-b border-[var(--bm-border)] last:border-b-0 sm:grid-cols-[130px_minmax(0,1fr)_130px_minmax(0,1fr)]">
      <div className="border-r border-[var(--bm-border)] bg-[var(--bm-control-bg-soft)] px-4 py-3 text-[var(--bm-text-muted)]">
        {label}
      </div>
      <div className="min-w-0 break-all px-4 py-3 text-[var(--bm-text)]">{value}</div>
      <div className="border-r border-t border-[var(--bm-border)] bg-[var(--bm-control-bg-soft)] px-4 py-3 text-[var(--bm-text-muted)] sm:border-t-0">
        {rightLabel}
      </div>
      <div className="min-w-0 break-all border-t border-[var(--bm-border)] px-4 py-3 text-[var(--bm-text)] sm:border-t-0">
        {rightValue}
      </div>
    </div>
  );
}

export function ExecutionWideRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="grid grid-cols-[110px_minmax(0,1fr)] border-b border-[var(--bm-border)] last:border-b-0 sm:grid-cols-[130px_minmax(0,1fr)_130px_minmax(0,1fr)]">
      <div className="border-r border-[var(--bm-border)] bg-[var(--bm-control-bg-soft)] px-4 py-3 text-[var(--bm-text-muted)]">
        {label}
      </div>
      <div className="min-w-0 break-all px-4 py-3 text-[var(--bm-text)] sm:col-span-3">{value}</div>
    </div>
  );
}
