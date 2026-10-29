import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import {
  AlertTriangle,
  BellRing,
  ChevronRight,
  FileText,
  FolderOpen,
  Monitor,
  ShieldCheck,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { fetchDashboard } from './api';
import {
  EmptyState,
  MethodTag,
  PageHeader,
  Panel,
  RelativeTime,
  TaskStatusTag,
  formatTime,
  panelStyle,
  taskSchedule,
} from './components';
import { StatusDonut } from './StatusDonut';
import { TrendChart } from './TrendChart';
import type { Alert, BackupTask, Dashboard } from './types';

export function DashboardPage() {
  const { data, isLoading } = useQuery({
    queryKey: ['backup', 'dashboard'],
    queryFn: fetchDashboard,
  });

  if (isLoading || !data) {
    return <EmptyState>正在加载概览数据</EmptyState>;
  }

  return (
    <div className="flex min-h-0 flex-col gap-5">
      <PageHeader
        title="概览"
        description="主机、任务、执行与告警的整体运行态势"
        className="mb-0"
      />
      <StatCards data={data} />
      <div className="grid grid-cols-1 gap-5 xl:grid-cols-[minmax(0,2fr)_minmax(340px,1fr)]">
        <TrendPanel data={data} />
        <StatusPanel data={data} />
      </div>
      <div className="grid grid-cols-1 gap-5 xl:grid-cols-[minmax(0,2fr)_minmax(340px,1fr)]">
        <AbnormalPanel data={data} />
        <RecentAlertsPanel data={data} />
      </div>
    </div>
  );
}

function StatCards({ data }: { data: Dashboard }) {
  const cards: Array<{
    label: string;
    value: string | number;
    suffix?: string;
    hint: string;
    icon: LucideIcon;
    color: string;
    to: string;
    search?: Record<string, string>;
  }> = [
    {
      label: '监控主机',
      value: data.hostCount,
      hint: `${data.hostNormal} 正常${
        data.hostAbnormal > 0 ? ` · ${data.hostAbnormal} 异常` : ''
      } · ${data.hostOffline} 离线`,
      icon: Monitor,
      color: '#38bdf8',
      to: '/hosts',
    },
    {
      label: '备份任务',
      value: data.taskCount,
      hint: `${data.statusSummary.find(item => item.status === 'running')?.count ?? 0} 个运行中`,
      icon: FolderOpen,
      color: '#a78bfa',
      to: '/tasks',
    },
    {
      label: '今日备份',
      value: data.todayTotal,
      hint: `成功 ${data.todaySuccess} · 失败 ${data.todayFailed}`,
      icon: FileText,
      color: '#3b82f6',
      to: '/executions',
    },
    {
      label: '备份成功率',
      value: data.successRate,
      suffix: '%',
      hint: '今日执行',
      icon: ShieldCheck,
      color: '#22d3ee',
      to: '/executions',
    },
    {
      label: '异常任务',
      value: data.abnormalTasks,
      hint: '需关注',
      icon: AlertTriangle,
      color: '#f87171',
      to: '/tasks',
      search: { status: 'failed' },
    },
    {
      label: '未处理告警',
      value: data.activeAlerts,
      hint: '待处理',
      icon: BellRing,
      color: '#fb7185',
      to: '/alerts',
    },
  ];

  return (
    <div className="grid grid-cols-2 gap-4 lg:grid-cols-3 xl:grid-cols-6">
      {cards.map(card => {
        const Icon = card.icon;
        return (
          <Link
            key={card.label}
            to={card.to as never}
            search={card.search as never}
            className="bm-card-hover group min-h-[7.75rem] rounded-xl p-4"
            style={panelStyle}
          >
            <div className="flex items-center justify-between gap-2">
              <span className="text-xs text-[var(--bm-text-muted)]">{card.label}</span>
              <span
                className="grid h-8 w-8 place-items-center rounded-lg"
                style={{ color: card.color, background: `${card.color}1f` }}
              >
                <Icon size={16} />
              </span>
            </div>
            <div className="mt-3 flex items-baseline gap-1">
              <span className="font-mono text-2xl font-semibold tabular-nums text-[var(--bm-text)]">
                {card.value}
              </span>
              {card.suffix ? (
                <span className="text-sm text-[var(--bm-text-muted)]">{card.suffix}</span>
              ) : null}
            </div>
            <div className="mt-2 text-[11px] text-[var(--bm-text-muted)]">{card.hint}</div>
          </Link>
        );
      })}
    </div>
  );
}

function TrendPanel({ data }: { data: Dashboard }) {
  return (
    <Panel
      title="备份执行趋势"
      extra={<span className="text-xs text-[var(--bm-text-muted)]">近 14 天</span>}
      className="min-h-[25rem]"
    >
      <TrendChart points={data.trend} />
    </Panel>
  );
}

function StatusPanel({ data }: { data: Dashboard }) {
  return (
    <Panel
      title="任务状态分布"
      extra={
        <span className="text-xs text-[var(--bm-text-muted)]">共 {data.taskCount} 个任务</span>
      }
      className="min-h-[25rem]"
      bodyClassName="flex items-center px-5 py-4"
    >
      <div className="w-full">
        <StatusDonut summary={data.statusSummary} total={data.taskCount} />
      </div>
    </Panel>
  );
}

function AbnormalPanel({ data }: { data: Dashboard }) {
  return (
    <Panel
      title="异常任务"
      className="h-[22rem]"
      bodyClassName="bm-hidden-scrollbar overscroll-contain overflow-y-auto"
      extra={
        <Link
          to="/tasks"
          search={{ status: 'failed' } as never}
          className="inline-flex items-center gap-1 text-xs text-[var(--bm-accent-text)] transition hover:underline"
        >
          查看全部 <ChevronRight size={13} />
        </Link>
      }
    >
      <div className="overflow-x-auto">
        <table className="w-full table-auto text-center text-sm">
          <thead className="sticky top-0 z-10 bg-[var(--bm-table-head-bg)] text-center text-xs font-semibold text-[var(--bm-text-muted)]">
            <tr>
              <th className="whitespace-nowrap px-4 py-3 font-semibold">序号</th>
              <th className="whitespace-nowrap px-4 py-3 font-semibold">主机</th>
              <th className="whitespace-nowrap px-4 py-3 font-semibold">任务</th>
              <th className="whitespace-nowrap px-4 py-3 font-semibold">备份方式</th>
              <th className="whitespace-nowrap px-4 py-3 font-semibold">连续失败</th>
              <th className="whitespace-nowrap px-4 py-3 font-semibold">最近成功</th>
              <th className="whitespace-nowrap px-4 py-3 font-semibold">状态</th>
            </tr>
          </thead>
          <tbody>
            {data.recentAbnormalTasks.map((task, index) => (
              <AbnormalRow key={task.id} task={task} index={index} />
            ))}
            {data.recentAbnormalTasks.length === 0 ? (
              <tr>
                <td
                  colSpan={7}
                  className="px-4 py-12 text-center text-xs text-[var(--bm-text-muted)]"
                >
                  暂无异常任务
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </Panel>
  );
}

function AbnormalRow({ task, index }: { task: BackupTask; index: number }) {
  return (
    <tr className="border-t border-[var(--bm-border)] transition bm-table-row-hover">
      <td className="px-3 py-3 text-xs text-[var(--bm-text-muted)]">{index + 1}</td>
      <td className="px-3 py-3 text-xs text-[var(--bm-text)]">{task.hostName}</td>
      <td className="px-3 py-3">
        <div className="text-xs font-medium text-[var(--bm-text)]">{task.name}</div>
        <div className="mt-0.5 font-mono text-[10px] text-[var(--bm-text-muted)]">
          {taskSchedule(task)}
        </div>
      </td>
      <td className="px-3 py-3">
        <MethodTag method={task.type} />
      </td>
      <td className="px-3 py-3 text-xs text-[var(--bm-text-muted)]">
        {task.consecutiveFailures} 次
      </td>
      <td className="whitespace-nowrap px-3 py-3 text-xs text-[var(--bm-text-muted)]">
        {task.lastSuccessTime ? formatTime(task.lastSuccessTime) : '-'}
      </td>
      <td className="px-3 py-3">
        <TaskStatusTag status={task.status} />
      </td>
    </tr>
  );
}

function RecentAlertsPanel({ data }: { data: Dashboard }) {
  return (
    <Panel
      title="最近告警"
      className="h-[22rem]"
      bodyClassName="bm-hidden-scrollbar overscroll-contain overflow-y-auto"
      extra={
        <Link
          to="/alerts"
          className="inline-flex items-center gap-1 text-xs text-[var(--bm-accent-text)] transition hover:underline"
        >
          查看全部 <ChevronRight size={13} />
        </Link>
      }
    >
      <div className="divide-y divide-[var(--bm-border)]">
        {data.recentAlerts.map(alert => (
          <RecentAlertItem key={alert.id} alert={alert} />
        ))}
        {data.recentAlerts.length === 0 ? (
          <div className="px-4 py-12 text-center text-xs text-[var(--bm-text-muted)]">暂无告警</div>
        ) : null}
      </div>
    </Panel>
  );
}

function RecentAlertItem({ alert }: { alert: Alert }) {
  const accent = alertAccent(alert);
  const targetLabel = alert.taskId ? '查看任务详情' : '查看主机详情';
  const content = (
    <>
      <span
        aria-hidden="true"
        className="absolute bottom-3 left-0 top-3 w-[3px] rounded-full transition-all duration-200 group-hover:bottom-2 group-hover:top-2 group-hover:w-[4px]"
        style={{ background: accent, boxShadow: `0 0 10px ${accent}33` }}
      />
      <div className="flex items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2">
          <span className="truncate text-xs font-medium text-[var(--bm-text)] group-hover:text-[var(--bm-accent-text)]">
            {alert.hostName || '未知主机'}
          </span>
          <span className="truncate text-[11px] text-[var(--bm-text-muted)]">
            {alert.taskName || '主机'}
          </span>
        </div>
        <span className="flex shrink-0 items-center gap-1 text-[11px] text-[var(--bm-text-muted)]">
          <RelativeTime value={alert.createdAt} />
          <ChevronRight
            size={13}
            aria-hidden="true"
            className="-translate-x-1 opacity-0 transition group-hover:translate-x-0 group-hover:opacity-100"
          />
        </span>
      </div>
      <div className="mt-1 line-clamp-1 text-xs text-[var(--bm-text-muted)]">{alert.message}</div>
    </>
  );
  const className =
    'group relative block rounded-lg py-3 pl-4 pr-2.5 transition duration-200 hover:bg-[var(--bm-control-bg-soft)] focus-visible:bg-[var(--bm-control-bg-soft)] focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[var(--bm-accent-text)]';

  if (alert.taskId) {
    return (
      <Link
        to="/tasks/$taskID"
        params={{ taskID: alert.taskId }}
        search={{}}
        aria-label={`${targetLabel}：${alert.taskName || alert.hostName}`}
        className={className}
      >
        {content}
      </Link>
    );
  }
  return (
    <Link
      to="/hosts/$hostID"
      params={{ hostID: alert.hostId }}
      aria-label={`${targetLabel}：${alert.hostName}`}
      className={className}
    >
      {content}
    </Link>
  );
}

function alertAccent(alert: Alert) {
  if (alert.status === 'resolved') return '#34d399';
  return alert.level === 'critical' ? '#f87171' : '#fbbf24';
}
