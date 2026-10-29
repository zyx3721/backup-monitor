export type UserConfigurationLabel = '用户名' | '用户群组' | '用户角色';

export function createdUserConfigurationMessage(label: UserConfigurationLabel, name: string) {
  return `${label} ${name} 已创建`;
}

export function updatedUserConfigurationMessage(label: UserConfigurationLabel, name: string) {
  return `${label} ${name} 配置已更新`;
}

export function deletedUserConfigurationMessage(label: UserConfigurationLabel, name: string) {
  return `${label} ${name} 已删除`;
}

export function statusUserConfigurationMessage(
  label: UserConfigurationLabel,
  name: string,
  disabled: boolean
) {
  return `${label} ${name} 已${disabled ? '禁用' : '启用'}`;
}

export function userConfigurationSaveError(
  error: unknown,
  fallback: string,
  duplicateError: string,
  label: UserConfigurationLabel,
  name: string
) {
  const message = error instanceof Error ? error.message : fallback;
  return message === duplicateError ? `${label} ${name} 已存在` : message;
}
