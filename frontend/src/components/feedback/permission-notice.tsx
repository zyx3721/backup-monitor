import { ShieldAlert } from 'lucide-react';

// PermissionNotice 缺少集合查看权限时的整页占位，阻断页面渲染与数据请求
export function PermissionNotice({ section }: { section: string }) {
  return (
    <div
      className="grid h-full min-h-[50vh] place-items-center rounded-xl border border-[var(--bm-border)] bg-[var(--bm-card)] p-6"
      style={{ boxShadow: 'var(--shadow-card)' }}
    >
      <span className="flex max-w-xl items-center gap-2 text-sm text-[var(--bm-text-muted)]">
        <ShieldAlert size={16} className="shrink-0 text-amber-300" aria-hidden="true" />
        暂无{section}查看权限，如需查看请联系管理员分配
      </span>
    </div>
  );
}
