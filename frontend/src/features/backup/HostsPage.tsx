import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Download, Search } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { DataTableShell } from '@/components/data/data-table-shell';
import { Pagination } from '@/components/data/pagination';
import { Input } from '@/components/ui/input';
import { userHasPermission } from '@/lib/permissions';
import { getStoredUser } from '@/lib/session';
import { useHostRefreshPlan, type HostRefreshGroup } from '@/lib/refresh-plan';
import { showErrorToast } from '@/lib/toast-errors';
import { fetchHosts, updateHostRole } from './api';
import { ExportDialog } from './ExportDialog';
import { hostExportColumns } from './export-columns';
import {
  BackupSelect,
  EmptyState,
  HostNameLink,
  HostStatusTag,
  PageHeader,
  RelativeTime,
  platformLabel,
  roleLabel,
} from './components';
import type { Host } from './types';

const roles = [
  ['database', '数据库'],
  ['web', 'Web'],
  ['application', '应用'],
  ['file', '文件'],
  ['virtualization', '虚拟化'],
] as const;

function mergeHosts(old: Host[] | undefined, fresh: Host[]): Host[] | undefined {
  if (!old) return old;
  const freshByID = new Map(fresh.map(host => [host.id, host]));
  const merged = old.map(host => freshByID.get(host.id) ?? host);
  const known = new Set(old.map(host => host.id));
  fresh.forEach(host => {
    if (!known.has(host.id)) merged.push(host);
  });
  return merged;
}

