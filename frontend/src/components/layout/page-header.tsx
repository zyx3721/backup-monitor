import type { ReactNode } from 'react';

export function PageHeader({
  title,
  description,
  actions,
}: {
  title: string;
  description: string;
  actions?: ReactNode;
}) {
  return (
    <div className="mb-5 flex shrink-0 flex-wrap items-start justify-between gap-4">
      <div className="min-w-0">
        <h1 className="text-lg font-semibold text-[var(--bm-text)]">{title}</h1>
        <p className="mt-1 text-sm text-[var(--bm-text-muted)]">{description}</p>
      </div>
      {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
    </div>
  );
}
