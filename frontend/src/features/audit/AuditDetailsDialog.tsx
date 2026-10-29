import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import type { AuditEntry } from './AuditPage';
import { formatAuditTime } from './AuditPage';

export function AuditDetailsDialog({
  item,
  moduleLabel,
  onOpenChange,
}: {
  item: AuditEntry | null;
  moduleLabel: (module: string) => string;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog open={item !== null} onOpenChange={onOpenChange}>
      <DialogContent className="bm-dialog-panel gap-0 overflow-hidden p-0 sm:max-w-[600px]">
        <DialogHeader className="border-b border-[var(--bm-border)] px-5 py-4">
          <DialogTitle>审计详情</DialogTitle>
          <DialogDescription>
            {item ? `${formatAuditTime(item.timestamp, true)} · ${item.username || '-'}` : ''}
          </DialogDescription>
        </DialogHeader>
        {item ? (
          <div className="space-y-5 p-5">
            <div className="grid gap-4 sm:grid-cols-2">
              <DetailItem label="操作" value={item.action || '-'} />
              <DetailItem label="模块" value={moduleLabel(item.module)} />
              <DetailItem
                label="目标"
                value={item.target || '-'}
                valueClassName="text-emerald-700 dark:text-emerald-300"
              />
              <DetailItem label="来源 IP" value={item.ip || '-'} mono />
              <DetailItem
                label="结果"
                value={item.result === 'failure' ? '失败' : '成功'}
                valueClassName={
                  item.result === 'failure'
                    ? 'text-red-600 dark:text-red-300'
                    : 'text-emerald-600 dark:text-emerald-300'
                }
              />
              <DetailItem label="用户" value={item.username || '-'} />
            </div>
            <div className="rounded-xl border border-[var(--bm-border)] bg-[var(--bm-control-bg-soft)]/60 p-4">
              <p className="text-xs font-semibold text-[var(--bm-text-muted)]">详情</p>
              <p className="mt-2 break-words text-sm leading-6 text-[var(--bm-text)]">
                {item.detail || '无详情'}
              </p>
            </div>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function DetailItem({
  label,
  value,
  mono = false,
  valueClassName = '',
}: {
  label: string;
  value: string;
  mono?: boolean;
  valueClassName?: string;
}) {
  return (
    <div className="min-w-0 rounded-lg border border-[var(--bm-border)] bg-[var(--bm-control-bg)] px-3 py-2.5">
      <p className="text-xs text-[var(--bm-text-muted)]">{label}</p>
      <p
        className={`mt-1 truncate text-sm ${valueClassName || 'text-[var(--bm-text)]'}${mono ? ' font-mono' : ''}`}
      >
        {value}
      </p>
    </div>
  );
}
