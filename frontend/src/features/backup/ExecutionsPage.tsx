import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Download, Eye, Search, Trash2 } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/feedback/app-tooltip';
import { Button } from '@/components/ui/button';
import { DataTableShell } from '@/components/data/data-table-shell';
import { Pagination } from '@/components/data/pagination';
import { ConfirmDialog } from '@/components/dialogs/confirm-dialog';
import { Input } from '@/components/ui/input';
import { userHasPermission } from '@/lib/permissions';
import { getStoredUser } from '@/lib/session';
import { showErrorToast } from '@/lib/toast-errors';
import { deleteExecution, fetchExecutions } from './api';
import { ExecutionDetailDialog } from './ExecutionDetailDialog';
import { ExportDialog } from './ExportDialog';
import { executionExportColumns } from './export-columns';
import { executionDuration, sizeLabel } from './execution-utils';
import {
  BackupSelect,
  EmptyState,
  ExecutionStateTag,
  HostNameLink,
  MethodTag,
  PageHeader,
  TaskNameLink,
  formatTime,
} from './components';
import type { BackupExecution } from './types';

export function ExecutionsPage() {
  const queryClient = useQueryClient();
  const user = getStoredUser();
  const canManage = userHasPermission(user, 'backup.executions.manage');
  const [keyword, setKeyword] = useState('');
  const [method, setMethod] = useState('');
  const [state, setState] = useState('');
  const [detail, setDetail] = useState<BackupExecution | null>(null);
  const [deleting, setDeleting] = useState<BackupExecution | null>(null);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(18);
  const [exportOpen, setExportOpen] = useState(false);
  const { data = [] } = useQuery({
    queryKey: ['backup', 'executions'],
    queryFn: () => fetchExecutions(),
  });

  const filtered = useMemo(
    () =>
      data.filter(item => {
        const hit =
          !keyword ||
          (item.taskName ?? '').toLowerCase().includes(keyword.toLowerCase()) ||
          (item.hostName ?? '').toLowerCase().includes(keyword.toLowerCase()) ||
          (item.artifactPath ?? '').toLowerCase().includes(keyword.toLowerCase());
        return (
          hit &&
          (method === '' || item.taskType === method) &&
          (state === '' || item.state === state)
        );
      }),
    [data, keyword, method, state]
  );
  const effectivePageSize = pageSize === -1 ? Math.max(filtered.length, 1) : pageSize;
  const pageCount = Math.max(1, Math.ceil(filtered.length / effectivePageSize));
  const currentPage = Math.min(page, pageCount);
  const pageStart = (currentPage - 1) * effectivePageSize;
  const pageItems = filtered.slice(pageStart, pageStart + effectivePageSize);

  useEffect(() => setPage(1), [keyword, method, pageSize, state]);
  useEffect(() => {
    if (page > pageCount) setPage(pageCount);
  }, [page, pageCount]);

  function openExport() {
    if (filtered.length === 0) {
      toast.error('没有可导出的执行记录');
      return;
    }
    setExportOpen(true);
  }

  const remove = useMutation({
    mutationFn: (id: string) => deleteExecution(id),
    onSuccess: () => {
      toast.success('执行记录已删除');
      setDeleting(null);
      void queryClient.invalidateQueries({ queryKey: ['backup'] });
    },
    onError: error => showErrorToast(error, '删除执行记录失败'),
  });

  return (
    <div className="bm-page-fill">
      <PageHeader title="执行记录" description="每次备份执行的状态、体积、产物与耗时" />
      <DataTableShell
        className="bm-page-card"
        contentClassName="bm-scroll-area flex-1"
        toolbarClassName="bm-hidden-scrollbar flex-nowrap overflow-x-auto"
        footer={
          <Pagination
            page={currentPage}
            pageCount={pageCount}
            pageSize={pageSize}
            total={filtered.length}
            onPageChange={setPage}
            onPageSizeChange={setPageSize}
          />
        }
        toolbar={
          <>
            <div className="relative min-w-56 flex-1 sm:max-w-80">
              <Search
                size={15}
                className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--bm-text-muted)]"
              />
              <Input
                value={keyword}
                onChange={event => setKeyword(event.target.value)}
                placeholder="搜索任务、主机、产物路径"
                className="pl-9"
              />
            </div>
            <BackupSelect
              value={method}
              onValueChange={setMethod}
              className="h-10 w-36"
              ariaLabel="备份方式"
              placeholder="备份方式"
              clearable
              options={[
                { value: 'veeam', label: 'Veeam' },
                { value: 'shell', label: 'Shell' },
                { value: 'powershell', label: 'PowerShell' },
                { value: 'batch', label: '批处理' },
                { value: 'exe', label: '程序' },
              ]}
            />
            <BackupSelect
              value={state}
              onValueChange={setState}
              className="h-10 w-32"
              ariaLabel="执行状态"
              placeholder="状态"
              clearable
              options={[
                { value: 'success', label: '成功' },
                { value: 'failed', label: '失败' },
                { value: 'running', label: '运行中' },
              ]}
            />
            {canManage ? (
              <Button
                variant="outline"
                className="bm-audit-export-button shrink-0"
                onClick={openExport}
              >
                <Download size={16} />
                导出
              </Button>
            ) : null}
            <span className="ml-auto text-xs text-[var(--bm-text-muted)]">
              共 {filtered.length} 条
            </span>
          </>
        }
      >
        <div className="flex min-h-full flex-col">
          <table className="w-full min-w-[1100px] text-center text-sm">
            <thead className="sticky top-0 z-10 bg-[var(--bm-table-head-bg)] text-center text-xs font-semibold text-[var(--bm-text-muted)]">
              <tr>
                <th className="px-4 py-3 font-semibold">序号</th>
                <th className="px-4 py-3 font-semibold">任务</th>
                <th className="px-4 py-3 font-semibold">主机</th>
                <th className="px-4 py-3 font-semibold">备份方式</th>
                <th className="px-4 py-3 font-semibold">状态</th>
                <th className="px-4 py-3 font-semibold">触发</th>
                <th className="px-4 py-3 font-semibold">执行时间</th>
                <th className="px-4 py-3 font-semibold">结束时间</th>
                <th className="px-4 py-3 font-semibold">耗时</th>
                <th className="px-4 py-3 font-semibold">备份大小</th>
                <th className="px-4 py-3 font-semibold">操作</th>
              </tr>
            </thead>
            <tbody>
              {pageItems.map((item, index) => (
                <tr
                  key={item.id}
                  className="border-t border-[var(--bm-border)]/60 bm-table-row-hover"
                >
                  <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                    {pageStart + index + 1}
                  </td>
                  <td className="px-4 py-3">
                    <TaskNameLink taskID={item.taskId} name={item.taskName || item.taskId} />
                  </td>
                  <td className="px-4 py-3">
                    {item.hostId ? (
                      <HostNameLink hostID={item.hostId} hostname={item.hostName || '-'} />
                    ) : (
                      <span className="text-sm text-[var(--bm-text)]">{item.hostName || '-'}</span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    {item.taskType ? <MethodTag method={item.taskType} /> : '-'}
                  </td>
                  <td className="px-4 py-3">
                    <ExecutionStateTag state={item.state} />
                  </td>
                  <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                    {item.trigger === 'manual'
                      ? '手动'
                      : item.trigger === 'scheduled'
                        ? '定时'
                        : '-'}
                  </td>
                  <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                    {formatTime(item.startTime || item.creationTime)}
                  </td>
                  <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                    {formatTime(item.endTime)}
                  </td>
                  <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                    {executionDuration(item)}
                  </td>
                  <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                    {sizeLabel(item.transferredSize || item.processedSize)}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center justify-center gap-2">
                      <AppTooltip label="查看详情">
                        <Button
                          variant="ghost"
                          size="sm"
                          aria-label="查看详情"
                          className="bm-row-action-view h-8 w-8 rounded-lg px-0"
                          onClick={() => setDetail(item)}
                        >
                          <Eye size={14} />
                        </Button>
                      </AppTooltip>
                      {canManage ? (
                        <AppTooltip label="删除记录">
                          <Button
                            variant="ghost"
                            size="sm"
                            aria-label="删除记录"
                            className="bm-row-action-delete h-8 w-8 rounded-lg px-0"
                            onClick={() => setDeleting(item)}
                          >
                            <Trash2 size={14} />
                          </Button>
                        </AppTooltip>
                      ) : null}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {filtered.length === 0 ? <EmptyState className="flex-1">暂无数据</EmptyState> : null}
        </div>
      </DataTableShell>

      <ExportDialog
        open={exportOpen}
        onOpenChange={setExportOpen}
        title="执行记录"
        description={`将当前筛选后的 ${filtered.length} 条执行记录进行导出`}
        filenamePrefix="执行记录"
        items={filtered}
        columns={executionExportColumns}
        successText={`已导出 ${filtered.length} 条执行记录`}
        auditModule="executions"
      />
      <ExecutionDetailDialog execution={detail} onOpenChange={open => !open && setDetail(null)} />

      <ConfirmDialog
        open={Boolean(deleting)}
        title="删除执行记录"
        description="删除后不可恢复，将无法再看到该执行记录详情"
        detail={
          deleting ? `确认删除任务「${deleting.taskName ?? '-'}」的这条执行记录吗？` : undefined
        }
        horizontalHeader
        busy={remove.isPending}
        onOpenChange={open => !open && setDeleting(null)}
        onConfirm={() => deleting && remove.mutate(deleting.id)}
      />
    </div>
  );
}
