import { ArrowLeft, Cpu, Server, Trash2 } from 'lucide-react';
import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import { Link, useNavigate } from '@tanstack/react-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { ConfirmDialog } from '@/components/dialogs/confirm-dialog';
import { userHasPermission } from '@/lib/permissions';
import { getStoredUser } from '@/lib/session';
import { showErrorToast } from '@/lib/toast-errors';
import { deleteHost, fetchHost } from './api';
import {
  EmptyState,
  HostStatusTag,
  MethodTag,
  TaskNameLink,
  TaskStatusTag,
  formatTime,
  panelStyle,
  roleLabel,
  taskSchedule,
} from './components';
import type { ReactNode } from 'react';

export function HostDetailPage({ hostID }: { hostID: string }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const canManage = userHasPermission(getStoredUser(), 'backup.hosts.manage');
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const { data, isFetching } = useQuery({
    queryKey: ['backup', 'host', hostID],
    queryFn: () => fetchHost(hostID),
  });
  const tasks = data?.tasks ?? [];
  const visibleTasks = expanded ? tasks : tasks.slice(0, 6);
  const taskCardRef = useRef<HTMLElement>(null);
  const [taskCardHeight, setTaskCardHeight] = useState<number>();

  useEffect(() => {
    setExpanded(false);
  }, [hostID]);

  useLayoutEffect(() => {
    const card = taskCardRef.current;
    const main = card?.closest('main');
    if (!card || !main) return;
    const measure = () => {
      const header = card.querySelector<HTMLElement>('[data-card-heading]');
      const table = card.querySelector('table');
      if (!header) return;
      const page = main.firstElementChild;
      const paddingBottom = page ? parseFloat(getComputedStyle(page).paddingBottom) || 0 : 0;
      const mainRect = main.getBoundingClientRect();
      const cardRect = card.getBoundingClientRect();
      const cardOffsetTop = cardRect.top - mainRect.top + main.scrollTop;
      const available = main.clientHeight - paddingBottom - cardOffsetTop;
      const content =
        header.getBoundingClientRect().height + (table?.getBoundingClientRect().height ?? 0);
      setTaskCardHeight(Math.ceil(Math.max(content, available)));
    };
    measure();
    const frame = requestAnimationFrame(measure);
    const timer = window.setTimeout(measure, 360);
    window.addEventListener('resize', measure);
    return () => {
      cancelAnimationFrame(frame);
      clearTimeout(timer);
      window.removeEventListener('resize', measure);
    };
  }, [hostID, data, tasks.length, expanded]);

  const toggleTaskExpanded = () => {
    const next = !expanded;
    setExpanded(next);
    if (!next) {
      requestAnimationFrame(() => {
        taskCardRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' });
      });
    }
  };

  const remove = useMutation({
    mutationFn: () => deleteHost(hostID),
    onSuccess: () => {
      toast.success('主机已删除');
      void queryClient.invalidateQueries({ queryKey: ['backup'] });
      void navigate({ to: '/hosts' });
    },
    onError: error => showErrorToast(error, '删除主机失败'),
  });

  return (
    <div className={`bm-page-fill gap-5 ${expanded ? 'h-auto min-h-full' : ''}`}>
      <div className="flex shrink-0 flex-wrap items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-3">
          <Link to="/hosts" className="bm-back-link group">
            <ArrowLeft
              size={14}
              className="transition-transform duration-200 group-hover:-translate-x-0.5"
            />
            返回主机列表
          </Link>
          <span className="text-sm text-[var(--bm-text-muted)]">/</span>
          <h1 className="flex min-w-0 items-center gap-2 text-lg font-semibold text-[var(--bm-text)]">
            <Server size={19} className="shrink-0 text-[var(--bm-accent-text)]" />
            <span className="truncate">{data?.hostname ?? '主机详情'}</span>
          </h1>
          {data ? <HostStatusTag status={data.status} /> : null}
        </div>
        {data && canManage ? (
          <Button
            variant="outline"
            className="bm-danger-soft-button shrink-0"
            onClick={() => setDeleteOpen(true)}
          >
            <Trash2 size={15} />
            删除主机
          </Button>
        ) : null}
      </div>

      {isFetching && !data ? <EmptyState>正在加载主机详情</EmptyState> : null}

      {!isFetching && !data ? (
        <section className="bm-card-hover rounded-xl p-6" style={panelStyle}>
          <EmptyState>主机不存在或已被删除</EmptyState>
        </section>
      ) : null}

      {data ? (
        <>
          <section className="bm-card-hover shrink-0 rounded-xl p-5" style={panelStyle}>
            <div className="mb-4 flex items-center gap-2 text-sm font-semibold text-[var(--bm-text)]">
              <Server size={16} className="text-[var(--bm-accent-text)]" />
              主机信息
            </div>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
              <Info label="IP 地址" value={data.ip} />
              <Info label="操作系统" value={data.os || '-'} />
              <Info label="角色" value={roleLabel(data.role)} />
              <Info label="状态" value={<HostStatusTag status={data.status} />} />
              <Info label="Agent 版本" value={data.agentVersion || '-'} />
              <Info label="任务数" value={data.taskCount} />
              <Info
                label="机器标识"
                value={<span className="break-all">{data.nodeId || '-'}</span>}
              />
              <Info label="最近上报" value={formatTime(data.lastReportTime)} />
            </div>
          </section>

          <section
            ref={taskCardRef}
            className="bm-card-hover flex min-h-0 shrink-0 flex-col overflow-hidden rounded-xl"
            style={{ ...panelStyle, height: taskCardHeight }}
          >
            <div
              data-card-heading
              className="flex shrink-0 flex-wrap items-center justify-between gap-3 border-b border-[var(--bm-border)] px-4 py-3"
            >
              <div className="flex items-center gap-2 text-sm font-semibold text-[var(--bm-text)]">
                <Cpu size={16} className="text-[var(--bm-accent-text)]" />
                备份任务
              </div>
              <div className="flex items-center gap-3">
                <span className="text-xs text-[var(--bm-text-muted)]">共 {tasks.length} 个</span>
                {tasks.length > 6 ? (
                  <button
                    type="button"
                    className={`bm-detail-toggle ${
                      expanded ? 'bm-detail-toggle-collapse' : 'bm-detail-toggle-expand'
                    }`}
                    onClick={toggleTaskExpanded}
                  >
                    {expanded ? '收起' : `展开全部 ${tasks.length} 个`}
                  </button>
                ) : null}
              </div>
            </div>
            {tasks.length === 0 ? (
              <EmptyState className="flex-1">该主机暂无任务</EmptyState>
            ) : (
              <div
                className={
                  expanded ? 'overflow-x-auto' : 'bm-scroll-area min-h-0 flex-1 overflow-x-auto'
                }
              >
                <table className="w-full min-w-[760px] text-center text-sm">
                  <thead className="sticky top-0 z-10 bg-[var(--bm-table-head-bg)] text-center text-xs font-semibold text-[var(--bm-text-muted)]">
                    <tr>
                      <th className="px-4 py-3 font-semibold">序号</th>
                      <th className="px-4 py-3 font-semibold">任务</th>
                      <th className="px-4 py-3 font-semibold">备份方式</th>
                      <th className="px-4 py-3 font-semibold">执行计划</th>
                      <th className="px-4 py-3 font-semibold">最近执行</th>
                      <th className="px-4 py-3 font-semibold">状态</th>
                    </tr>
                  </thead>
                  <tbody>
                    {visibleTasks.map((task, index) => (
                      <tr
                        key={task.id}
                        className="border-t border-[var(--bm-border)]/60 transition bm-table-row-hover"
                      >
                        <td className="px-4 py-3 text-sm text-[var(--bm-text-muted)]">
                          {index + 1}
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex justify-center">
                            <TaskNameLink taskID={task.id} name={task.name} />
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <MethodTag method={task.type} />
                        </td>
                        <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                          {taskSchedule(task)}
                        </td>
                        <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                          {formatTime(task.lastExecutionTime)}
                        </td>
                        <td className="px-4 py-3">
                          <TaskStatusTag status={task.status} />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        </>
      ) : null}

      <ConfirmDialog
        open={deleteOpen}
        title="删除主机"
        description="删除后不可恢复，关联的数据也会一并移除"
        detail={`确认删除主机「${data?.hostname ?? ''}」吗？其 ${data?.tasks.length ?? 0} 个任务及全部执行记录、告警将一并删除。`}
        confirmText="删除主机"
        icon="delete"
        horizontalHeader
        busy={remove.isPending}
        onOpenChange={setDeleteOpen}
        onConfirm={() => remove.mutate()}
      />
    </div>
  );
}

function Info({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="bm-info-card rounded-lg border border-[var(--bm-border)]/60 bg-[var(--bm-control-bg-soft)] p-4">
      <div className="text-xs text-[var(--bm-text-muted)]">{label}</div>
      <div className="mt-1.5 text-sm font-medium text-[var(--bm-text)]">{value}</div>
    </div>
  );
}
