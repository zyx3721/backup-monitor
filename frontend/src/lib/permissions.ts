export const PERM = {
  dashboardRead: 'backup.dashboard.read',
  hostsRead: 'backup.hosts.read',
  hostsManage: 'backup.hosts.manage',
  tasksRead: 'backup.tasks.read',
  tasksManage: 'backup.tasks.manage',
  executionsRead: 'backup.executions.read',
  executionsManage: 'backup.executions.manage',
  alertsRead: 'backup.alerts.read',
  alertsManage: 'backup.alerts.manage',
  notificationsRead: 'backup.notifications.read',
  notificationsManage: 'backup.notifications.manage',
  agentsRead: 'backup.agents.read',
  agentsManage: 'backup.agents.manage',
  auditRead: 'audit.read',
  auditManage: 'audit.manage',
  settingsBaseRead: 'settings.base.read',
  settingsBaseManage: 'settings.base.manage',
  settingsUsersRead: 'settings.users.read',
  settingsUsersManage: 'settings.users.manage',
  settingsAuthRead: 'settings.auth.read',
  settingsAuthManage: 'settings.auth.manage',
  settingsNotificationsRead: 'settings.notifications.read',
  settingsNotificationsManage: 'settings.notifications.manage',
} as const;

import type { AuthUser } from '@/lib/session';

// userHasPermission 判断用户是否持有某项权限，管理员角色默认全通过
export function userHasPermission(user: AuthUser | null, permission: string) {
  if (!user) return false;
  if (user.role === 'admin') return true;
  return user.permissions?.includes(permission) ?? false;
}

export function userHasAnyPermission(user: AuthUser | null, permissions: string[]) {
  return permissions.some(permission => userHasPermission(user, permission));
}

export const SETTINGS_READ_PERMISSIONS = [
  PERM.settingsBaseRead,
  PERM.settingsUsersRead,
  PERM.settingsAuthRead,
  PERM.settingsNotificationsRead,
];
