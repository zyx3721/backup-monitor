import {
  Plus,
  Search,
  ShieldCheck,
  ToggleLeft,
  ToggleRight,
  UserCog,
  UsersRound,
} from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { toast } from 'sonner';
import { showErrorToast } from '@/lib/toast-errors';
import {
  deleteRole,
  deleteSettingsUser,
  deleteUserGroup,
  fetchPermissions,
  fetchRoles,
  fetchSettingsUsers,
  fetchUserGroups,
  updateSettingsUserDisabled,
  updateUserGroup,
  updateRoleDisabled,
} from '@/features/settings/api';
import type { Permission, Role, SettingsUser, UserGroup } from '../types';
import { RoleEditorDialog } from '../dialogs/RoleEditorDialog';
import { GroupEditorDialog, UserEditorDialog } from '../dialogs/UserSettingsDialogs';
import {
  deletedUserConfigurationMessage,
  statusUserConfigurationMessage,
} from '../user-configuration-messages';

type ConfigSection = 'users' | 'groups' | 'roles';

const cards = [
  {
    id: 'users' as const,
    title: '用户',
    description: '维护本地账号和允许 AD/LDAP 登录的用户',
    icon: UserCog,
    color: '#38bdf8',
  },
  {
    id: 'groups' as const,
    title: '用户群组',
    description: '按团队聚合用户，并统一分配角色',
    icon: UsersRound,
    color: '#22c55e',
  },
  {
    id: 'roles' as const,
    title: '用户角色',
    description: '维护内置角色和自定义权限集合',
    icon: ShieldCheck,
    color: '#f59e0b',
  },
];

