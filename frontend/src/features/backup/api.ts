import { api, readApiError } from '@/lib/api';
import { getAuthToken } from '@/lib/session';
import type {
  AgentBinaryMeta,
  AgentBinaryTargets,
  AgentProfile,
  AgentToken,
  Alert,
  BackupExecution,
  BackupSettings,
  BackupTask,
  Dashboard,
  Host,
  HostDetail,
  NotificationRecord,
  NotifyChannel,
  PushResult,
  PushSummary,
} from './types';

export const fetchDashboard = () => api<Dashboard>('/api/backup/dashboard');

export const fetchHosts = (ids?: string[]) =>
  ids && ids.length > 0
    ? api<Host[]>(`/api/backup/hosts?ids=${encodeURIComponent(ids.join(','))}`)
    : api<Host[]>('/api/backup/hosts');
export const fetchHost = (id: string) => api<HostDetail>(`/api/backup/hosts/${id}`);
export const updateHostRole = (id: string, role: string | null) =>
  api<{ ok: boolean; role: string }>(`/api/backup/hosts/${id}/role`, {
    method: 'PUT',
    body: JSON.stringify({ role }),
  });
export const deleteHost = (id: string) =>
  api<{ ok: boolean }>(`/api/backup/hosts/${id}`, { method: 'DELETE' });

export const fetchTasks = () => api<BackupTask[]>('/api/backup/tasks');
export const fetchTask = (id: string) => api<BackupTask>(`/api/backup/tasks/${id}`);
export const fetchTaskExecutions = (id: string) =>
  api<BackupExecution[]>(`/api/backup/tasks/${id}/executions`);
export const deleteTask = (id: string) =>
  api<{ ok: boolean }>(`/api/backup/tasks/${id}`, { method: 'DELETE' });
export const updateTaskSchedule = (id: string, scheduleCron: string) =>
  api<{ ok: boolean; delivery: { delivered: boolean; ok: boolean; message: string } }>(
    `/api/backup/tasks/${id}/schedule`,
    { method: 'PUT', body: JSON.stringify({ scheduleCron }) }
  );

export const fetchExecutions = () => api<BackupExecution[]>('/api/backup/executions');
export const deleteExecution = (id: string) =>
  api<{ ok: boolean }>(`/api/backup/executions/${id}`, { method: 'DELETE' });

export const fetchAlerts = () => api<Alert[]>('/api/backup/alerts');
export const resolveAlert = (id: string) =>
  api<{ ok: boolean }>(`/api/backup/alerts/${id}/resolve`, { method: 'POST' });
export const deleteAlert = (id: string) =>
  api<{ ok: boolean }>(`/api/backup/alerts/${id}`, { method: 'DELETE' });

export const fetchChannels = () => api<NotifyChannel[]>('/api/backup/notify/channels');
export const createChannel = (body: Partial<NotifyChannel>) =>
  api<{ channel: NotifyChannel; target: string }>('/api/backup/notify/channels', {
    method: 'POST',
    body: JSON.stringify(body),
  });
export const updateChannel = (id: string, body: Partial<NotifyChannel>) =>
  api<{ channel: NotifyChannel; target: string }>(`/api/backup/notify/channels/${id}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
export const deleteChannel = (id: string) =>
  api<{ ok: boolean }>(`/api/backup/notify/channels/${id}`, { method: 'DELETE' });
export const testChannel = (id: string) =>
  api<{ ok: boolean; message: string }>(`/api/backup/notify/channels/${id}/test`, {
    method: 'POST',
  });
export const fetchNotificationRecords = (limit = 100, channelId?: string) => {
  const query = channelId ? `&channel=${encodeURIComponent(channelId)}` : '';
  return api<NotificationRecord[]>(`/api/backup/notifications?limit=${limit}${query}`);
};

export const fetchBackupSettings = () => api<BackupSettings>('/api/backup/settings');
export const saveBackupSettings = (body: BackupSettings, section?: string) =>
  api<BackupSettings>('/api/backup/settings', {
    method: 'PUT',
    body: JSON.stringify(section ? { ...body, section } : body),
  });

export const fetchProfiles = () => api<AgentProfile[]>('/api/backup/profiles');
export const createProfile = (body: Partial<AgentProfile>) =>
  api<AgentProfile>('/api/backup/profiles', { method: 'POST', body: JSON.stringify(body) });
export const updateProfile = (id: string, body: Partial<AgentProfile>) =>
  api<AgentProfile>(`/api/backup/profiles/${id}`, { method: 'PUT', body: JSON.stringify(body) });
export const deleteProfile = (id: string) =>
  api<{ ok: boolean }>(`/api/backup/profiles/${id}`, { method: 'DELETE' });

export const fetchTokens = () => api<AgentToken[]>('/api/backup/tokens');
export const createToken = (body: { name: string; hostname?: string; token?: string }) =>
  api<AgentToken>('/api/backup/tokens', { method: 'POST', body: JSON.stringify(body) });
export const updateToken = (
  id: string,
  body: { name: string; hostname?: string; enabled: boolean }
) =>
  api<{ ok: boolean }>(`/api/backup/tokens/${id}`, { method: 'PUT', body: JSON.stringify(body) });
export const deleteToken = (id: string) =>
  api<{ ok: boolean }>(`/api/backup/tokens/${id}`, { method: 'DELETE' });

export const fetchAgentBinaries = () =>
  api<{ items: AgentBinaryMeta[]; targets: AgentBinaryTargets }>('/api/backup/agent-binaries');
export const pushAgentUpdate = (body: { all: boolean; ips: string; platform: string }) =>
  api<{
    platform: 'linux' | 'windows';
    results: PushResult[];
    summary?: PushSummary;
  }>('/api/backup/agent-update/push', {
    method: 'PUT',
    body: JSON.stringify(body),
  });

export async function uploadAgentBinary(platform: string, version: string, file: File) {
  const headers = new Headers();
  headers.set('Content-Type', 'application/octet-stream');
  const token = getAuthToken();
  if (token) headers.set('Authorization', `Bearer ${token}`);
  const params = new URLSearchParams({ platform });
  if (version) params.set('version', version);
  const response = await fetch(`/api/backup/agent-binary?${params.toString()}`, {
    method: 'POST',
    headers,
    body: file,
  });
  if (!response.ok) {
    throw new Error(await readApiError(response, `上传失败：${response.status}`));
  }
  return (await response.json()) as AgentBinaryMeta;
}
