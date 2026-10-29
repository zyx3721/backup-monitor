import type { ExportColumn } from '@/lib/export-data';
import {
  alertTypeLabel,
  formatTime,
  methodLabel,
  platformLabel,
  roleLabel,
  taskSchedule,
} from './components';
import { executionDuration, sizeLabel } from './execution-utils';
import type { Alert, BackupExecution, BackupTask, Host } from './types';

const hostStatusLabels: Record<Host['status'], string> = {
  normal: '正常',
  abnormal: '异常',
  offline: '离线',
};

const taskStatusLabels: Record<BackupTask['status'], string> = {
  normal: '正常',
  running: '运行中',
  failed: '失败',
  not_run: '未执行',
  no_data: '无数据',
};

const executionStateLabels: Record<BackupExecution['state'], string> = {
  success: '成功',
  failed: '失败',
  running: '运行中',
};

const alertLevelLabels: Record<Alert['level'], string> = {
  warning: '警告',
  critical: '严重',
};

const alertStatusLabels: Record<Alert['status'], string> = {
  active: '未处理',
  resolved: '已恢复',
};

function indexColumn<T>(): ExportColumn<T> {
  return { id: 'index', header: '序号', value: (_item, index) => index + 1 };
}

export const hostExportColumns: ExportColumn<Host>[] = [
  indexColumn<Host>(),
  { id: 'hostname', header: '主机', value: item => item.hostname },
  { id: 'ip', header: 'IP 地址', value: item => item.ip || '-' },
  { id: 'platform', header: '平台', value: item => platformLabel(item.platform) },
  { id: 'os', header: '操作系统', value: item => item.os || '-' },
  { id: 'role', header: '角色', value: item => roleLabel(item.role) },
  { id: 'agent', header: 'Agent 版本', value: item => item.agentVersion || '-' },
  { id: 'taskCount', header: '任务数', value: item => item.taskCount },
  { id: 'lastReportTime', header: '最后上报', value: item => formatTime(item.lastReportTime) },
  { id: 'status', header: '状态', value: item => hostStatusLabels[item.status] ?? item.status },
];

export const taskExportColumns: ExportColumn<BackupTask>[] = [
  indexColumn<BackupTask>(),
  { id: 'name', header: '任务', value: item => item.name },
  { id: 'hostName', header: '主机', value: item => item.hostName },
  { id: 'hostIp', header: 'IP', value: item => item.hostIp || '-' },
  { id: 'type', header: '备份方式', value: item => methodLabel(item.type) },
  { id: 'schedule', header: '执行计划', value: item => taskSchedule(item) },
  { id: 'lastSuccessTime', header: '最近成功', value: item => formatTime(item.lastSuccessTime) },
  { id: 'status', header: '状态', value: item => taskStatusLabels[item.status] ?? item.status },
];

export const executionExportColumns: ExportColumn<BackupExecution>[] = [
  indexColumn<BackupExecution>(),
  { id: 'taskName', header: '任务', value: item => item.taskName || '-' },
  { id: 'hostName', header: '主机', value: item => item.hostName || '-' },
  {
    id: 'taskType',
    header: '备份方式',
    value: item => (item.taskType ? methodLabel(item.taskType) : '-'),
  },
  { id: 'state', header: '状态', value: item => executionStateLabels[item.state] ?? item.state },
  {
    id: 'trigger',
    header: '触发',
    value: item =>
      item.trigger === 'manual' ? '手动' : item.trigger === 'scheduled' ? '定时' : '-',
  },
  {
    id: 'startTime',
    header: '执行时间',
    value: item => formatTime(item.startTime || item.creationTime),
  },
  { id: 'endTime', header: '结束时间', value: item => formatTime(item.endTime) },
  { id: 'duration', header: '耗时', value: item => executionDuration(item) },
  {
    id: 'backupSize',
    header: '备份大小',
    value: item => sizeLabel(item.transferredSize || item.processedSize),
  },
];

export const alertExportColumns: ExportColumn<Alert>[] = [
  indexColumn<Alert>(),
  { id: 'createdAt', header: '告警时间', value: item => formatTime(item.createdAt) },
  { id: 'level', header: '级别', value: item => alertLevelLabels[item.level] ?? item.level },
  { id: 'hostName', header: '主机', value: item => item.hostName || '-' },
  { id: 'type', header: '告警类型', value: item => alertTypeLabel(item.type) },
  { id: 'message', header: '告警内容', value: item => item.message || '-' },
  { id: 'status', header: '状态', value: item => alertStatusLabels[item.status] ?? item.status },
  { id: 'resolvedAt', header: '恢复时间', value: item => formatTime(item.resolvedAt) },
];
