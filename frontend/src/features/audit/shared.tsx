import { LoaderCircle, RefreshCcw, Search } from 'lucide-react';
import type { ReactNode } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

export function SearchBox({
  value,
  onChange,
  placeholder = '搜索',
  className = 'w-full sm:w-72',
}: {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
}) {
  return (
    <div className={`relative ${className}`}>
      <Search
        size={15}
        className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--bm-text-muted)]"
      />
      <Input
        value={value}
        onChange={event => onChange(event.target.value)}
        placeholder={placeholder}
        className="pl-9"
      />
    </div>
  );
}

export function QueryState({
  loading,
  error,
  onRetry,
  children,
}: {
  loading: boolean;
  error: boolean;
  onRetry: () => void;
  children: ReactNode;
}) {
  if (loading) {
    return (
      <div className="flex min-h-52 items-center justify-center gap-2 text-sm text-[var(--bm-text-muted)]">
        <LoaderCircle size={18} className="animate-spin" />
        正在加载数据
      </div>
    );
  }
  if (error) {
    return (
      <div className="flex min-h-52 flex-col items-center justify-center gap-3 text-sm text-[var(--bm-text-muted)]">
        <span>数据加载失败</span>
        <Button size="sm" variant="outline" onClick={onRetry}>
          <RefreshCcw size={14} />
          重新加载
        </Button>
      </div>
    );
  }
  return <>{children}</>;
}
