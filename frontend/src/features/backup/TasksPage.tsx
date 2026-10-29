import { useQuery } from '@tanstack/react-query';
import { useSearch } from '@tanstack/react-router';
import { Download, Search } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { DataTableShell } from '@/components/data/data-table-shell';
import { Pagination } from '@/components/data/pagination';
import { Input } from '@/components/ui/input';
import { userHasPermission } from '@/lib/permissions';
import { getStoredUser } from '@/lib/session';
import { fetchTasks } from './api';
import { ExportDialog } from './ExportDialog';
import type { BackupTask } from './types';
import { taskExportColumns } from './export-columns';
import {
  BackupSelect,
  EmptyState,
  HostNameLink,
  MethodTag,
  PageHeader,
  TaskNameLink,
  TaskStatusTag,
  formatTime,
  taskSchedule,
} from './components';

// compareHostIp 按 IPv4 四段数值比较，缺失段按 -1 处理。
function compareHostIp(left: string, right: string): number {
  const toParts = (ip: string) => ip.split('.').map(part => Number.parseInt(part, 10));
  const leftParts = toParts(left);
  const rightParts = toParts(right);
  for (let index = 0; index < 4; index += 1) {
    const leftValue = Number.isFinite(leftParts[index]) ? leftParts[index] : -1;
    const rightValue = Number.isFinite(rightParts[index]) ? rightParts[index] : -1;
    if (leftValue !== rightValue) return leftValue - rightValue;
  }
  return 0;
}

// compareTaskOrder 先按主机 IP，再按主机名与任务名排序。
function compareTaskOrder(left: BackupTask, right: BackupTask): number {
  const byIp = compareHostIp(left.hostIp ?? '', right.hostIp ?? '');
  if (byIp !== 0) return byIp;
  if (left.hostName !== right.hostName) return left.hostName.localeCompare(right.hostName);
  return left.name.localeCompare(right.name);
}

export function TasksPage() {
  const user = getStoredUser();
  const canManage = userHasPermission(user, 'backup.tasks.manage');
  const [keyword, setKeyword] = useState('');
  const search = useSearch({ strict: false }) as { status?: string };
  const [status, setStatus] = useState(
    search.status && search.status !== 'all' ? search.status : ''
  );
  const [method, setMethod] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(18);
  const [exportOpen, setExportOpen] = useState(false);
  const { data = [] } = useQuery({ queryKey: ['backup', 'tasks'], queryFn: fetchTasks });

  useEffect(() => {
    setStatus(search.status && search.status !== 'all' ? search.status : '');
  }, [search.status]);

  const filtered = useMemo(
    () =>
      [...data]
        .filter(task => {
          const hit =
            !keyword ||
            task.name.toLowerCase().includes(keyword.toLowerCase()) ||
            task.hostName.toLowerCase().includes(keyword.toLowerCase()) ||
            (task.hostIp ?? '').toLowerCase().includes(keyword.toLowerCase());
          return (
            hit &&
            (status === '' || task.status === status) &&
            (method === '' || task.type === method)
          );
        })
        .sort(compareTaskOrder),
    [data, keyword, status, method]
  );
  const effectivePageSize = pageSize === -1 ? Math.max(filtered.length, 1) : pageSize;
  const pageCount = Math.max(1, Math.ceil(filtered.length / effectivePageSize));
  const currentPage = Math.min(page, pageCount);
  const pageStart = (currentPage - 1) * effectivePageSize;
  const pageItems = filtered.slice(pageStart, pageStart + effectivePageSize);

  useEffect(() => setPage(1), [keyword, method, pageSize, status]);
  useEffect(() => {
    if (page > pageCount) setPage(pageCount);
  }, [page, pageCount]);

  function openExport() {
    if (filtered.length === 0) {
      toast.error('没有可导出的任务');
      return;
    }
    setExportOpen(true);
  }

  return (
    <div className="bm-page-fill">
      <PageHeader title="备份任务" description="由 Agent 采集的备份任务、执行计划与健康状态" />
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
            <div className="relative min-w-56 flex-1 sm:max-w-72">
              <Search
                size={15}
                className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--bm-text-muted)]"
              />
              <Input
                value={keyword}
                onChange={event => setKeyword(event.target.value)}
                placeholder="搜索任务、主机、IP"
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
              value={status}
              onValueChange={setStatus}
              className="h-10 w-32"
              ariaLabel="任务状态"
              placeholder="状态"
              clearable
              options={[
                { value: 'normal', label: '正常' },
                { value: 'failed', label: '失败' },
                { value: 'not_run', label: '未执行' },
                { value: 'running', label: '运行中' },
                { value: 'no_data', label: '无数据' },
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
              共 {filtered.length} 个
            </span>
          </>
        }
      >
        <div className="flex min-h-full flex-col">
          <table className="w-full min-w-[1050px] text-center text-sm">
            <thead className="sticky top-0 z-10 bg-[var(--bm-table-head-bg)] text-center text-xs font-semibold text-[var(--bm-text-muted)]">
              <tr>
                <th className="px-4 py-3 font-semibold">序号</th>
                <th className="px-4 py-3 font-semibold">任务</th>
                <th className="px-4 py-3 font-semibold">主机</th>
                <th className="px-4 py-3 font-semibold">IP</th>
                <th className="px-4 py-3 font-semibold">备份方式</th>
                <th className="px-4 py-3 font-semibold">执行计划</th>
                <th className="px-4 py-3 font-semibold">最近成功</th>
                <th className="px-4 py-3 font-semibold">状态</th>
              </tr>
            </thead>
            <tbody>
              {pageItems.map((task, index) => (
                <tr
                  key={task.id}
                  className="border-t border-[var(--bm-border)]/60 bm-table-row-hover"
                >
                  <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                    {pageStart + index + 1}
                  </td>
                  <td className="px-4 py-3">
                    <TaskNameLink taskID={task.id} name={task.name} />
                  </td>
                  <td className="px-4 py-3">
                    <HostNameLink hostID={task.hostId} hostname={task.hostName} />
                  </td>
                  <td className="px-4 py-3 font-mono text-sm text-[var(--bm-text)]">
                    {task.hostIp || '-'}
                  </td>
                  <td className="px-4 py-3">
                    <MethodTag method={task.type} />
                  </td>
                  <td className="px-4 py-3 text-sm text-[var(--bm-text)]">{taskSchedule(task)}</td>
                  <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                    {task.lastSuccessTime ? formatTime(task.lastSuccessTime) : '-'}
                  </td>
                  <td className="px-4 py-3">
                    <TaskStatusTag status={task.status} />
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
        title="备份任务"
        description={`将当前筛选后的 ${filtered.length} 条任务进行导出`}
        filenamePrefix="备份任务"
        items={filtered}
        columns={taskExportColumns}
        successText={`已导出 ${filtered.length} 条任务`}
        auditModule="tasks"
      />
    </div>
  );
}
