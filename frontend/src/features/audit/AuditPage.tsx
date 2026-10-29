import { useEffect, useMemo, useState } from 'react';
import { CheckCircle2, Download, Search, Trash2, XCircle } from 'lucide-react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { AppTooltip } from '@/components/feedback/app-tooltip';
import { ConfirmDialog } from '@/components/dialogs/confirm-dialog';
import { api } from '@/lib/api';
import { userHasPermission } from '@/lib/permissions';
import { getStoredUser } from '@/lib/session';
import { showErrorToast } from '@/lib/toast-errors';
import { PERM } from '@/lib/permissions';
import { DataTableShell } from '@/components/data/data-table-shell';
import { EmptyState } from '@/components/data/empty-state';
import { PageHeader } from '@/components/layout/page-header';
import { Pagination } from '@/components/data/pagination';
import { BackupSelect } from '@/features/backup/components';
import { QueryState, SearchBox } from './shared';
import { DateRangePicker, type DateRange } from './DateRangePicker';
import { AuditDetailsDialog } from './AuditDetailsDialog';
import { AuditExportDialog } from './AuditExportDialog';

export type AuditEntry = {
  id: number;
  timestamp: number;
  username: string;
  module: string;
  action: string;
  target: string;
  ip: string;
  result: string;
  detail: string;
};

type AuditHistoryResponse = { items: AuditEntry[]; total: number; limit: number };

const moduleLabels: Record<string, string> = {
  auth: '身份认证',
  hosts: '备份主机',
  tasks: '备份任务',
  executions: '执行记录',
  alerts: '告警中心',
  audit: '审计日志',
  notify: '通知管理',
  agents: 'Agent 管理',
  settings: '系统配置',
};
const auditModules = Object.entries(moduleLabels);

export function moduleLabel(module: string) {
  return moduleLabels[module] ?? (module || '其他');
}

export function formatAuditTime(timestamp: number, includeSeconds = false) {
  if (!timestamp) return '-';
  const date = new Date(timestamp * 1000);
  if (Number.isNaN(date.valueOf())) return '-';
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    ...(includeSeconds ? { second: '2-digit' } : {}),
  }).format(date);
}

