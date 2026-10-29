import { Check, Eye, EyeOff } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';
import { showErrorToast } from '@/lib/toast-errors';
import { AppTooltip } from '@/components/feedback/app-tooltip';
import { focusTextAreaAtEnd } from '../settings-dialog-focus';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  createSettingsUser,
  createUserGroup,
  updateSettingsUser,
  updateUserGroup,
} from '@/features/settings/api';
import type { Role, SettingsUser, UserGroup } from '../types';
import { DangerButton, PrimaryButton, StateButton } from './UserSettingsDialogActions';
import {
  createdUserConfigurationMessage,
  updatedUserConfigurationMessage,
  userConfigurationSaveError,
} from '../user-configuration-messages';

const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

type CommonProps = {
  open: boolean;
  canManage: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: () => void | Promise<void>;
  onDelete?: () => void;
  onToggleDisabled?: () => void | Promise<void>;
};
export function UserEditorDialog({
  open,
  user,
  roles,
  canManage,
  onOpenChange,
  onSaved,
  onDelete,
  onToggleDisabled,
}: CommonProps & { user: SettingsUser | null; roles: Role[] }) {
  const [username, setUsername] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [roleKeys, setRoleKeys] = useState<string[]>(['viewer']);
  const [disabled, setDisabled] = useState(false);
  const [busy, setBusy] = useState(false);
  const [toggling, setToggling] = useState(false);
  const usernameInputRef = useRef<HTMLInputElement>(null);
  const emailInputRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (!open) return;
    setUsername(user?.username ?? '');
    setDisplayName(user?.displayName ?? '');
    setEmail(user?.email ?? '');
    setPassword('');
    setRoleKeys(
      (user?.directRoles ?? []).length
        ? (user?.directRoles ?? []).map(role => role.key)
        : [user?.role || 'viewer']
    );
    setDisabled(user?.disabled ?? false);
  }, [open, user]);
  async function save() {
    if (!username.trim()) return toast.error('用户名不能为空');
    if (!emailPattern.test(email.trim())) return toast.error('请输入有效的邮箱地址');
    if (!user && password.length < 6) return toast.error('密码至少 6 个字符');
    if (user && password && password.length < 6) return toast.error('密码至少 6 个字符');
    if (roleKeys.length === 0) return toast.error('请选择角色');
    setBusy(true);
    try {
      const body = {
        username: username.trim(),
        displayName: displayName.trim() || username.trim(),
        email: email.trim(),
        password,
        roleKeys,
        disabled,
      };
      if (user) await updateSettingsUser(user.id, body);
      else await createSettingsUser(body);
      toast.success(
        user
          ? updatedUserConfigurationMessage('用户名', username.trim())
          : createdUserConfigurationMessage('用户名', username.trim())
      );
      await onSaved();
    } catch (err) {
      showErrorToast(
        userConfigurationSaveError(err, '用户保存失败', '用户名已存在', '用户名', username.trim())
      );
    } finally {
      setBusy(false);
    }
  }
  async function toggleDisabled() {
    if (!onToggleDisabled) return;
    setToggling(true);
    try {
      await onToggleDisabled();
    } finally {
      setToggling(false);
    }
  }
  const lockedAdmin = user?.username === 'admin';
  const usernameLocked = Boolean(user);
  return (
    <EditorFrame
      open={open}
      title={user ? '编辑用户' : '新增用户'}
      subtitle={user?.username || '配置平台账号和登录权限'}
      onOpenChange={onOpenChange}
      onOpenAutoFocus={event => {
        event.preventDefault();
        window.requestAnimationFrame(() => {
          focusInputAtEnd(usernameLocked ? emailInputRef.current : usernameInputRef.current);
        });
      }}
      footer={
        canManage ? (
          <>
            {user && !lockedAdmin && onDelete ? (
              <DangerButton busy={busy} disabled={!user.disabled} onClick={onDelete} label="删除" />
            ) : null}
            <StateButton
              disabled={!user || lockedAdmin || busy || toggling || !onToggleDisabled}
              active={Boolean(user?.disabled)}
              onClick={() => void toggleDisabled()}
            />
            <PrimaryButton busy={busy} onClick={() => void save()} label="保存" />
          </>
        ) : null
      }
    >
      <SectionTitle title="基础信息" />
      <InlineRow label="用户名" required>
        <TextInput
          value={username}
          placeholder="用户名"
          disabled={!canManage || usernameLocked}
          inputRef={usernameInputRef}
          onChange={setUsername}
        />
      </InlineRow>
      <InlineRow label="邮箱" required>
        <TextInput
          value={email}
          type="email"
          placeholder="用于验证找回密码身份"
          disabled={!canManage}
          inputRef={emailInputRef}
          onChange={setEmail}
        />
      </InlineRow>
      <InlineRow label="显示名称">
        <TextInput
          value={displayName}
          placeholder="用户显示名称"
          disabled={!canManage}
          onChange={setDisplayName}
        />
      </InlineRow>
      <InlineRow label={user ? '新密码' : '密码'} required={!user}>
        <PasswordInput
          value={password}
          placeholder={user ? '留空则不修改' : '至少 6 个字符'}
          disabled={!canManage}
          onChange={setPassword}
        />
      </InlineRow>
      <SectionTitle title="授权与状态" />
      <SelectionList
        title="角色"
        items={roles
          .filter(role => !role.disabled || roleKeys.includes(role.key))
          .map(role => ({ key: role.key, label: role.name, helper: role.description }))}
        value={roleKeys}
        disabled={!canManage || lockedAdmin}
        single
        hideActions
        onChange={setRoleKeys}
      />
    </EditorFrame>
  );
}
export function GroupEditorDialog({
  open,
  group,
  users,
  roles,
  canManage,
  onOpenChange,
  onSaved,
  onDelete,
  onToggleDisabled,
}: CommonProps & { group: UserGroup | null; users: SettingsUser[]; roles: Role[] }) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [disabled, setDisabled] = useState(false);
  const [memberIds, setMemberIds] = useState<string[]>([]);
  const [roleKeys, setRoleKeys] = useState<string[]>(['viewer']);
  const [busy, setBusy] = useState(false);
  const [toggling, setToggling] = useState(false);
  const nameInputRef = useRef<HTMLInputElement>(null);
  const descriptionInputRef = useRef<HTMLTextAreaElement>(null);
  useEffect(() => {
    if (!open) return;
    setName(group?.name ?? '');
    setDescription(group?.description ?? '');
    setDisabled(group?.disabled ?? false);
    setMemberIds((group?.members ?? []).map(member => member.id));
    setRoleKeys(
      (group?.roles ?? []).length ? (group?.roles ?? []).map(role => role.key) : ['viewer']
    );
  }, [group, open]);
  async function save() {
    if (!name.trim()) return toast.error('用户群组名称不能为空');
    setBusy(true);
    try {
      const body = {
        name: name.trim(),
        description: description.trim(),
        disabled,
        memberIds,
        roleKeys,
      };
      if (group) await updateUserGroup(group.id, body);
      else await createUserGroup(body);
      toast.success(
        group
          ? updatedUserConfigurationMessage('用户群组', name.trim())
          : createdUserConfigurationMessage('用户群组', name.trim())
      );
      await onSaved();
    } catch (err) {
      showErrorToast(
        userConfigurationSaveError(
          err,
          '用户群组保存失败',
          '用户群组名称已存在',
          '用户群组',
          name.trim()
        )
      );
    } finally {
      setBusy(false);
    }
  }
  async function toggleDisabled() {
    if (!onToggleDisabled) return;
    setToggling(true);
    try {
      await onToggleDisabled();
    } finally {
      setToggling(false);
    }
  }
  return (
    <EditorFrame
      open={open}
      title={group ? '编辑群组' : '新增群组'}
      subtitle={group?.name || '批量组织成员和角色'}
      onOpenChange={onOpenChange}
      onOpenAutoFocus={event => {
        event.preventDefault();
        window.requestAnimationFrame(() => {
          focusTextAreaAtEnd(group ? descriptionInputRef.current : null, nameInputRef.current);
        });
      }}
      footer={
        canManage ? (
          <>
            {group && onDelete ? (
              <DangerButton
                busy={busy}
                disabled={!group.disabled}
                onClick={onDelete}
                label="删除"
              />
            ) : null}
            <StateButton
              disabled={!group || busy || toggling || !onToggleDisabled}
              active={Boolean(group?.disabled)}
              onClick={() => void toggleDisabled()}
            />
            <PrimaryButton busy={busy} onClick={() => void save()} label="保存" />
          </>
        ) : null
      }
    >
      <SectionTitle title="基础信息" />
      <InlineRow label="群组名称" required>
        <TextInput
          value={name}
          placeholder="运维组"
          disabled={!canManage || Boolean(group)}
          inputRef={nameInputRef}
          onChange={setName}
        />
      </InlineRow>
      <InlineRow label="描述" alignStart>
        <TextArea
          value={description}
          placeholder="群组用途"
          disabled={!canManage}
          textareaRef={descriptionInputRef}
          onChange={setDescription}
        />
      </InlineRow>
      <SectionTitle title="成员与角色" />
      <SelectionList
        title="群组成员"
        items={users.map(user => ({
          key: user.id,
          label: user.displayName || user.username,
          helper: user.username,
        }))}
        value={memberIds}
        disabled={!canManage}
        onChange={setMemberIds}
      />
      <SelectionList
        title="群组角色"
        items={roles
          .filter(role => !role.disabled || roleKeys.includes(role.key))
          .map(role => ({ key: role.key, label: role.name, helper: role.description }))}
        value={roleKeys}
        disabled={!canManage}
        single
        hideActions
        onChange={setRoleKeys}
      />
    </EditorFrame>
  );
}
export function EditorFrame({
  open,
  title,
  subtitle,
  onOpenChange,
  onOpenAutoFocus,
  footer,
  children,
}: {
  open: boolean;
  title: string;
  subtitle: string;
  onOpenChange: (open: boolean) => void;
  onOpenAutoFocus?: React.ComponentProps<typeof DialogContent>['onOpenAutoFocus'];
  footer: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="bm-dialog-panel flex max-h-[88vh] w-[min(92vw,720px)] flex-col gap-0 overflow-hidden p-0 sm:max-w-none"
        onOpenAutoFocus={onOpenAutoFocus}
      >
        <DialogHeader className="space-y-0 border-b border-[var(--bm-border)] p-5 pr-14 text-left">
          <DialogTitle className="text-base tracking-normal">{title}</DialogTitle>
          <DialogDescription className="mt-1 text-xs">{subtitle}</DialogDescription>
        </DialogHeader>
        <div className="bm-hidden-scrollbar min-h-0 flex-1 space-y-3 overflow-y-auto p-5">
          {children}
        </div>
        {footer ? (
          <DialogFooter className="border-t border-[var(--bm-border)] bg-white/[0.018] p-4 sm:justify-end">
            {footer}
          </DialogFooter>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

export function SectionTitle({ title }: { title: string }) {
  return <div className="pt-1 text-sm font-semibold text-[var(--bm-text)]">{title}</div>;
}

export function InlineRow({
  label,
  required = false,
  alignStart = false,
  children,
}: {
  label: string;
  required?: boolean;
  alignStart?: boolean;
  children: React.ReactNode;
}) {
  return (
    <label
      className={`grid grid-cols-1 gap-1.5 text-xs text-[var(--bm-text-muted)] sm:grid-cols-[64px_minmax(0,1fr)] sm:gap-3 ${alignStart ? 'sm:items-start' : 'sm:items-center'}`}
    >
      <span className={alignStart ? 'sm:pt-2 sm:text-right' : 'sm:text-right'}>
        {label}
        {required ? <span className="text-red-400">*</span> : null}
      </span>
      <span className="min-w-0">{children}</span>
    </label>
  );
}

export function TextInput({
  value,
  onChange,
  placeholder,
  type = 'text',
  disabled = false,
  inputRef,
}: {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  type?: string;
  disabled?: boolean;
  inputRef?: React.Ref<HTMLInputElement>;
}) {
  return (
    <input
      type={type}
      ref={inputRef}
      value={value}
      disabled={disabled}
      onChange={event => onChange(event.target.value)}
      placeholder={placeholder}
      className="w-full rounded-lg px-3 py-2 text-sm outline-none disabled:opacity-60"
      style={{
        background: 'var(--bm-control-bg)',
        border: '1px solid var(--bm-border)',
        color: 'var(--bm-text)',
      }}
    />
  );
}

function focusInputAtEnd(input: HTMLInputElement | null) {
  if (!input || input.disabled) return;
  input.focus();
  try {
    input.setSelectionRange(input.value.length, input.value.length);
  } catch {
    return;
  }
}

function PasswordInput(props: Omit<React.ComponentProps<typeof TextInput>, 'type'>) {
  const [visible, setVisible] = useState(false);
  return (
    <span className="relative block">
      <TextInput {...props} type={visible ? 'text' : 'password'} />
      <AppTooltip label={visible ? '隐藏密码' : '显示密码'} placement="top">
        <button
          type="button"
          disabled={props.disabled}
          onClick={() => setVisible(value => !value)}
          className="bm-action-button absolute right-2 top-1/2 grid h-7 w-7 -translate-y-1/2 place-items-center rounded-md disabled:cursor-not-allowed disabled:opacity-50"
          aria-label={visible ? '隐藏密码' : '显示密码'}
        >
          {visible ? <EyeOff size={15} /> : <Eye size={15} />}
        </button>
      </AppTooltip>
    </span>
  );
}

export function TextArea({
  value,
  onChange,
  placeholder,
  disabled,
  textareaRef,
}: {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  disabled: boolean;
  textareaRef?: React.Ref<HTMLTextAreaElement>;
}) {
  return (
    <textarea
      rows={3}
      ref={textareaRef}
      value={value}
      disabled={disabled}
      onChange={event => onChange(event.target.value)}
      placeholder={placeholder}
      className="bm-hidden-scrollbar w-full resize-y rounded-lg px-3 py-2 text-sm outline-none disabled:opacity-60"
      style={{
        background: 'var(--bm-control-bg)',
        border: '1px solid var(--bm-border)',
        color: 'var(--bm-text)',
      }}
    />
  );
}

export function SelectionList({
  title,
  items,
  value,
  disabled,
  single = false,
  hideActions = false,
  onChange,
  onToggle,
}: {
  title: string;
  items: Array<{ key: string; label: string; helper: string }>;
  value: string[];
  disabled: boolean;
  single?: boolean;
  hideActions?: boolean;
  onChange: (value: string[]) => void;
  onToggle?: (key: string) => void;
}) {
  const itemKeys = items.map(item => item.key);
  const allSelected = itemKeys.length > 0 && itemKeys.every(key => value.includes(key));

  function change(key: string) {
    if (onToggle) return onToggle(key);
    onChange(single ? [key] : toggle(value, key));
  }
  return (
    <section>
      <div className="mb-2 flex items-center justify-between gap-3">
        <h4 className="text-sm font-semibold text-[var(--bm-text)]">{title}</h4>
        {!hideActions ? (
          <div className="flex gap-1">
            <button
              type="button"
              disabled={disabled || itemKeys.length === 0 || allSelected}
              className="bm-action-button cursor-pointer rounded-md border px-2.5 py-1 text-[11px] transition disabled:cursor-not-allowed disabled:opacity-50"
              style={{
                borderColor: 'rgba(59,130,246,0.32)',
                color: 'var(--bm-accent-text)',
                background: 'rgba(59,130,246,0.08)',
              }}
              onClick={() => onChange([...new Set([...value, ...itemKeys])])}
            >
              全选
            </button>
            <button
              type="button"
              disabled={
                disabled || itemKeys.length === 0 || !itemKeys.some(key => value.includes(key))
              }
              className="bm-action-button cursor-pointer rounded-md border px-2.5 py-1 text-[11px] transition disabled:cursor-not-allowed disabled:opacity-50"
              style={{
                borderColor: 'var(--bm-border)',
                color: 'var(--bm-text-muted)',
                background: 'rgba(255,255,255,0.04)',
              }}
              onClick={() => onChange(value.filter(key => !itemKeys.includes(key)))}
            >
              清空
            </button>
          </div>
        ) : null}
      </div>
      <div className="bm-hidden-scrollbar grid gap-1.5 pb-1 pr-1">
        {items.map(item => {
          const active = value.includes(item.key);
          return (
            <label
              key={item.key}
              className={`flex items-start gap-2 rounded-lg border border-transparent p-2 text-sm ${disabled ? 'cursor-not-allowed opacity-70' : 'cursor-pointer'}`}
              style={{
                background: active ? 'rgba(59,130,246,0.11)' : 'rgba(255,255,255,0.026)',
                boxShadow: `inset 0 0 0 1px ${active ? 'rgba(96,165,250,0.45)' : 'var(--bm-border)'}`,
              }}
            >
              <input
                type={single ? 'radio' : 'checkbox'}
                name={single ? title : undefined}
                className="mt-1"
                checked={active}
                disabled={disabled}
                onChange={() => change(item.key)}
              />
              <span className="min-w-0">
                <span className="block truncate text-sm font-medium text-[var(--bm-text)]">
                  {item.label}
                </span>
                <span className="mt-0.5 block text-xs leading-4 text-[var(--bm-text-muted)]">
                  {item.helper}
                </span>
              </span>
              {active ? <Check className="ml-auto shrink-0 text-blue-300" size={15} /> : null}
            </label>
          );
        })}
      </div>
    </section>
  );
}

function toggle(items: string[], key: string) {
  return items.includes(key) ? items.filter(item => item !== key) : [...items, key];
}
