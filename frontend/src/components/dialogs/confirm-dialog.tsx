import { AlertTriangle, CheckCircle2, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';

export function ConfirmDialog({
  open,
  title,
  description,
  detail,
  confirmText = '确认删除',
  horizontalHeader = false,
  tone = 'danger',
  icon = 'alert',
  softDestructive = false,
  confirmClassName,
  busy,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  title: string;
  description: string;
  detail?: string;
  confirmText?: string;
  horizontalHeader?: boolean;
  tone?: 'danger' | 'success';
  icon?: 'alert' | 'delete';
  softDestructive?: boolean;
  confirmClassName?: string;
  busy?: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  const isSuccess = tone === 'success';
  const Icon = isSuccess ? CheckCircle2 : icon === 'delete' ? Trash2 : AlertTriangle;
  const iconClassName = isSuccess
    ? 'bg-emerald-500/10 text-emerald-500'
    : 'bg-red-500/10 text-red-500';

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="bm-dialog-panel sm:max-w-md">
        {horizontalHeader ? (
          <DialogHeader className="space-y-0">
            <div className="flex items-start gap-3">
              <span
                className={`grid h-10 w-10 shrink-0 place-items-center rounded-xl ${iconClassName}`}
              >
                <Icon size={20} aria-hidden="true" />
              </span>
              <div className="min-w-0 space-y-2">
                <DialogTitle>{title}</DialogTitle>
                <DialogDescription className="leading-5">{description}</DialogDescription>
              </div>
            </div>
            {detail ? (
              <DialogDescription className="pt-4 leading-6 text-[var(--bm-text)]">
                {detail}
              </DialogDescription>
            ) : null}
          </DialogHeader>
        ) : (
          <DialogHeader>
            <span className={`mb-3 grid h-10 w-10 place-items-center rounded-xl ${iconClassName}`}>
              <Icon size={20} aria-hidden="true" />
            </span>
            <DialogTitle>{title}</DialogTitle>
            <DialogDescription>{description}</DialogDescription>
            {detail ? (
              <DialogDescription className="pt-2 leading-6 text-[var(--bm-text)]">
                {detail}
              </DialogDescription>
            ) : null}
          </DialogHeader>
        )}
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            type="button"
            variant={isSuccess || softDestructive ? 'outline' : 'destructive'}
            className={
              isSuccess
                ? 'bm-success-soft-button'
                : softDestructive
                  ? 'bm-danger-soft-button'
                  : confirmClassName
            }
            disabled={busy}
            onClick={onConfirm}
          >
            {busy ? '处理中...' : confirmText}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
