import { ArrowLeft, CalendarClock, Clock, Pencil, Trash2 } from 'lucide-react';
import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import { Link, useNavigate } from '@tanstack/react-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { ConfirmDialog } from '@/components/dialogs/confirm-dialog';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { userHasPermission } from '@/lib/permissions';
import { getStoredUser } from '@/lib/session';
import { showErrorToast } from '@/lib/toast-errors';
import { deleteTask, fetchTask, fetchTaskExecutions, updateTaskSchedule } from './api';
import { ExecutionDetailDialog } from './ExecutionDetailDialog';
import { executionDuration, sizeLabel } from './execution-utils';
import {
  EmptyState,
  ExecutionStateTag,
  HostNameLink,
  MethodTag,
  Tag,
  TaskStatusTag,
  cronScheduleLabel,
  formatTime,
  panelStyle,
  taskSchedule,
} from './components';
import type { ReactNode } from 'react';
import type { BackupExecution, BackupTask } from './types';

export function TaskDetailPage({ taskID }: { taskID: string }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const canManage = userHasPermission(getStoredUser(), 'backup.tasks.manage');
  const [cron, setCron] = useState('');
  const [editing, setEditing] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const [detail, setDetail] = useState<BackupExecution | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const { data: task } = useQuery({
    queryKey: ['backup', 'task', taskID],
    queryFn: () => fetchTask(taskID),
  });
  const { data: executions = [] } = useQuery({
    queryKey: ['backup', 'task', taskID, 'executions'],
    queryFn: () => fetchTaskExecutions(taskID),
  });
  const visibleExecutions = expanded ? executions : executions.slice(0, 6);
  const configDetails = task ? configRows(task) : [];
  const cronPreview = cronScheduleLabel(cron.trim());
  const retention = task
    ? task.retention || (typeof task.config.retention === 'string' ? task.config.retention : '-')
    : '-';
  const executionCardRef = useRef<HTMLElement>(null);
  const [executionCardHeight, setExecutionCardHeight] = useState<number>();

  useLayoutEffect(() => {
    const card = executionCardRef.current;
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
      setExecutionCardHeight(Math.ceil(Math.max(content, available)));
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
  }, [taskID, task?.id, executions, expanded]);

  const toggleExecutionExpanded = () => {
    const next = !expanded;
    setExpanded(next);
    if (!next) {
      requestAnimationFrame(() => {
        executionCardRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' });
      });
    }
  };

  useEffect(() => {
    if (!task || editing) return;
    setCron(task.scheduleCron ?? '');
  }, [task, editing]);

  useEffect(() => {
    setExpanded(false);
    setDetail(null);
  }, [taskID]);

  const saveSchedule = useMutation({
    mutationFn: () => updateTaskSchedule(taskID, cron),
    onSuccess: result => {
      if (result.delivery?.delivered === false) {
        toast.error(result.delivery.message || '执行计划下发失败');
        return;
      }
      toast.success(`${task?.name ?? '任务'} 执行计划已更新`);
      setEditing(false);
      void queryClient.invalidateQueries({ queryKey: ['backup'] });
    },
    onError: error => showErrorToast(error, '修改执行计划失败'),
  });

  const remove = useMutation({
    mutationFn: () => deleteTask(taskID),
    onSuccess: () => {
      toast.success('任务已删除');
      void queryClient.invalidateQueries({ queryKey: ['backup'] });
      void navigate({ to: '/tasks', search: {} });
    },
    onError: error => showErrorToast(error, '删除任务失败'),
  });

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-3">
          <Link to="/tasks" search={{}} className="bm-back-link group">
            <ArrowLeft
              size={14}
              className="transition-transform duration-200 group-hover:-translate-x-0.5"
            />
            返回任务列表
          </Link>
          <span className="text-sm text-[var(--bm-text-muted)]">/</span>
          <h1 className="flex min-w-0 items-center gap-2 text-lg font-semibold text-[var(--bm-text)]">
            <CalendarClock size={19} className="shrink-0 text-[var(--bm-accent-text)]" />
            <span className="truncate">{task?.name ?? '任务详情'}</span>
          </h1>
          {task ? <TaskStatusTag status={task.status} /> : null}
        </div>
        {task && canManage ? (
          <Button
            variant="outline"
            className="bm-danger-soft-button shrink-0"
            onClick={() => setDeleteOpen(true)}
          >
            <Trash2 size={15} />
            删除任务
          </Button>
        ) : null}
      </div>

      {!task ? <EmptyState>正在加载任务详情</EmptyState> : null}

      {task ? (
        <>
          <section className="bm-card-hover rounded-xl p-5" style={panelStyle}>
            <div className="mb-4 flex items-center gap-2 text-sm font-semibold text-[var(--bm-text)]">
              <CalendarClock size={16} className="text-[var(--bm-accent-text)]" />
              任务信息
            </div>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
              <Info
                label="主机"
                value={<HostNameLink hostID={task.hostId} hostname={task.hostName} />}
              />
              <Info label="备份方式" value={<MethodTag method={task.type} />} />
              <Info label="最近成功" value={formatTime(task.lastSuccessTime)} />
              <Info label="成功率" value={`${task.successRate}%`} />
              <Info
                label="连续失败"
                value={task.consecutiveFailures ? `${task.consecutiveFailures} 次` : '无'}
              />
              <Info label="最近备份大小" value={sizeLabel(task.latestSize)} />
              {task.type === 'shell' ? (
                <Info
                  label="Cron"
                  value={
                    <span className="flex flex-wrap items-center gap-2">
                      <code className="rounded bg-[var(--bm-control-bg-soft)] px-2 py-0.5 font-mono text-xs text-[var(--bm-text)]">
                        {task.scheduleCron || '-'}
                      </code>
                      {canManage && task.taskKey ? (
                        <button
                          type="button"
                          aria-label="修改执行计划"
                          className="bm-edit-outline-button inline-grid h-7 w-7 place-items-center rounded-md border transition"
                          onClick={() => {
                            setCron(task.scheduleCron ?? '');
                            setEditing(true);
                          }}
                        >
                          <Pencil size={13} />
                        </button>
                      ) : null}
                    </span>
                  }
                />
              ) : (
                <Info label="执行计划" value={taskSchedule(task)} />
              )}
              {task.type === 'veeam' ? <Info label="保留策略" value={retention} /> : null}
            </div>
          </section>

          <section className="bm-card-hover overflow-hidden rounded-xl" style={panelStyle}>
            <div className="border-b border-[var(--bm-border)] px-4 py-3 text-sm font-semibold text-[var(--bm-text)]">
              配置详情
            </div>
            {configDetails.length === 0 ? (
              <div className="px-4 py-6 text-sm text-[var(--bm-text-muted)]">暂无配置详情</div>
            ) : (
              <div className="divide-y divide-dashed divide-[var(--bm-border)]/60">
                {configDetails.map(row => (
                  <div
                    key={row.label}
                    className="grid gap-1 px-4 py-2.5 sm:grid-cols-[150px_minmax(0,1fr)] sm:gap-4"
                  >
                    <span className="text-xs text-[var(--bm-text-muted)]">{row.label}</span>
                    {row.values ? (
                      <span className="flex flex-wrap gap-1.5">
                        {row.values.length === 0 ? (
                          <span className="text-sm text-[var(--bm-text)]">-</span>
                        ) : (
                          row.values.map(value => (
                            <Tag key={value} tone="info">
                              {value}
                            </Tag>
                          ))
                        )}
                      </span>
                    ) : (
                      <span className="break-all text-sm text-[var(--bm-text)]">{row.value}</span>
                    )}
                  </div>
                ))}
              </div>
            )}
          </section>

          <section
            ref={executionCardRef}
            className="bm-card-hover flex min-h-0 shrink-0 flex-col overflow-hidden rounded-xl"
            style={{ ...panelStyle, height: executionCardHeight }}
          >
            <div
              data-card-heading
              className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--bm-border)] px-4 py-3"
            >
              <div className="text-sm font-semibold text-[var(--bm-text)]">执行记录</div>
              <div className="flex items-center gap-3">
                <span className="text-xs text-[var(--bm-text-muted)]">
                  共 {executions.length} 条
                </span>
                {executions.length > 6 ? (
                  <button
                    type="button"
                    className={`bm-detail-toggle ${
                      expanded ? 'bm-detail-toggle-collapse' : 'bm-detail-toggle-expand'
                    }`}
                    onClick={toggleExecutionExpanded}
                  >
                    {expanded ? '收起' : `展开全部 ${executions.length} 条`}
                  </button>
                ) : null}
              </div>
            </div>
            {executions.length === 0 ? (
              <EmptyState className="flex-1">暂无执行记录</EmptyState>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full min-w-[860px] text-center text-sm">
                  <thead className="bg-[var(--bm-table-head-bg)] text-center text-xs font-semibold text-[var(--bm-text-muted)]">
                    <tr>
                      <th className="px-4 py-3 font-semibold">序号</th>
                      <th className="px-4 py-3 font-semibold">执行时间</th>
                      <th className="px-4 py-3 font-semibold">结束时间</th>
                      <th className="px-4 py-3 font-semibold">状态</th>
                      <th className="px-4 py-3 font-semibold">耗时</th>
                      <th className="px-4 py-3 font-semibold">触发</th>
                      <th className="px-4 py-3 font-semibold">备份大小</th>
                      <th className="px-4 py-3 font-semibold">操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {visibleExecutions.map((execution, index) => (
                      <tr
                        key={execution.id}
                        className="border-t border-[var(--bm-border)]/60 transition bm-table-row-hover"
                      >
                        <td className="px-4 py-3 text-sm text-[var(--bm-text-muted)]">
                          {index + 1}
                        </td>
                        <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                          {formatTime(execution.creationTime)}
                        </td>
                        <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                          {formatTime(execution.endTime)}
                        </td>
                        <td className="px-4 py-3">
                          <ExecutionStateTag state={execution.state} />
                        </td>
                        <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                          {executionDuration(execution)}
                        </td>
                        <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                          {execution.trigger === 'manual'
                            ? '手动'
                            : execution.trigger === 'scheduled'
                              ? '定时'
                              : '-'}
                        </td>
                        <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                          {sizeLabel(execution.transferredSize || execution.processedSize)}
                        </td>
                        <td className="px-4 py-3">
                          <Button variant="ghost" size="sm" onClick={() => setDetail(execution)}>
                            详情
                          </Button>
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

      <Dialog open={editing} onOpenChange={setEditing}>
        <DialogContent className="max-w-xl">
          <DialogHeader className="flex flex-row items-center gap-3 space-y-0 text-left">
            <span className="grid h-10 w-10 shrink-0 place-items-center rounded-xl border border-[var(--bm-border)] bg-[var(--bm-control-bg-soft)] text-[var(--bm-accent-text)]">
              <Clock size={18} />
            </span>
            <DialogTitle className="min-w-0">编辑执行计划 - {task?.name ?? ''}</DialogTitle>
          </DialogHeader>
          <div className="space-y-5">
            <div className="grid gap-2 sm:grid-cols-[64px_minmax(0,1fr)] sm:items-start sm:gap-3">
              <label
                htmlFor="task-cron-input"
                className="pt-2 text-sm text-[var(--bm-text-muted)] sm:text-right"
              >
                Cron
              </label>
              <div>
                <Input
                  id="task-cron-input"
                  value={cron}
                  onChange={event => setCron(event.target.value)}
                  placeholder="0 23 * * *"
                  className="font-mono"
                />
                {cronPreview ? (
                  <div className="mt-2 text-sm text-emerald-500">{cronPreview}</div>
                ) : null}
              </div>
            </div>
            <div className="grid gap-2 border-t border-[var(--bm-border)] pt-4 sm:grid-cols-[64px_minmax(0,1fr)] sm:gap-3">
              <div className="text-sm text-[var(--bm-text-muted)] sm:text-right">下发</div>
              <p className="text-sm leading-6 text-[var(--bm-text-muted)]">
                保存后将同步下发至 Agent，直接修改服务器 crontab 中该任务（key 标签定位）的执行计划
              </p>
            </div>
          </div>
          <DialogFooter className="gap-2 sm:space-x-0">
            <Button variant="outline" onClick={() => setEditing(false)}>
              取消
            </Button>
            <Button
              className="bm-primary-soft-button"
              disabled={saveSchedule.isPending || !cron.trim()}
              onClick={() => {
                if (cron.trim() === (task?.scheduleCron ?? '').trim()) {
                  toast.info('当前执行计划未变化');
                  return;
                }
                saveSchedule.mutate();
              }}
            >
              {saveSchedule.isPending ? '下发中' : '保存并下发'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={deleteOpen}
        title="删除任务"
        description="删除后不可恢复，关联的数据也会一并移除"
        detail={`确认删除任务「${task?.name ?? ''}」吗？其全部执行记录与告警将一并删除。`}
        confirmText="删除任务"
        icon="delete"
        horizontalHeader
        busy={remove.isPending}
        onOpenChange={setDeleteOpen}
        onConfirm={() => remove.mutate()}
      />

      <ExecutionDetailDialog
        execution={detail}
        task={task}
        onOpenChange={open => !open && setDetail(null)}
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

type ConfigDetail = {
  label: string;
  value?: string;
  values?: string[];
};

function configRows(task: BackupTask) {
  const config = task.config ?? {};
  const rows: ConfigDetail[] = [];
  const text = (key: string) => {
    const value = config[key];
    return typeof value === 'string'
      ? value.trim()
      : typeof value === 'number' || typeof value === 'boolean'
        ? String(value)
        : '';
  };
  const push = (label: string, value: string) => {
    rows.push({ label, value: value || '-' });
  };

  if (task.type !== 'veeam') {
    const taskKey = task.taskKey || text('key');
    if (taskKey) rows.push({ label: '任务标识 key', value: taskKey });
  }

  if (task.type === 'veeam') {
    push('Repository', text('repository'));
    push('计划', text('schedule'));
    push('Active Full', text('activeFull'));
    push('Retention', text('retention'));
    push('Compression', text('compression'));
    rows.push({
      label: '备份对象',
      values: Array.isArray(config.objects)
        ? config.objects.filter(item => typeof item === 'string' && item.trim())
        : [],
    });
    return rows;
  }

  const taskName = text('taskName');
  if (taskName) push('计划任务', taskName.replace(/^\\+/, ''));
  push('脚本', text('script'));
  const backupDir = text('backupDir');
  if (backupDir) push('产物目录', backupDir);
  const match = text('match');
  if (match) push('产物匹配', match);
  const statusFile = text('statusFile');
  if (statusFile) push('状态文件', statusFile);
  return rows;
}