export function UserSettingsPanel({ canManage }: { canManage: boolean }) {
  const [active, setActive] = useState<ConfigSection>('users');
  const [users, setUsers] = useState<SettingsUser[]>([]);
  const [groups, setGroups] = useState<UserGroup[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [permissions, setPermissions] = useState<Permission[]>([]);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [userEditor, setUserEditor] = useState<SettingsUser | null | undefined>(undefined);
  const [groupEditor, setGroupEditor] = useState<UserGroup | null | undefined>(undefined);
  const [roleEditor, setRoleEditor] = useState<Role | null | undefined>(undefined);
  const loadPromiseRef = useRef<Promise<void> | null>(null);

  const load = useCallback(() => {
    if (loadPromiseRef.current) return loadPromiseRef.current;
    const promise = (async () => {
      setLoading(true);
      setLoadError('');
      try {
        const [userResult, groupResult, roleResult, permissionResult] = await Promise.all([
          fetchSettingsUsers(),
          fetchUserGroups(),
          fetchRoles(),
          fetchPermissions(),
        ]);
        setUsers(userResult.items);
        setGroups(groupResult.items);
        setRoles(roleResult.items);
        setPermissions(permissionResult.items);
      } catch (err) {
        const message = err instanceof Error ? err.message : '读取用户配置失败';
        showErrorToast(err, '读取用户配置失败');
        setLoadError(message.includes('当前用户无权执行此操作') ? '' : message);
      } finally {
        setLoading(false);
      }
    })().finally(() => {
      loadPromiseRef.current = null;
    });
    loadPromiseRef.current = promise;
    return promise;
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const keyword = search.trim().toLowerCase();
  const filteredUsers = useMemo(
    () =>
      users.filter(item =>
        `${item.username} ${item.displayName} ${item.email} ${(item.roles ?? []).map(role => `${role.name} ${role.key}`).join(' ')}`
          .toLowerCase()
          .includes(keyword)
      ),
    [keyword, users]
  );
  const filteredGroups = useMemo(
    () =>
      groups.filter(item =>
        `${item.name} ${item.description} ${(item.members ?? []).map(member => `${member.username} ${member.displayName}`).join(' ')} ${(item.roles ?? []).map(role => `${role.name} ${role.key}`).join(' ')}`
          .toLowerCase()
          .includes(keyword)
      ),
    [groups, keyword]
  );
  const filteredRoles = useMemo(
    () =>
      roles.filter(item =>
        `${item.key} ${item.name} ${item.description}`.toLowerCase().includes(keyword)
      ),
    [keyword, roles]
  );

  async function deleteUser(user: SettingsUser) {
    try {
      await deleteSettingsUser(user.id);
      toast.success(deletedUserConfigurationMessage('用户名', user.username));
      setUserEditor(undefined);
      await load();
    } catch (err) {
      showErrorToast(err, '删除用户失败');
    }
  }

  async function deleteGroup(group: UserGroup) {
    try {
      await deleteUserGroup(group.id);
      toast.success(deletedUserConfigurationMessage('用户群组', group.name));
      setGroupEditor(undefined);
      await load();
    } catch (err) {
      showErrorToast(err, '删除用户群组失败');
    }
  }

  async function deleteUserRole(role: Role) {
    try {
      await deleteRole(role.id);
      toast.success(deletedUserConfigurationMessage('用户角色', role.name));
      setRoleEditor(undefined);
      await load();
    } catch (err) {
      showErrorToast(err, '删除用户角色失败');
    }
  }

  async function toggleUserDisabled(user: SettingsUser) {
    try {
      const saved = await updateSettingsUserDisabled(user.id, !user.disabled);
      setUsers(current => current.map(item => (item.id === saved.id ? saved : item)));
      setUserEditor(saved);
      toast.success(statusUserConfigurationMessage('用户名', saved.username, saved.disabled));
    } catch (err) {
      showErrorToast(err, '更新用户状态失败');
    }
  }

  async function toggleGroupDisabled(group: UserGroup) {
    try {
      const saved = await updateUserGroup(group.id, {
        name: group.name,
        description: group.description,
        disabled: !group.disabled,
        memberIds: group.members.map(member => member.id),
        roleKeys: group.roles.map(role => role.key),
      });
      setGroups(current => current.map(item => (item.id === saved.id ? saved : item)));
      setGroupEditor(saved);
      toast.success(statusUserConfigurationMessage('用户群组', saved.name, saved.disabled));
    } catch (err) {
      showErrorToast(err, '更新用户群组状态失败');
    }
  }

  async function toggleRoleDisabled(role: Role) {
    try {
      const saved = await updateRoleDisabled(role.id, !role.disabled);
      setRoles(current => current.map(item => (item.id === saved.id ? saved : item)));
      setRoleEditor(saved);
      toast.success(statusUserConfigurationMessage('用户角色', saved.name, saved.disabled));
    } catch (err) {
      showErrorToast(err, '更新用户角色状态失败');
    }
  }

  const selectedCard = cards.find(item => item.id === active) ?? cards[0];
  const SelectedIcon = selectedCard.icon;
  const count =
    active === 'users' ? users.length : active === 'groups' ? groups.length : roles.length;
  const filteredCount =
    active === 'users'
      ? filteredUsers.length
      : active === 'groups'
        ? filteredGroups.length
        : filteredRoles.length;
  const emptyText =
    active === 'users' ? '暂无用户' : active === 'groups' ? '暂无用户群组' : '暂无用户角色';
  const filteredEmptyText =
    active === 'users'
      ? '没有匹配的用户'
      : active === 'groups'
        ? '没有匹配的群组'
        : '没有匹配的角色';
  const showRows = !loading && count > 0 && filteredCount > 0;

  return (
    <>
      <div className="grid min-h-0 flex-1 grid-cols-1 gap-4 overflow-hidden xl:grid-cols-[430px_minmax(0,1fr)]">
        <nav
          className="bm-hidden-scrollbar max-h-64 space-y-2 overflow-y-auto rounded-lg border border-[var(--bm-border)] bg-white/[0.035] p-2 xl:max-h-none"
          aria-label="用户配置"
        >
          {cards.map(card => {
            const Icon = card.icon;
            const itemCount =
              card.id === 'users'
                ? users.length
                : card.id === 'groups'
                  ? groups.length
                  : roles.length;
            return (
              <button
                key={card.id}
                type="button"
                onClick={() => {
                  setActive(card.id);
                  setSearch('');
                }}
                className="bm-config-side-card flex w-full items-start gap-3 rounded-lg p-3 text-left transition-all duration-200"
                data-active={active === card.id}
              >
                <span
                  className="grid h-10 w-10 shrink-0 place-items-center rounded-lg border border-white/[0.08] bg-white/[0.05]"
                  style={{ color: card.color }}
                >
                  <Icon size={19} />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="flex items-center justify-between gap-3">
                    <span className="text-sm font-semibold">{card.title}</span>
                    <span className="text-xs text-[var(--bm-text-muted)]">{itemCount}</span>
                  </span>
                  <span className="mt-1 block line-clamp-2 text-xs leading-5 text-[var(--bm-text-muted)]">
                    {card.description}
                  </span>
                </span>
              </button>
            );
          })}
        </nav>

        <aside className="flex min-h-0 flex-col overflow-hidden rounded-lg border border-[var(--bm-border)] bg-white/[0.035] p-4">
          <div className="mb-4 flex items-center gap-3">
            <span
              className="grid h-10 w-10 place-items-center rounded-lg border border-white/[0.08] bg-white/[0.05]"
              style={{ color: selectedCard.color }}
            >
              <SelectedIcon size={19} />
            </span>
            <div className="min-w-0">
              <h2 className="text-sm font-semibold">{selectedCard.title}</h2>
              <p className="mt-0.5 truncate text-xs text-[var(--bm-text-muted)]">
                {selectedCard.description}。
              </p>
            </div>
          </div>
          {loadError ? (
            <div
              className="mb-4 rounded-lg border p-3 text-sm"
              style={{
                background: 'rgba(245,158,11,0.1)',
                borderColor: 'rgba(245,158,11,0.25)',
                color: '#f59e0b',
              }}
            >
              {loadError}
            </div>
          ) : null}
          <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-[var(--bm-border)] bg-white/[0.02]">
            <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--bm-border)] p-3">
              <div className="flex items-baseline gap-2">
                <span className="text-sm font-semibold">{selectedCard.title}</span>
                <span className="text-xs text-[var(--bm-text-muted)]">{count} 项</span>
              </div>
              <div className="flex min-w-0 flex-1 items-center justify-end gap-2">
                <div className="relative min-w-[220px] max-w-sm flex-1">
                  <Search
                    size={14}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--bm-text-muted)]"
                  />
                  <input
                    value={search}
                    onChange={event => setSearch(event.target.value)}
                    placeholder={
                      active === 'users'
                        ? '搜索用户名、显示名或角色'
                        : active === 'groups'
                          ? '搜索群组、描述、成员或角色'
                          : '搜索角色名称、标识或描述'
                    }
                    className="h-9 w-full rounded-lg pl-9 pr-3 text-sm outline-none"
                    style={{
                      background: 'var(--bm-control-bg)',
                      border: '1px solid var(--bm-border)',
                      color: 'var(--bm-text)',
                    }}
                  />
                </div>
                {canManage ? (
                  <button
                    type="button"
                    className="bm-action-button flex h-9 shrink-0 items-center gap-2 rounded-lg border px-3 text-sm"
                    onClick={() =>
                      active === 'users'
                        ? setUserEditor(null)
                        : active === 'groups'
                          ? setGroupEditor(null)
                          : setRoleEditor(null)
                    }
                    style={{
                      borderColor: 'rgba(59,130,246,0.38)',
                      color: 'var(--bm-accent-text)',
                      background: 'rgba(59,130,246,0.1)',
                    }}
                  >
                    <Plus size={14} />
                    新增{selectedCard.title === '用户角色' ? '角色' : selectedCard.title}
                  </button>
                ) : null}
              </div>
            </div>
            <div className="bm-hidden-scrollbar min-h-0 flex-1 space-y-2 overflow-y-auto p-3">
              {loading ? (
                <EmptyState text="正在读取用户配置" />
              ) : count === 0 ? (
                <EmptyState text={emptyText} />
              ) : filteredCount === 0 ? (
                <EmptyState text={filteredEmptyText} />
              ) : null}
              {showRows && active === 'users'
                ? filteredUsers.map(user => (
                    <CompactRow
                      key={user.id}
                      title={user.displayName || user.username}
                      meta={`${user.username} · ${(user.directRoles ?? []).map(role => role.name).join('、') || user.role}`}
                      extra={`创建时间：${formatDateTime(user.createdAt)} · 最近登录：${formatLastLogin(user.lastLoginAt)}`}
                      badge={user.disabled ? '禁用' : '启用'}
                      enabled={!user.disabled}
                      active={userEditor?.id === user.id}
                      onClick={() => setUserEditor(user)}
                    />
                  ))
                : null}
              {showRows && active === 'groups'
                ? filteredGroups.map(group => (
                    <CompactRow
                      key={group.id}
                      title={group.name}
                      meta={`${(group.members ?? []).length} 个用户 · ${(group.roles ?? []).map(role => role.name).join('、') || '无角色'}`}
                      extra={group.description || '未填写描述'}
                      badge={`${(group.roles ?? []).length} 角色`}
                      enabled={!group.disabled}
                      active={groupEditor?.id === group.id}
                      onClick={() => setGroupEditor(group)}
                    />
                  ))
                : null}
              {showRows && active === 'roles'
                ? filteredRoles.map(role => (
                    <CompactRow
                      key={role.id}
                      title={role.name}
                      meta={`${role.key} · ${(role.permissions ?? []).length} 项权限`}
                      extra={role.description || '未填写描述'}
                      badge={role.builtin ? '内置' : role.disabled ? '禁用' : '自定义'}
                      enabled={!role.disabled}
                      active={roleEditor?.id === role.id}
                      onClick={() => setRoleEditor(role)}
                    />
                  ))
                : null}
            </div>
          </div>
        </aside>
      </div>

      {userEditor !== undefined ? (
        <UserEditorDialog
          open
          user={userEditor}
          roles={roles}
          canManage={canManage}
          onOpenChange={open => !open && setUserEditor(undefined)}
          onSaved={async () => {
            setUserEditor(undefined);
            await load();
          }}
          onToggleDisabled={userEditor ? () => toggleUserDisabled(userEditor) : undefined}
          onDelete={
            userEditor && userEditor.username !== 'admin'
              ? () => void deleteUser(userEditor)
              : undefined
          }
        />
      ) : null}
      {groupEditor !== undefined ? (
        <GroupEditorDialog
          open
          group={groupEditor}
          users={users}
          roles={roles}
          canManage={canManage}
          onOpenChange={open => !open && setGroupEditor(undefined)}
          onSaved={async () => {
            setGroupEditor(undefined);
            await load();
          }}
          onToggleDisabled={groupEditor ? () => toggleGroupDisabled(groupEditor) : undefined}
          onDelete={groupEditor ? () => void deleteGroup(groupEditor) : undefined}
        />
      ) : null}
      {roleEditor !== undefined ? (
        <RoleEditorDialog
          open
          role={roleEditor}
          permissions={permissions}
          canManage={canManage}
          onOpenChange={open => !open && setRoleEditor(undefined)}
          onSaved={async () => {
            setRoleEditor(undefined);
            await load();
          }}
          onToggleDisabled={roleEditor ? () => toggleRoleDisabled(roleEditor) : undefined}
          onDelete={
            roleEditor && !roleEditor.builtin ? () => void deleteUserRole(roleEditor) : undefined
          }
        />
      ) : null}
    </>
  );
}