export function HostsPage() {
  const queryClient = useQueryClient();
  const user = getStoredUser();
  const canManage = userHasPermission(user, 'backup.hosts.manage');
  const [keyword, setKeyword] = useState('');
  const [role, setRole] = useState('');
  const [platform, setPlatform] = useState('');
  const [status, setStatus] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(18);
  const [exportOpen, setExportOpen] = useState(false);
  const { data = [] } = useQuery({ queryKey: ['backup', 'hosts'], queryFn: () => fetchHosts() });
  const { data: refreshPlan } = useHostRefreshPlan();
  const refreshGroups = useMemo<HostRefreshGroup[]>(() => {
    if (refreshPlan && refreshPlan.groups.length > 0) return refreshPlan.groups;
    if (data.length === 0) return [];
    const intervalSec = refreshPlan?.defaultIntervalSec ?? 60;
    return [{ intervalSec, hostIds: data.map(host => host.id) }];
  }, [refreshPlan, data]);
  const refreshGroupsRef = useRef(refreshGroups);
  const refreshGroupsKey = refreshGroups.map(group => `${group.intervalSec}:${group.hostIds.join(',')}`).join('|');

  useEffect(() => {
    refreshGroupsRef.current = refreshGroups;
  }, [refreshGroups]);

  useEffect(() => {
    const refresh = (group: HostRefreshGroup) => {
      void fetchHosts(group.hostIds)
        .then(fresh => queryClient.setQueryData<Host[]>(['backup', 'hosts'], old => mergeHosts(old, fresh)))
        .catch(() => undefined);
    };
    const timers = refreshGroupsRef.current.map(group =>
      window.setInterval(() => {
        if (document.visibilityState !== 'visible') return;
        refresh(group);
      }, Math.max(group.intervalSec, 10) * 1_000),
    );
    const onVisible = () => {
      if (document.visibilityState !== 'visible') return;
      refreshGroupsRef.current.forEach(refresh);
    };
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      timers.forEach(timer => window.clearInterval(timer));
      document.removeEventListener('visibilitychange', onVisible);
    };
  }, [refreshGroupsKey, queryClient]);
  const changeRole = useMutation({
    mutationFn: ({ id, role }: { id: string; role: string | null; hostname: string }) =>
      updateHostRole(id, role),
    onSuccess: (_, variables) => {
      toast.success(`主机 ${variables.hostname} 角色已更新`);
      void queryClient.invalidateQueries({ queryKey: ['backup'] });
    },
    onError: error => showErrorToast(error, '更新主机角色失败'),
  });

  const filtered = useMemo(
    () =>
      data.filter(host => {
        const hit =
          !keyword ||
          host.hostname.toLowerCase().includes(keyword.toLowerCase()) ||
          host.ip.includes(keyword) ||
          (host.os ?? '').toLowerCase().includes(keyword.toLowerCase());
        return (
          hit &&
          (role === '' || host.role === role) &&
          (platform === '' || (host.platform ?? '').toLowerCase() === platform) &&
          (status === '' || host.status === status)
        );
      }),
    [data, keyword, role, platform, status]
  );
  const effectivePageSize = pageSize === -1 ? Math.max(filtered.length, 1) : pageSize;
  const pageCount = Math.max(1, Math.ceil(filtered.length / effectivePageSize));
  const currentPage = Math.min(page, pageCount);
  const pageStart = (currentPage - 1) * effectivePageSize;
  const pageItems = filtered.slice(pageStart, pageStart + effectivePageSize);

  useEffect(() => setPage(1), [keyword, pageSize, role, platform, status]);
  useEffect(() => {
    if (page > pageCount) setPage(pageCount);
  }, [page, pageCount]);

  function openExport() {
    if (filtered.length === 0) {
      toast.error('没有可导出的主机');
      return;
    }
    setExportOpen(true);
  }

  return (
    <div className="bm-page-fill">
      <PageHeader title="备份主机" description="Agent 上报的主机清单、在线状态与任务数量" />
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
                placeholder="搜索主机、IP 地址、操作系统"
                className="pl-9"
              />
            </div>
            <BackupSelect
              value={platform}
              onValueChange={setPlatform}
              className="h-10 w-32"
              ariaLabel="主机平台"
              placeholder="平台"
              clearable
              options={[
                { value: 'linux', label: 'Linux' },
                { value: 'windows', label: 'Windows' },
              ]}
            />
            <BackupSelect
              value={role}
              onValueChange={setRole}
              className="h-10 w-32"
              ariaLabel="主机角色"
              placeholder="角色"
              clearable
              options={roles.map(([value, label]) => ({ value, label }))}
            />
            <BackupSelect
              value={status}
              onValueChange={setStatus}
              className="h-10 w-32"
              ariaLabel="主机状态"
              placeholder="状态"
              clearable
              options={[
                { value: 'normal', label: '正常' },
                { value: 'abnormal', label: '异常' },
                { value: 'offline', label: '离线' },
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
              共 {filtered.length} 台
            </span>
          </>
        }
      >
        <div className="flex min-h-full flex-col">
          <table className="w-full min-w-[1060px] text-center text-sm">
            <thead className="sticky top-0 z-10 bg-[var(--bm-table-head-bg)] text-center text-xs font-semibold text-[var(--bm-text-muted)]">
              <tr>
                <th className="px-4 py-3 font-semibold">序号</th>
                <th className="px-4 py-3 font-semibold">主机</th>
                <th className="px-4 py-3 font-semibold">IP 地址</th>
                <th className="px-4 py-3 font-semibold">平台</th>
                <th className="px-4 py-3 font-semibold">操作系统</th>
                <th className="px-4 py-3 font-semibold">角色</th>
                <th className="px-4 py-3 font-semibold">Agent 版本</th>
                <th className="px-4 py-3 font-semibold">任务数</th>
                <th className="px-4 py-3 font-semibold">最后上报</th>
                <th className="px-4 py-3 font-semibold">状态</th>
              </tr>
            </thead>
            <tbody>
              {pageItems.map((host, index) => (
                <tr
                  key={host.id}
                  className="border-t border-[var(--bm-border)]/60 bm-table-row-hover"
                >
                  <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                    {pageStart + index + 1}
                  </td>
                  <td className="px-4 py-3">
                    <HostNameLink hostID={host.id} hostname={host.hostname} />
                  </td>
                  <td className="px-4 py-3 font-mono text-sm text-[var(--bm-text)]">{host.ip}</td>
                  <td className="px-4 py-3 text-[var(--bm-text)]">
                    {platformLabel(host.platform)}
                  </td>
                  <td className="px-4 py-3 text-[var(--bm-text)]">{host.os || '-'}</td>
                  <td className="px-4 py-3">
                    {canManage ? (
                      <BackupSelect
                        value={host.role ?? ''}
                        onValueChange={value =>
                          changeRole.mutate({
                            id: host.id,
                            hostname: host.hostname,
                            role: value === 'clear' ? null : value,
                          })
                        }
                        className="bm-role-select mx-auto"
                        placeholder="-"
                        ariaLabel={`${host.hostname} 角色`}
                        disabled={changeRole.isPending}
                        options={roles.map(([value, label]) => ({ value, label }))}
                        footerOption={host.role ? { value: 'clear', label: '清除' } : undefined}
                        contentClassName="bm-role-menu"
                      />
                    ) : (
                      <span className="text-[var(--bm-text)]">{roleLabel(host.role)}</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-[var(--bm-text)]">{host.agentVersion || '-'}</td>
                  <td className="px-4 py-3 text-[var(--bm-text)]">{host.taskCount}</td>
                  <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                    <RelativeTime value={host.lastReportTime} />
                  </td>
                  <td className="px-4 py-3">
                    <HostStatusTag status={host.status} />
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
        title="备份主机"
        description={`将当前筛选后的 ${filtered.length} 条主机进行导出`}
        filenamePrefix="备份主机"
        items={filtered}
        columns={hostExportColumns}
        successText={`已导出 ${filtered.length} 条主机`}
        auditModule="hosts"
      />
    </div>
  );
}
