export type HostStatus = 'normal' | 'abnormal' | 'offline';
export type TaskStatus = 'normal' | 'running' | 'failed' | 'not_run' | 'no_data';
export type BackupMethod = 'veeam' | 'shell' | 'powershell' | 'batch' | 'exe';
export type AlertStatus = 'active' | 'resolved';

export type Host = {
  id: string;
  hostname: string;
  nodeId?: string;
  ip: string;
  os?: string;
  role?: string;
  agentVersion?: string;
  agentStartedAt?: string;
  platform?: string;
  status: HostStatus;
  lastReportTime: string;
  taskCount: number;
};

export type HostDetail = Host & { tasks: BackupTask[] };

export type BackupTask = {
  id: string;
  hostId: string;
  hostName: string;
  hostIp: string;
  name: string;
  taskKey?: string;
  type: BackupMethod;
  tool?: string;
  schedule: string;
  scheduleCron?: string;
  status: TaskStatus;
  lastExecutionTime?: string;
  lastSuccessTime?: string;
  consecutiveFailures: number;
  successRate: number;
  latestSize?: string;
  retention?: string;
  config: Record<string, unknown>;
};

export type BackupExecution = {
  id: string;
  taskId: string;
  taskName?: string;
  hostId?: string;
  hostName?: string;
  taskType?: BackupMethod;
  sessionId?: string;
  jobId?: string;
  oibId?: string;
  state: 'success' | 'running' | 'failed';
  trigger?: 'scheduled' | 'manual';
  creationTime: string;
  startTime?: string;
  endTime?: string;
  processedSize?: string;
  transferredSize?: string;
  artifactPath?: string;
  exitCode?: number;
  attempt?: number;
  maxAttempts?: number;
  isLastAttempt?: boolean;
  message?: string;
};

export type Alert = {
  id: string;
  hostId: string;
  taskId?: string;
  hostName: string;
  taskName?: string;
  taskType?: BackupMethod;
  type: 'backup_failed' | 'not_run' | 'report_timeout';
  level: 'warning' | 'critical';
  status: AlertStatus;
  createdAt: string;
  resolvedAt?: string;
  message: string;
};

export type StatusSummary = { status: TaskStatus; count: number };
export type MethodSummary = {
  method: BackupMethod;
  total: number;
  normal: number;
  failed: number;
  notRun: number;
};
export type TrendPoint = { date: string; success: number; failed: number; other: number };

export type Dashboard = {
  hostCount: number;
  hostNormal: number;
  hostAbnormal: number;
  hostOffline: number;
  taskCount: number;
  todaySuccess: number;
  todayFailed: number;
  todayTotal: number;
  successRate: number;
  abnormalTasks: number;
  notRunTasks: number;
  activeAlerts: number;
  statusSummary: StatusSummary[];
  methodSummary: MethodSummary[];
  trend: TrendPoint[];
  recentAbnormalTasks: BackupTask[];
  recentAlerts: Alert[];
};

export type AgentGlobalConfig = {
  reportIntervalSec: number;
  httpTimeoutSec: number;
  wsBackoffMinSec: number;
  wsBackoffMaxSec: number;
};

export type BackupSettings = {
  reportTimeoutMinutes: number;
  offlineTimeoutMinutes: number;
  notRunGraceMinutes: number;
  scheduleMatchWindowMinutes: number;
  schedulerIntervalSec: number;
  dedupeAlerts: boolean;
  retentionDays: number;
  auditRetentionDays: number;
  notifyRetentionDays: number;
  requireAgentToken: boolean;
  agent: AgentGlobalConfig;
};

export type AgentProfileConfig = {
  reportIntervalSec: number;
  collectCrontab: boolean;
  collectVeeam: boolean;
  artifactCheck: boolean;
  labelPrefix: string;
  includeKeywords: string[];
  includePaths: string[];
  excludeKeywords: string[];
  maxArtifactAgeHours: number;
  heartbeatIntervalSec: number;
  artifactMaxDepth: number;
  maxSessionsPerJob: number;
  maxSessionInfoCalls: number;
  maxJobInfoCalls: number;
  manualTriggerMinutes: number;
  artifactPreToleranceMinutes: number;
  autoUpdateMinutes: number;
};

export type AgentProfile = {
  id: string;
  name: string;
  hostname: string;
  description?: string;
  enabled: boolean;
  priority: number;
  config: AgentProfileConfig;
  createdAt: string;
  updatedAt: string;
};

export type AgentToken = {
  id: string;
  token: string;
  name: string;
  hostname?: string;
  enabled: boolean;
  lastSeenAt?: string;
  createdAt: string;
};

export type AgentBinaryMeta = {
  platform: 'linux' | 'windows';
  arch: string;
  variant: string;
  goVersion: string;
  version: string;
  sha256: string;
  size: number;
  uploadedAt: string;
};

export type AgentBinaryTargetCount = {
  total: number;
  online: number;
};

export type AgentBinaryTargets = {
  linux: AgentBinaryTargetCount;
  windows: AgentBinaryTargetCount;
  unknown: number;
};

export type NotifyChannel = {
  id: string;
  name: string;
  type: 'mail' | 'webhook' | 'wecom' | 'dingtalk' | 'feishu';
  eventType: string[];
  level: 'all' | 'info' | 'warning' | 'critical';
  remainDays: number;
  triggerGroup: string;
  remark?: string;
  enabled: boolean;
  recover: boolean;
  config: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
  target?: string;
};

export type NotificationRecord = {
  id: string;
  channel: string;
  channelName?: string;
  type: string;
  level?: string;
  recipients?: string;
  subject?: string;
  status: 'sent' | 'failed';
  error?: string;
  hostName?: string;
  taskName?: string;
  message?: string;
  createdAt: string;
};

export type PushResult = {
  hostname: string;
  ip: string;
  agentVersion: string;
  agentStartedAt?: string;
  lastReportTime?: string;
  restarted?: boolean;
  delivered: boolean;
  ok: boolean;
  message: string;
  skipped?: boolean;
  reason?: string;
  warning?: string;
  arch?: string;
  osBuild?: number;
  version?: string;
  variant?: string;
};

export type PushSummary = {
  total: number;
  delivered: number;
  ok: number;
  failed: number;
  skipped: number;
};
