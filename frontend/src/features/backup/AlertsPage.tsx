import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Download, RotateCcw, Search, Trash2 } from 'lucide-react';
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
import { errorMessageOf } from '@/lib/toast-errors';
import { deleteAlert, fetchAlerts, resolveAlert } from './api';
import { ExportDialog } from './ExportDialog';
import { alertExportColumns } from './export-columns';
import {
  AlertStatusTag,
  BackupSelect,
  EmptyState,
  PageHeader,
  Tag,
  alertTypeLabel,
  formatTime,
} from './components';
import type { Alert } from './types';

export function AlertsPage() {
  const queryClient = useQueryClient();
  const user = getStoredUser();
  const canManage = userHasPermission(user, 'backup.alerts.manage');
  const [keyword, setKeyword] = useState('');
  const [status, setStatus] = useState('');
  const [type, setType] = useState('');
  const [level, setLevel] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(18);
  const [exportOpen, setExportOpen] = useState(false);
  const [deleting, setDeleting] = useState<Alert | null>(null);
  const { data = [], isLoading } = useQuery({
    queryKey: ['backup', 'alerts'],
    queryFn: fetchAlerts,
  });

  const resolve = useMutation({
    mutationFn: (id: string) => resolveAlert(id),
    onSuccess: () => {
      toast.success('告警记录已恢复');
      void queryClient.invalidateQueries({ queryKey: ['backup'] });
    },
    onError: error => toast.error(`告警记录恢复失败：${errorMessageOf(error, '未知错误')}`),
  });

  const remove = useMutation({
    mutationFn: (id: string) => deleteAlert(id),
    onSuccess: () => {
      toast.success('告警记录已删除');
      setDeleting(null);
      void queryClient.invalidateQueries({ queryKey: ['backup'] });
    },
    onError: error => toast.error(`告警记录删除失败：${errorMessageOf(error, '未知错误')}`),
  });

  const filtered = useMemo(
    () =>
      data.filter(alert => {
        const hit =
          !keyword ||
          alert.hostName.toLowerCase().includes(keyword.toLowerCase()) ||
          (alert.taskName ?? '').toLowerCase().includes(keyword.toLowerCase()) ||
          alert.message.toLowerCase().includes(keyword.toLowerCase());
        return (
          hit &&
          (status === '' || alert.status === status) &&
          (type === '' || alert.type === type) &&
          (level === '' || alert.level === level)
        );
      }),
    [data, keyword, status, type, level]
  );
  const effectivePageSize = pageSize === -1 ? Math.max(filtered.length, 1) : pageSize;
  const pageCount = Math.max(1, Math.ceil(filtered.length / effectivePageSize));
  const currentPage = Math.min(page, pageCount);
  const pageStart = (currentPage - 1) * effectivePageSize;
  const pageItems = filtered.slice(pageStart, pageStart + effectivePageSize);

  useEffect(() => setPage(1), [keyword, pageSize, status, type, level]);
  useEffect(() => {
    if (page > pageCount) setPage(pageCount);
  }, [page, pageCount]);

  function openExport() {
    if (filtered.length === 0) {
      toast.error('没有可导出的告警');
      return;
    }
    setExportOpen(true);
  }

  return (
    <div className="bm-page-fill">
      <PageHeader title="告警中心" description="备份失败、未按时执行与主机异常/离线事件" />
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
                placeholder="搜索主机、任务、告警内容"
                className="pl-9"
              />
            </div>
            <BackupSelect
              value={status}
              onValueChange={setStatus}
              className="h-10 w-32"
              ariaLabel="告警状态"
              placeholder="状态"
              clearable
              options={[
                { value: 'active', label: '未处理' },
                { value: 'resolved', label: '已恢复' },
              ]}
            />
            <BackupSelect
              value={type}
              onValueChange={setType}
              className="h-10 w-36"
              ariaLabel="告警类型"
              placeholder="告警类型"
              clearable
              options={[
                { value: 'backup_failed', label: '备份失败' },
                { value: 'not_run', label: '未按时执行' },
                { value: 'report_timeout', label: '主机异常/离线' },
              ]}
            />
            <BackupSelect
              value={level}
              onValueChange={setLevel}
              className="h-10 w-32"
              ariaLabel="告警级别"
              placeholder="级别"
              clearable
              options={[
                { value: 'warning', label: '警告' },
                { value: 'critical', label: '严重' },
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
          <table className="w-full min-w-[1000px] text-center text-sm">
            <thead className="sticky top-0 z-10 bg-[var(--bm-table-head-bg)] text-center text-xs font-semibold text-[var(--bm-text-muted)]">
              <tr>
                <th className="px-4 py-3 font-semibold">序号</th>
                <th className="px-4 py-3 font-semibold">告警时间</th>
                <th className="px-4 py-3 font-semibold">级别</th>
                <th className="px-4 py-3 font-semibold">主机</th>
                <th className="px-4 py-3 font-semibold">告警类型</th>
                <th className="px-4 py-3 font-semibold">告警内容</th>
                <th className="px-4 py-3 font-semibold">状态</th>
                <th className="px-4 py-3 font-semibold">恢复时间</th>
                {canManage ? <th className="px-4 py-3 font-semibold">操作</th> : null}
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                <tr>
                  <td
                    colSpan={canManage ? 9 : 8}
                    className="px-4 py-12 text-xs text-[var(--bm-text-muted)]"
                  >
                    正在读取告警
                  </td>
                </tr>
              ) : null}
              {pageItems.map((alert, index) => (
                <tr
                  key={alert.id}
                  className="border-t border-[var(--bm-border)]/60 bm-table-row-hover"
                >
                  <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                    {pageStart + index + 1}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-sm text-[var(--bm-text)]">
                    {formatTime(alert.createdAt)}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-xs">
                    <span
                      className={`font-semibold ${alert.level === 'critical' ? 'text-red-300' : 'text-amber-300'}`}
                    >
                      {alert.level === 'critical' ? '严重' : '警告'}
                    </span>
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-[var(--bm-text)]">
                    {alert.hostName}
                  </td>
                  <td className="px-4 py-3">
                    <Tag
                      tone={
                        alert.type === 'backup_failed'
                          ? 'danger'
                          : alert.type === 'not_run'
                            ? 'warning'
                            : 'info'
                      }
                    >
                      {alertTypeLabel(alert.type)}
                    </Tag>
                  </td>
                  <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                    <div className="mx-auto line-clamp-2 max-w-md leading-5">{alert.message}</div>
                  </td>
                  <td className="px-4 py-3">
                    <AlertStatusTag status={alert.status} />
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-sm text-[var(--bm-text)]">
                    {alert.resolvedAt ? formatTime(alert.resolvedAt) : '-'}
                  </td>
                  {canManage ? (
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-center gap-2">
                        <AppTooltip label="恢复">
                          <Button
                            variant="ghost"
                            size="sm"
                            aria-label="恢复"
                            className="bm-row-action-success h-8 w-8 rounded-lg px-0"
                            disabled={alert.status === 'resolved' || resolve.isPending}
                            onClick={() => resolve.mutate(alert.id)}
                          >
                            <RotateCcw size={14} />
                          </Button>
                        </AppTooltip>
                        <AppTooltip label="删除">
                          <Button
                            variant="ghost"
                            size="sm"
                            aria-label="删除"
                            className="bm-row-action-delete h-8 w-8 rounded-lg px-0"
                            onClick={() => setDeleting(alert)}
                          >
                            <Trash2 size={14} />
                          </Button>
                        </AppTooltip>
                      </div>
                    </td>
                  ) : null}
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
        title="告警记录"
        description={`将当前筛选后的 ${filtered.length} 条告警进行导出`}
        filenamePrefix="告警记录"
        items={filtered}
        columns={alertExportColumns}
        successText={`已导出 ${filtered.length} 条告警`}
        auditModule="alerts"
      />
      <ConfirmDialog
        open={Boolean(deleting)}
        title="删除告警"
        description="删除后不可恢复，将无法再看到该告警记录详情"
        detail={deleting ? `确认删除主机「${deleting.hostName}」的这条告警记录吗？` : undefined}
        icon="delete"
        horizontalHeader
        busy={remove.isPending}
        onOpenChange={open => !open && setDeleting(null)}
        onConfirm={() => deleting && remove.mutate(deleting.id)}
      />
    </div>
  );
}