export function AuditPage() {
  const canManageAudit = userHasPermission(getStoredUser(), PERM.auditManage);
  const query = useQuery({
    queryKey: ['backup-monitor', 'audits'],
    queryFn: () => api<AuditHistoryResponse>('/api/history'),
  });
  const [search, setSearch] = useState('');
  const [module, setModule] = useState('');
  const [result, setResult] = useState('');
  const [range, setRange] = useState<DateRange>({
    start: '',
    end: '',
    startTime: '',
    endTime: '',
  });
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState<number>(18);
  const [detailItem, setDetailItem] = useState<AuditEntry | null>(null);
  const [exportOpen, setExportOpen] = useState(false);
  const [clearOpen, setClearOpen] = useState(false);
  const items = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    const startBound = rangeStartBound(range.start, range.startTime);
    const endBound = rangeEndBound(range.end, range.endTime);
    return (query.data?.items ?? []).filter(
      item =>
        (module === '' || item.module === module) &&
        (result === '' || item.result === result) &&
        (startBound === null || item.timestamp >= startBound) &&
        (endBound === null || item.timestamp <= endBound) &&
        (!keyword ||
          `${formatAuditTime(item.timestamp)} ${item.username} ${moduleLabel(item.module)} ${item.action} ${item.target} ${item.ip} ${item.result === 'failure' ? '失败' : '成功'} ${item.detail}`
            .toLowerCase()
            .includes(keyword))
    );
  }, [module, query.data?.items, range, result, search]);
  const effectivePageSize = pageSize === -1 ? Math.max(items.length, 1) : pageSize;
  const pageCount = Math.max(1, Math.ceil(items.length / effectivePageSize));
  const currentPage = Math.min(page, pageCount);
  const pageItems = items.slice(
    (currentPage - 1) * effectivePageSize,
    currentPage * effectivePageSize
  );

  useEffect(() => setPage(1), [module, pageSize, result, search, range]);
  useEffect(() => {
    if (page > pageCount) setPage(pageCount);
  }, [page, pageCount]);

  function openExport() {
    if (items.length === 0) {
      toast.error('没有可导出的审计日志');
      return;
    }
    setExportOpen(true);
  }

  const clearMutation = useMutation({
    mutationFn: () =>
      api<{ deleted: number }>('/api/history/clear', {
        method: 'POST',
        body: JSON.stringify({ ids: items.map(item => item.id) }),
      }),
    onSuccess: response => {
      setClearOpen(false);
      toast.success(`已清空 ${response.deleted} 条审计日志`);
      void query.refetch();
    },
    onError: error => {
      showErrorToast(error, '清空审计日志失败');
    },
  });

  function openClear() {
    if (items.length === 0) {
      toast.error('没有可清空的审计日志');
      return;
    }
    setClearOpen(true);
  }

  return (
    <div className="bm-page-fill">
      <PageHeader title="审计日志" description="追踪身份认证、用户配置、系统配置与安全事件操作" />
      <DataTableShell
        className="bm-page-card"
        contentClassName="bm-scroll-area flex-1"
        toolbarClassName="flex-nowrap overflow-x-auto bm-hidden-scrollbar"
        footer={
          items.length > 0 ? (
            <Pagination
              page={currentPage}
              pageCount={pageCount}
              pageSize={pageSize}
              total={items.length}
              onPageChange={setPage}
              onPageSizeChange={setPageSize}
            />
          ) : null
        }
        toolbar={
          <>
            <SearchBox
              value={search}
              onChange={setSearch}
              placeholder="搜索审计日志"
              className="min-w-64 flex-1"
            />
            <div className="w-96 shrink-0">
              <DateRangePicker value={range} onChange={setRange} />
            </div>
            <div className="w-36 shrink-0">
              <BackupSelect
                value={module}
                onValueChange={setModule}
                ariaLabel="模块"
                placeholder="模块"
                clearable
                options={auditModules.map(([value, label]) => ({ value, label }))}
              />
            </div>
            <div className="w-36 shrink-0">
              <BackupSelect
                value={result}
                onValueChange={setResult}
                ariaLabel="结果"
                placeholder="结果"
                clearable
                options={[
                  { value: 'success', label: '成功' },
                  { value: 'failure', label: '失败' },
                ]}
              />
            </div>
            {canManageAudit && (
              <Button
                variant="outline"
                className="bm-audit-export-button ml-auto shrink-0"
                onClick={openExport}
              >
                <Download size={16} />
                导出
              </Button>
            )}
            {canManageAudit && (
              <Button
                variant="outline"
                className="bm-danger-soft-button shrink-0"
                onClick={openClear}
                disabled={clearMutation.isPending}
              >
                <Trash2 size={16} />
                清空
              </Button>
            )}
          </>
        }
      >
        <QueryState
          loading={query.isLoading}
          error={query.isError}
          onRetry={() => void query.refetch()}
        >
          {items.length === 0 ? (
            <EmptyState title="没有匹配的审计日志" />
          ) : (
            <div className="min-w-[1200px] divide-y divide-[var(--bm-border)]">
              <div className="sticky top-0 z-10 grid grid-cols-[160px_120px_120px_150px_minmax(150px,1fr)_140px_100px_100px] gap-4 bg-[var(--bm-table-head-bg)] px-4 py-3 text-center text-xs font-semibold text-[var(--bm-text-muted)]">
                <span>时间</span>
                <span>用户</span>
                <span>模块</span>
                <span>操作</span>
                <span>目标</span>
                <span>来源 IP</span>
                <span>结果</span>
                <span>详情</span>
              </div>
              {pageItems.map(item => (
                <div
                  key={item.id}
                  className="grid grid-cols-[160px_120px_120px_150px_minmax(150px,1fr)_140px_100px_100px] items-center gap-4 px-4 py-3.5 text-sm hover:bg-[var(--bm-control-bg-soft)]"
                >
                  <time className="justify-self-center whitespace-nowrap text-sm font-medium text-[var(--bm-text)]">
                    {formatAuditTime(item.timestamp)}
                  </time>
                  <span className="justify-self-center truncate font-medium">
                    {item.username || '-'}
                  </span>
                  <span
                    className={`justify-self-center truncate rounded-full px-2.5 py-1 text-xs font-medium ${moduleBadgeClassName(item.module)}`}
                  >
                    {moduleLabel(item.module)}
                  </span>
                  <span className="justify-self-center truncate font-medium">
                    {item.action || '-'}
                  </span>
                  <span className="justify-self-center min-w-0 max-w-full">
                    <AppTooltip
                      label={item.target}
                      wrap
                      disabled={!item.target}
                      className="block min-w-0 max-w-full"
                    >
                      <span className="block truncate font-medium text-emerald-700 dark:text-emerald-300">
                        {item.target || '-'}
                      </span>
                    </AppTooltip>
                  </span>
                  <span className="justify-self-center truncate font-medium">{item.ip || '-'}</span>
                  <AuditResultBadge result={item.result} />
                  <Button
                    variant="ghost"
                    size="sm"
                    className="bm-audit-view-button justify-self-center px-2"
                    onClick={() => setDetailItem(item)}
                    aria-label="查看审计详情"
                  >
                    <Search size={15} />
                    查看
                  </Button>
                </div>
              ))}
            </div>
          )}
        </QueryState>
      </DataTableShell>
      <AuditDetailsDialog
        item={detailItem}
        moduleLabel={moduleLabel}
        onOpenChange={open => !open && setDetailItem(null)}
      />
      <AuditExportDialog
        open={exportOpen}
        items={items}
        moduleLabel={moduleLabel}
        onOpenChange={setExportOpen}
      />
      <ConfirmDialog
        open={clearOpen}
        title="清空日志记录"
        description="清空后不可恢复"
        detail="确定清空当前审计日志记录吗？"
        confirmText="确认清空"
        horizontalHeader
        softDestructive
        busy={clearMutation.isPending}
        onOpenChange={open => !open && setClearOpen(false)}
        onConfirm={() => clearMutation.mutate()}
      />
    </div>
  );
}