function CompactRow({
  title,
  meta,
  extra,
  badge,
  enabled,
  active,
  onClick,
}: {
  title: string;
  meta: string;
  extra: string;
  badge: string;
  enabled?: boolean;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="bm-action-button group grid min-h-[74px] w-full grid-cols-[3px_minmax(0,1fr)_auto] items-center gap-3 rounded-lg border p-0 pr-3 text-left"
      style={{
        background: active ? 'rgba(59,130,246,0.11)' : 'rgba(255,255,255,0.026)',
        borderColor: active ? 'rgba(96,165,250,0.52)' : 'var(--bm-border)',
        color: 'var(--bm-text)',
      }}
    >
      <span
        className="h-full rounded-l-lg"
        style={{ background: active ? '#60a5fa' : 'transparent' }}
      />
      <span className="min-w-0 py-2">
        <span className="flex min-w-0 items-center gap-2">
          <span className="truncate text-sm font-semibold">{title}</span>
          <span
            className="shrink-0 rounded-md border px-1.5 py-0.5 text-[11px]"
            style={{
              color: 'var(--bm-text-muted)',
              borderColor: 'var(--bm-border)',
              background: 'rgba(255,255,255,0.028)',
            }}
          >
            {badge}
          </span>
        </span>
        <span className="mt-1 block truncate text-xs" style={{ color: 'var(--bm-text-muted)' }}>
          {meta}
        </span>
        <span className="mt-0.5 block truncate text-xs" style={{ color: 'var(--bm-text-muted)' }}>
          {extra}
        </span>
      </span>
      {typeof enabled === 'boolean' ? (
        enabled ? (
          <ToggleRight size={18} style={{ color: '#86efac' }} />
        ) : (
          <ToggleLeft size={18} style={{ color: 'var(--bm-text-muted)' }} />
        )
      ) : (
        <span
          className="h-2 w-2 rounded-full"
          style={{ background: active ? '#60a5fa' : 'var(--bm-border)' }}
        />
      )}
    </button>
  );
}

function EmptyState({ text }: { text: string }) {
  return (
    <div
      className="flex min-h-48 items-center justify-center rounded-lg border text-sm"
      style={{
        borderColor: 'var(--bm-border)',
        color: 'var(--bm-text-muted)',
        background: 'rgba(255,255,255,0.02)',
      }}
    >
      {text}
    </div>
  );
}

function formatDateTime(value?: string) {
  if (!value) return '-';
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? '-' : date.toLocaleString();
}

function formatLastLogin(value?: string) {
  return value ? formatDateTime(value) : '从未登录';
}
