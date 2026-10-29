export type SettingsTab = 'base' | 'users' | 'auth' | 'findpwd';

export type SystemBaseConfig = {
  siteName: string;
  loginName: string;
  appName: string;
  appSubtitle: string;
  iconData: string;
  resetCodeTtlMinutes: number;
  resetCaptchaTtlMinutes: number;
  passwordResetSendCooldownMinutes: number;
  passwordResetRateLimitMinutes: number;
  wecomStateTtlMinutes: number;
  loginMaxFailures: number;
  loginLockoutMinutes: number;
  section?: 'brand' | 'security';
};

export type Permission = {
  key: string;
  name: string;
  description: string;
  category: string;
  impliedReadPermission?: string;
  impliedPermissions?: string[];
};

export type Role = {
  id: string;
  key: string;
  name: string;
  description: string;
  permissions: string[];
  builtin: boolean;
  disabled: boolean;
  createdAt: string;
  updatedAt: string;
};

export type SettingsUser = {
  id: string;
  username: string;
  displayName: string;
  email: string;
  role: string;
  source: 'local' | 'ldap';
  roles: Role[];
  directRoles: Role[];
  disabled: boolean;
  lastLoginAt?: string;
  createdAt: string;
  updatedAt: string;
  permissions: string[];
  effectiveUserRoles?: Role[];
  effectiveUserPermissions?: string[];
};

export type UserGroup = {
  id: string;
  name: string;
  description: string;
  disabled: boolean;
  members: SettingsUser[];
  roles: Role[];
  createdAt: string;
  updatedAt: string;
};

export type AuthProviderSetting = {
  id: string;
  type: string;
  name: string;
  enabled: boolean;
  config: Record<string, unknown>;
  updatedAt: string;
};

export type EmailSetting = {
  id: string;
  name: string;
  passwordResetEnabled: boolean;
  config: Record<string, unknown>;
  updatedAt: string;
};

export type UserInput = {
  username: string;
  displayName: string;
  email: string;
  password: string;
  roleKeys: string[];
  disabled: boolean;
};

export type RoleInput = Pick<Role, 'key' | 'name' | 'description' | 'permissions'>;

export type UserGroupInput = {
  name: string;
  description: string;
  disabled: boolean;
  memberIds: string[];
  roleKeys: string[];
};