function rangeStartBound(start: string, startTime: string) {
  const date = parseRangeDay(start);
  if (!date) return null;
  applyTimeOfDay(date, startTime, 0, 0, 0);
  return Math.floor(date.getTime() / 1000);
}

function rangeEndBound(end: string, endTime: string) {
  const date = parseRangeDay(end);
  if (!date) return null;
  applyTimeOfDay(date, endTime, 23, 59, 59);
  return Math.floor(date.getTime() / 1000);
}

function applyTimeOfDay(date: Date, time: string, hours: number, minutes: number, seconds: number) {
  const match = /^(\d{1,2}):(\d{1,2})(?::(\d{1,2}))?$/.exec(time.trim());
  if (!match) {
    date.setHours(hours, minutes, seconds, 0);
    return;
  }
  date.setHours(Number(match[1]), Number(match[2]), Number(match[3] ?? 0), 0);
}

function parseRangeDay(value: string) {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (!match) return null;
  const date = new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]));
  return Number.isNaN(date.valueOf()) ? null : date;
}

function moduleBadgeClassName(module: string) {
  return (
    {
      auth: 'bg-sky-500/12 text-sky-700 dark:bg-sky-400/15 dark:text-sky-300',
      hosts: 'bg-blue-500/12 text-blue-700 dark:bg-blue-400/15 dark:text-blue-300',
      tasks: 'bg-violet-500/12 text-violet-700 dark:bg-violet-400/15 dark:text-violet-300',
      executions: 'bg-amber-500/12 text-amber-700 dark:bg-amber-400/15 dark:text-amber-300',
      alerts: 'bg-rose-500/12 text-rose-700 dark:bg-rose-400/15 dark:text-rose-300',
      audit: 'bg-fuchsia-500/12 text-fuchsia-700 dark:bg-fuchsia-400/15 dark:text-fuchsia-300',
      notify: 'bg-teal-500/12 text-teal-700 dark:bg-teal-400/15 dark:text-teal-300',
      agents: 'bg-indigo-500/12 text-indigo-700 dark:bg-indigo-400/15 dark:text-indigo-300',
      settings: 'bg-lime-500/12 text-lime-700 dark:bg-lime-400/15 dark:text-lime-300',
    }[module] ?? 'bg-slate-500/12 text-slate-700 dark:bg-slate-400/15 dark:text-slate-300'
  );
}

function AuditResultBadge({ result }: { result: AuditEntry['result'] }) {
  const success = result !== 'failure';
  const Icon = success ? CheckCircle2 : XCircle;
  return (
    <span
      className={`inline-flex items-center justify-self-center gap-1 rounded-full border px-2 py-0.5 text-xs ${success ? 'border-emerald-400/30 bg-emerald-400/10 text-emerald-600 dark:text-emerald-300' : 'border-red-400/30 bg-red-400/10 text-red-600 dark:text-red-300'}`}
    >
      <Icon size={12} />
      {success ? '成功' : '失败'}
    </span>
  );
}
