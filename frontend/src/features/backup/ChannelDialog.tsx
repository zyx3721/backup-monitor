import { AlertTriangle, Check, ChevronDown, Clock, Plus, WifiOff, X } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/feedback/app-tooltip';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { CodeEditor } from './CodeEditor';
import { BackupSelect, alertTypeLabel } from './components';
import type { ChannelFormState, MessageType } from './notify-config';
import {
  bodyTemplateFor,
  defaultMessageTypes,
  eventLevelHints,
  eventOptions,
  levelLabels,
  messageTypeOptions,
  parseRecipients,
  preserveTarget,
  receiveIdTypeOptions,
  typeLabels,
  variableChips,
} from './notify-config';

const eventMeta = {
  backup_failed: { icon: AlertTriangle, tone: 'bm-event-tone-danger' },
  not_run: { icon: Clock, tone: 'bm-event-tone-warning' },
  report_timeout: { icon: WifiOff, tone: 'bm-event-tone-info' },
} as const;

export function ChannelDialog({
  form,
  busy,
  title,
  onChange,
  onClose,
  onSubmit,
}: {
  form: ChannelFormState;
  busy: boolean;
  title?: string;
  onChange: (form: ChannelFormState) => void;
  onClose: () => void;
  onSubmit: () => void;
}) {
  const set = (patch: Partial<ChannelFormState>) => onChange({ ...form, ...patch });
  const setConfig = (key: string, value: string) =>
    onChange({ ...form, config: { ...form.config, [key]: value } });
  const recipients = form.type === 'mail' ? parseRecipients(form.config.recipients) : null;
  const validate = () => {
    if (!form.name.trim()) return '通知名称不能为空';
    if (form.eventType.length === 0) return '通知事件不能为空';
    if (form.type === 'mail') {
      if (!form.config.recipients.trim()) return '邮件列表不能为空';
      return recipients?.error || null;
    }
    if (form.type === 'webhook') {
      if (!form.config.method.trim()) return '请求方法不能为空';
      if (!form.config.url.trim()) return '请求地址不能为空';
      return null;
    }
    if (form.type === 'wecom') {
      if (!form.config.corpId.trim()) return '企业 ID 不能为空';
      if (!form.config.appSecret.trim()) return '凭证密钥不能为空';
      return null;
    }
    if (!form.config.appKey.trim()) return '应用 key 不能为空';
    if (!form.config.appSecret.trim()) return '应用密钥不能为空';
    if (form.type === 'feishu' && !form.config.receiveIdType.trim()) return 'ID 类型不能为空';
    return null;
  };
  const submit = () => {
    const error = validate();
    if (error) {
      toast.error(error);
      return;
    }
    onSubmit();
  };
  const changeType = (value: ChannelFormState['type']) => {
    const messageType = defaultMessageTypes[value] ?? 'text';
    onChange({
      ...form,
      type: value,
      config: { ...form.config, messageType, body: bodyTemplateFor(value, messageType) },
    });
  };
  const changeMessageType = (value: string) => {
    const nextTemplate = bodyTemplateFor(form.type, value as MessageType);
    onChange({
      ...form,
      config: {
        ...form.config,
        messageType: value,
        body: preserveTarget(form.type, form.config.body, nextTemplate),
      },
    });
  };
  const messageTypes = messageTypeOptions[form.type] ?? [];
  const messageType = (form.config.messageType || 'text') as MessageType;
  const messageTypeRow =
    messageTypes.length > 0 ? (
      <FormRow label="消息类型" required>
        <BackupSelect
          value={messageType}
          onValueChange={changeMessageType}
          className="h-10 w-full"
          ariaLabel="消息类型"
          options={messageTypes}
        />
      </FormRow>
    ) : null;

  return (
    <Dialog open onOpenChange={open => !open && onClose()}>
      <DialogContent className="flex max-h-[92vh] w-[92vw] max-w-4xl flex-col gap-0 overflow-hidden p-0">
        <div className="flex items-center gap-2 border-b border-[var(--bm-border)] px-6 py-4 pr-14">
          <Plus size={18} className="text-[var(--bm-accent-text)]" />
          <DialogTitle className="text-base font-semibold">
            {title ?? (form.id ? '编辑通知' : '添加通知')}
          </DialogTitle>
        </div>
        <div className="flex-1 space-y-4 overflow-y-auto px-6 py-5">
          <FormRow label="通知名称" required>
            <Input
              value={form.name}
              onChange={event => set({ name: event.target.value })}
              placeholder="如：运维邮件组 / 备份告警群"
            />
          </FormRow>

          <FormRow label="通知方式" required>
            <BackupSelect
              value={form.type}
              onValueChange={value => changeType(value as ChannelFormState['type'])}
              className="h-10 w-full"
              ariaLabel="通知方式"
              disabled={Boolean(form.id)}
              options={Object.entries(typeLabels).map(([value, label]) => ({ value, label }))}
            />
          </FormRow>

          <FormRow
            label="通知事件"
            required
            hint="可多选：该通知接收所选事件的告警（如勾选「备份失败」则不会收到主机离线通知）"
          >
            <EventMultiSelect
              value={form.eventType}
              onChange={value => set({ eventType: value })}
            />
          </FormRow>

          <FormRow label="告警级别">
            <BackupSelect
              value={form.level}
              onValueChange={value => set({ level: value as ChannelFormState['level'] })}
              className="h-10 w-full"
              ariaLabel="告警级别"
              options={Object.entries(levelLabels).map(([value, label]) => ({ value, label }))}
            />
          </FormRow>

          <FormRow label="恢复通知">
            <div className="flex items-center gap-3 pt-1.5">
              <Switch
                checked={form.recover}
                onChange={value => set({ recover: value })}
                label="恢复通知"
              />
              <span className="text-xs leading-5 text-[var(--bm-text-muted)]">
                开启后，告警恢复（状态变为「已恢复」）时也会推送一条通知
              </span>
            </div>
          </FormRow>

          {form.type === 'mail' ? (
            <FormRow
              label="邮件列表"
              required
              hint="提示：发件邮箱在【系统配置 → 找密设置 → 邮件媒介】中配置"
            >
              <CodeEditor
                value={form.config.recipients}
                onChange={value => setConfig('recipients', value)}
                rows={6}
                ariaLabel="邮件列表"
              />
            </FormRow>
          ) : null}

          {form.type === 'webhook' ? (
            <>
              <FormRow label="请求方法" required>
                <BackupSelect
                  value={form.config.method}
                  onValueChange={value => setConfig('method', value)}
                  className="h-10 w-full"
                  ariaLabel="请求方法"
                  options={[
                    { value: 'POST', label: 'POST' },
                    { value: 'GET', label: 'GET' },
                  ]}
                />
              </FormRow>
              <FormRow label="请求地址" required>
                <Input
                  value={form.config.url}
                  onChange={event => setConfig('url', event.target.value)}
                  placeholder="https://your-endpoint/notify"
                />
              </FormRow>
              <FormRow label="请求头">
                <CodeEditor
                  value={form.config.headers}
                  onChange={value => setConfig('headers', value)}
                  rows={4}
                  ariaLabel="请求头"
                />
              </FormRow>
              <FormRow
                label="请求体"
                hint={
                  <>
                    JSON 模板，支持 <b>{'{{变量}}'}</b> 或 <b>{'${变量}'}</b>
                    ，发送时自动替换为实际告警信息
                  </>
                }
              >
                <CodeEditor
                  value={form.config.body}
                  onChange={value => setConfig('body', value)}
                  rows={6}
                  ariaLabel="请求体"
                />
              </FormRow>
            </>
          ) : null}

          {form.type === 'wecom' ? (
            <>
              <FormRow label="企业 ID" required>
                <Input
                  value={form.config.corpId}
                  onChange={event => setConfig('corpId', event.target.value)}
                  placeholder="corpid"
                />
              </FormRow>
              <FormRow label="凭证密钥" required>
                <Input
                  type="password"
                  value={form.config.appSecret}
                  onChange={event => setConfig('appSecret', event.target.value)}
                  placeholder="corpsecret"
                />
              </FormRow>
              {messageTypeRow}
              <FormRow label="请求体">
                <CodeEditor
                  value={form.config.body}
                  onChange={value => setConfig('body', value)}
                  rows={6}
                  ariaLabel="请求体"
                />
              </FormRow>
            </>
          ) : null}

          {form.type === 'dingtalk' ? (
            <>
              <FormRow label="应用 key" required>
                <Input
                  value={form.config.appKey}
                  onChange={event => setConfig('appKey', event.target.value)}
                  placeholder="AppKey"
                />
              </FormRow>
              <FormRow label="应用密钥" required>
                <Input
                  type="password"
                  value={form.config.appSecret}
                  onChange={event => setConfig('appSecret', event.target.value)}
                  placeholder="AppSecret"
                />
              </FormRow>
              {messageTypeRow}
              <FormRow label="请求体">
                <CodeEditor
                  value={form.config.body}
                  onChange={value => setConfig('body', value)}
                  rows={8}
                  ariaLabel="请求体"
                />
              </FormRow>
            </>
          ) : null}

          {form.type === 'feishu' ? (
            <>
              <FormRow label="应用 key" required>
                <Input
                  value={form.config.appKey}
                  onChange={event => setConfig('appKey', event.target.value)}
                  placeholder="App ID，如 cli_xxx"
                />
              </FormRow>
              <FormRow label="应用密钥" required>
                <Input
                  type="password"
                  value={form.config.appSecret}
                  onChange={event => setConfig('appSecret', event.target.value)}
                  placeholder="App Secret"
                />
              </FormRow>
              <FormRow label="接收 ID 类型" required>
                <BackupSelect
                  value={form.config.receiveIdType}
                  onValueChange={value => setConfig('receiveIdType', value)}
                  className="h-10 w-full"
                  ariaLabel="接收 ID 类型"
                  options={receiveIdTypeOptions.map(option => ({ ...option, hint: option.value }))}
                />
              </FormRow>
              {messageTypeRow}
              <FormRow
                label="请求体"
                hint="接收 ID 类型会作为 receive_id_type 查询参数；receive_id 填对应 ID，多个接收人可用英文或中文逗号分隔，后端逐个发送"
              >
                <CodeEditor
                  value={form.config.body}
                  onChange={value => setConfig('body', value)}
                  rows={8}
                  ariaLabel="请求体"
                />
              </FormRow>
            </>
          ) : null}

          {form.type !== 'mail' ? (
            <FormRow
              label="可用变量"
              hint={
                <>
                  在请求体中以 <b>{'{{变量名}}'}</b> 或 <b>{'${变量名}'}</b> 引用；{' '}
                  <b>{'{{告警内容}}'}</b> 为完整多行信息
                </>
              }
            >
              <div className="flex flex-wrap gap-2 pt-1">
                {variableChips.map(chip => (
                  <code key={chip} className="bm-var-chip">
                    {`{{${chip}}}`}
                  </code>
                ))}
              </div>
            </FormRow>
          ) : null}

          <FormRow label="备注">
            <Input
              value={form.remark}
              onChange={event => set({ remark: event.target.value })}
              placeholder="可选"
            />
          </FormRow>

          <FormRow label="启用">
            <div className="flex items-center gap-3 pt-1.5">
              <Switch
                checked={form.enabled}
                onChange={value => set({ enabled: value })}
                label="启用"
              />
            </div>
          </FormRow>
        </div>
        <div className="flex justify-end gap-2 border-t border-[var(--bm-border)] px-6 py-4">
          <Button variant="outline" onClick={onClose}>
            取消
          </Button>
          <Button className="bm-primary-soft-button" disabled={busy} onClick={submit}>
            {busy ? '保存中' : '保存'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function FormRow({
  label,
  required,
  hint,
  error,
  children,
}: {
  label: string;
  required?: boolean;
  hint?: ReactNode;
  error?: string;
  children: ReactNode;
}) {
  return (
    <div className="grid gap-2 sm:grid-cols-[104px_1fr] sm:items-start sm:gap-4">
      <div className="text-left text-sm leading-5 text-[var(--bm-text-muted)] sm:pt-2.5 sm:text-right">
        {required ? <span className="mr-1 text-red-400">*</span> : null}
        {label}
      </div>
      <div className="min-w-0">
        {children}
        {hint ? (
          <p className="mt-1.5 text-xs leading-5 text-[var(--bm-text-muted)]">{hint}</p>
        ) : null}
        {error ? <p className="mt-1.5 text-xs leading-5 text-red-300">{error}</p> : null}
      </div>
    </div>
  );
}

function Switch({
  checked,
  onChange,
  label,
}: {
  checked: boolean;
  onChange: (value: boolean) => void;
  label: string;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      onClick={() => onChange(!checked)}
      className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border transition-colors ${
        checked
          ? 'border-[var(--bm-accent)]/60 bg-[var(--bm-accent)]/70'
          : 'border-[var(--bm-border)] bg-[var(--bm-control-bg)]'
      }`}
    >
      <span
        className={`inline-block h-4 w-4 rounded-full bg-white shadow-sm transition-transform ${
          checked ? 'translate-x-[18px]' : 'translate-x-0.5'
        }`}
      />
    </button>
  );
}

function EventMultiSelect({
  value,
  onChange,
}: {
  value: string[];
  onChange: (value: string[]) => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (event: MouseEvent) => {
      if (ref.current && !ref.current.contains(event.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', onPointerDown);
    return () => document.removeEventListener('mousedown', onPointerDown);
  }, [open]);

  const toggle = (event: string) => {
    onChange(value.includes(event) ? value.filter(item => item !== event) : [...value, event]);
  };

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => setOpen(current => !current)}
        className="flex min-h-10 w-full cursor-pointer flex-wrap items-center gap-1.5 rounded-lg border border-[var(--bm-border)] bg-[var(--bm-control-bg)] px-3 py-1.5 text-left text-sm text-[var(--bm-text)]"
      >
        {value.length === 0 ? (
          <span className="text-[var(--bm-text-muted)]">请选择通知事件</span>
        ) : (
          value.map(event => {
            const tone =
              eventMeta[event as keyof typeof eventMeta]?.tone ?? 'bm-event-tone-neutral';
            return (
              <span
                key={event}
                className={`inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-xs ${tone}`}
                style={{
                  backgroundColor: 'rgba(var(--bm-event-rgb), 0.14)',
                  borderColor: 'rgba(var(--bm-event-rgb), 0.35)',
                  color: 'var(--bm-event-text)',
                }}
              >
                {alertTypeLabel(event as never)}
                <span
                  role="button"
                  tabIndex={-1}
                  aria-label={`移除${alertTypeLabel(event as never)}`}
                  className="cursor-pointer opacity-70 hover:opacity-100"
                  onClick={clickEvent => {
                    clickEvent.stopPropagation();
                    toggle(event);
                  }}
                >
                  <X size={12} />
                </span>
              </span>
            );
          })
        )}
        <ChevronDown
          size={16}
          className={`ml-auto shrink-0 text-[var(--bm-text-muted)] transition-transform ${
            open ? 'rotate-180' : ''
          }`}
        />
      </button>
      {open ? (
        <div
          role="listbox"
          className="absolute left-0 right-0 z-40 mt-1 max-h-56 space-y-0.5 overflow-y-auto rounded-lg border border-[var(--bm-popover-border)] bg-[var(--bm-popover-bg)] p-1.5 shadow-2xl"
        >
          {eventOptions.map(event => {
            const checked = value.includes(event);
            const meta = eventMeta[event];
            const Icon = meta.icon;
            return (
              <button
                key={event}
                type="button"
                role="option"
                aria-selected={checked}
                onClick={() => toggle(event)}
                className={`flex w-full cursor-pointer items-center gap-2.5 rounded-lg border px-2.5 py-2 text-left text-sm transition-colors ${meta.tone} ${
                  checked ? '' : 'border-transparent hover:bg-[rgba(var(--bm-event-rgb),0.08)]'
                }`}
                style={
                  checked
                    ? {
                        backgroundColor: 'rgba(var(--bm-event-rgb), 0.14)',
                        borderColor: 'rgba(var(--bm-event-rgb), 0.38)',
                      }
                    : undefined
                }
              >
                <AppTooltip
                  label={eventLevelHints[event]}
                  className="flex flex-1 items-center gap-2.5"
                  wrap
                >
                  <span
                    className="grid h-6 w-6 shrink-0 place-items-center rounded-md"
                    style={{
                      backgroundColor: 'rgba(var(--bm-event-rgb), 0.16)',
                      color: 'var(--bm-event-text)',
                    }}
                  >
                    <Icon size={14} />
                  </span>
                  <span style={checked ? { color: 'var(--bm-event-text)' } : undefined}>
                    {alertTypeLabel(event)}
                  </span>
                </AppTooltip>
                <span
                  className={`grid h-4 w-4 place-items-center rounded border ${
                    checked ? 'text-white' : 'border-[var(--bm-border)] text-transparent'
                  }`}
                  style={
                    checked
                      ? {
                          backgroundColor: 'rgb(var(--bm-event-rgb))',
                          borderColor: 'rgb(var(--bm-event-rgb))',
                        }
                      : undefined
                  }
                >
                  <Check size={12} />
                </span>
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
