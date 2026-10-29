import { api } from '@/lib/api';
import type {
  AuthProviderSetting,
  EmailSetting,
  Permission,
  Role,
  RoleInput,
  SettingsUser,
  SystemBaseConfig,
  UserGroup,
  UserGroupInput,
  UserInput,
} from '@/features/settings/types';

type List<T> = { items: T[]; total: number };

export const fetchSystemBaseConfig = () => api<SystemBaseConfig>('/api/settings/base');

export const updateSystemBaseConfig = (body: SystemBaseConfig) =>
  api<SystemBaseConfig>('/api/settings/base', {
    method: 'PUT',
    body: JSON.stringify(body),
  });

export const fetchPermissions = () => api<List<Permission>>('/api/settings/permissions');

export const fetchRoles = () => api<List<Role>>('/api/settings/roles');

export const createRole = (body: RoleInput) =>
  api<Role>('/api/settings/roles', { method: 'POST', body: JSON.stringify(body) });

export const updateRole = (id: string, body: RoleInput) =>
  api<Role>(`/api/settings/roles/${id}`, { method: 'PUT', body: JSON.stringify(body) });

export const updateRoleDisabled = (id: string, disabled: boolean) =>
  api<Role>(`/api/settings/roles/${id}/disabled`, {
    method: 'POST',
    body: JSON.stringify({ disabled }),
  });

export const deleteRole = (id: string) =>
  api<void>(`/api/settings/roles/${id}`, { method: 'DELETE' });

export const fetchUserGroups = () => api<List<UserGroup>>('/api/settings/user-groups');

export const createUserGroup = (body: UserGroupInput) =>
  api<UserGroup>('/api/settings/user-groups', {
    method: 'POST',
    body: JSON.stringify(body),
  });

export const updateUserGroup = (id: string, body: UserGroupInput) =>
  api<UserGroup>(`/api/settings/user-groups/${id}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });

export const deleteUserGroup = (id: string) =>
  api<void>(`/api/settings/user-groups/${id}`, { method: 'DELETE' });

export const fetchSettingsUsers = () => api<List<SettingsUser>>('/api/settings/users');

export const createSettingsUser = (body: UserInput) =>
  api<SettingsUser>('/api/settings/users', {
    method: 'POST',
    body: JSON.stringify(body),
  });

export const updateSettingsUser = (id: string, body: UserInput) =>
  api<SettingsUser>(`/api/settings/users/${id}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });

export const deleteSettingsUser = (id: string) =>
  api<void>(`/api/settings/users/${id}`, { method: 'DELETE' });

export const updateSettingsUserDisabled = (id: string, disabled: boolean) =>
  api<SettingsUser>(`/api/settings/users/${id}/disabled`, {
    method: 'POST',
    body: JSON.stringify({ disabled }),
  });

export const fetchAuthProvider = () => api<AuthProviderSetting>('/api/settings/auth-provider');

export const saveAuthProvider = (body: {
  name: string;
  enabled: boolean;
  clearConfig: boolean;
  config: Record<string, unknown>;
}) =>
  api<AuthProviderSetting>('/api/settings/auth-provider', {
    method: 'PUT',
    body: JSON.stringify(body),
  });

export const testAuthProvider = () =>
  api<{ status: string; matchedUsers: number }>('/api/settings/auth-provider/test', {
    method: 'POST',
  });

export const fetchWeComProvider = () => api<AuthProviderSetting>('/api/settings/auth/wecom');

export const saveWeComProvider = (body: {
  name: string;
  enabled: boolean;
  clearConfig: boolean;
  config: Record<string, unknown>;
}) =>
  api<AuthProviderSetting>('/api/settings/auth/wecom', {
    method: 'PUT',
    body: JSON.stringify(body),
  });

export const fetchEmailSetting = () => api<EmailSetting>('/api/settings/email');

export const saveEmailSetting = (body: {
  passwordResetEnabled: boolean;
  clearConfig: boolean;
  config: Record<string, unknown>;
}) =>
  api<EmailSetting>('/api/settings/email', {
    method: 'PUT',
    body: JSON.stringify(body),
  });

export const testEmailSetting = (to: string) =>
  api<{ status: string }>('/api/settings/email/test', {
    method: 'POST',
    body: JSON.stringify({ to }),
  });
