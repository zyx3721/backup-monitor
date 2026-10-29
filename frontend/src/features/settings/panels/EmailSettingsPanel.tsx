import { Mail, Save, Send, Trash2 } from 'lucide-react';
import { useCallback, useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';
import { showErrorToast } from '@/lib/toast-errors';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { fetchEmailSetting, saveEmailSetting, testEmailSetting } from '@/features/settings/api';
import {
  ActionButton,
  ConfigField,
  EnableToggle,
  SettingsDetailHeader,
  SettingsDetailPanel,
  SettingsSplitLayout,
  UsageIndicator,
  type SettingsField,
} from '../settings-primitives';
import { cardStyle } from '../settings-style';

type EmailForm = Record<string, unknown>;

const defaultForm: EmailForm = {
  smtpHost: '',
  smtpPort: '',
  username: '',
  password: '',
  from: '',
  fromName: 'Backup Monitor',
  useTLS: false,
  startTLS: false,
  allowInsecureAuth: false,
};

const fields: SettingsField[] = [
  { key: 'smtpHost', label: 'SMTP 主机', placeholder: 'smtp.example.com', required: true },
  { key: 'smtpPort', label: 'SMTP 端口', placeholder: '465', required: true, inputMode: 'numeric' },
  { key: 'username', label: '用户名', placeholder: 'admin@example.com', required: true },
  { key: 'password', label: '密码', placeholder: 'SMTP 授权码', type: 'password', required: true },
  { key: 'from', label: '发件人', placeholder: 'admin@example.com', required: true },
  { key: 'fromName', label: '发件人名称', placeholder: 'Backup Monitor' },
  { key: 'useTLS', label: '启用 TLS/SSL', type: 'checkbox' },
  { key: 'startTLS', label: '启用 STARTTLS', type: 'checkbox' },
  {
    key: 'allowInsecureAuth',
    label: '允许明文认证',
    helper: '仅在 SMTP 服务明确要求明文认证时启用',
    type: 'checkbox',
  },
];

export function EmailSettingsPanel({ canManage }: { canManage: boolean }) {
  const [form, setForm] = useState<EmailForm>(defaultForm);
  const [enabled, setEnabled] = useState(false);
  const [savedEnabled, setSavedEnabled] = useState(false);
  const [busy, setBusy] = useState('');
  const [clearRequested, setClearRequested] = useState(false);
  const [testOpen, setTestOpen] = useState(false);
  const [testEmail, setTestEmail] = useState('');
  const loadPromiseRef = useRef<Promise<void> | null>(null);

  const load = useCallback(() => {
    if (loadPromiseRef.current) return loadPromiseRef.current;
    const promise = (async () => {
      try {
        const setting = await fetchEmailSetting();
        setEnabled(setting.passwordResetEnabled);
        setSavedEnabled(setting.passwordResetEnabled);
        setForm({ ...defaultForm, ...setting.config });
      } catch (err) {
        showErrorToast(err, '读取通知配置失败');
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

  function updateField(field: SettingsField, value: unknown) {
    setClearRequested(false);
    setForm(current => {
      const next = { ...current, [field.key]: value };
      if (field.key === 'useTLS' && value === true) {
        next.startTLS = false;
        next.allowInsecureAuth = false;
        next.smtpPort = '465';
      }
      if (field.key === 'startTLS' && value === true) {
        next.useTLS = false;
        next.allowInsecureAuth = false;
        next.smtpPort = '587';
      }
      if (field.key === 'allowInsecureAuth' && value === true) {
        next.useTLS = false;
        next.startTLS = false;
      }
      if (field.key === 'smtpPort') {
        const port = parsePort(value);
        if (port === 465) {
          next.useTLS = true;
          next.startTLS = false;
          next.allowInsecureAuth = false;
        } else if (port === 587) {
          next.useTLS = false;
          next.startTLS = true;
          next.allowInsecureAuth = false;
        }
      }
      return next;
    });
  }

  async function save() {
    if (enabled) {
      const error = validateEmailForm(form, Boolean(form.hasPassword));
      if (error) {
        showErrorToast(error);
        return;
      }
    }
    setBusy('save');
    try {
      const saved = await saveEmailSetting({
        passwordResetEnabled: enabled,
        clearConfig: clearRequested,
        config: prepareEmailConfig(form, enabled),
      });
      setEnabled(saved.passwordResetEnabled);
      setSavedEnabled(saved.passwordResetEnabled);
      setForm({ ...defaultForm, ...saved.config });
      setClearRequested(false);
      toast.success('邮件配置已保存');
    } catch (err) {
      showErrorToast(err, '邮件配置保存失败');
    } finally {
      setBusy('');
    }
  }

  async function sendTest() {
    if (!testEmail.trim()) {
      toast.error('请输入测试邮箱');
      return;
    }
    setBusy('test');
    try {
      await testEmailSetting(testEmail.trim());
      setTestOpen(false);
      toast.success('测试邮件已发送');
    } catch (err) {
      showErrorToast(err, '测试邮件发送失败');
    } finally {
      setBusy('');
    }
  }

  return (
    <SettingsSplitLayout
      sidebarLabel="通知媒介"
      sidebar={
        <button
          type="button"
          className="bm-config-side-card flex w-full items-start gap-3 rounded-lg p-3 text-left transition-all duration-200"
          data-active="true"
          style={cardStyle(true)}
        >
          <span className="grid h-10 w-10 shrink-0 place-items-center rounded-lg border border-white/[0.08] bg-white/[0.05] text-emerald-400">
            <Mail size={19} />
          </span>
          <span className="min-w-0 flex-1">
            <span className="flex items-center justify-between gap-3">
              <span className="text-sm font-semibold">邮件媒介</span>
              <UsageIndicator enabled={savedEnabled} />
            </span>
            <span className="mt-1 block text-xs leading-5 text-[var(--bm-text-muted)]">
              通过 SMTP 发送找回密码验证码。
            </span>
          </span>
        </button>
      }
    >
      <SettingsDetailPanel
        header={
          <SettingsDetailHeader
            icon={Mail}
            color="#10b981"
            title="邮件"
            subtitle={enabled ? '找回密码已启用' : '未启用'}
            active={enabled}
          />
        }
        actions={
          canManage ? (
            <>
              <ActionButton
                icon={<Trash2 size={14} />}
                label="清空配置"
                tone="danger"
                disabled={busy !== ''}
                onClick={() => {
                  setEnabled(false);
                  setForm(defaultForm);
                  setClearRequested(true);
                }}
              />
              <ActionButton
                icon={<Save size={14} />}
                label="保存"
                busy={busy === 'save'}
                disabled={busy !== '' && busy !== 'save'}
                onClick={save}
              />
              <ActionButton
                icon={<Send size={14} />}
                label="测试"
                tone="success"
                busy={busy === 'test'}
                disabled={!enabled || !savedEnabled || (busy !== '' && busy !== 'test')}
                onClick={() => {
                  setTestEmail('');
                  setTestOpen(true);
                }}
              />
            </>
          ) : null
        }
      >
        <p className="mb-5 text-sm leading-6 text-[var(--bm-text-muted)]">
          通过 SMTP 发送找回密码验证码。
        </p>
        <div className="space-y-3">
          <EnableToggle
            enabled={enabled}
            disabled={!canManage}
            onChange={value => {
              setEnabled(value);
              setClearRequested(false);
            }}
            label="找回密码"
            enabledText="可用于发送找回密码验证码"
            disabledText="关闭后不参与找回密码"
          />
          <div className="space-y-3">
            {fields.slice(0, 6).map(field => (
              <div key={field.key}>
                <ConfigField
                  field={field}
                  value={form[field.key]}
                  secretConfigured={field.key === 'password' && Boolean(form.hasPassword)}
                  disabled={!canManage}
                  onChange={value => updateField(field, value)}
                />
              </div>
            ))}
          </div>
          <div className="space-y-3">
            {fields.slice(6).map(field => (
              <div key={field.key}>
                <ConfigField
                  field={field}
                  value={form[field.key]}
                  disabled={!canManage}
                  onChange={value => updateField(field, value)}
                />
              </div>
            ))}
          </div>
        </div>
      </SettingsDetailPanel>
      <Dialog open={testOpen} onOpenChange={setTestOpen}>
        <DialogContent className="bm-dialog-panel sm:max-w-md">
          <DialogHeader className="space-y-1.5">
            <DialogTitle>发送测试邮件</DialogTitle>
            <DialogDescription>输入临时测试邮箱，仅用于本次发送，不会保存。</DialogDescription>
          </DialogHeader>
          <div className="space-y-1.5">
            <label className="text-xs text-[var(--bm-text-muted)]" htmlFor="smtp-test-email">
              测试收件人
            </label>
            <Input
              id="smtp-test-email"
              type="email"
              value={testEmail}
              placeholder="admin@example.com"
              onChange={event => setTestEmail(event.target.value)}
              onKeyDown={event => {
                if (event.key === 'Enter') {
                  event.preventDefault();
                  void sendTest();
                }
              }}
            />
          </div>
          <DialogFooter>
            <button
              type="button"
              className="bm-action-button rounded-lg border px-4 py-2 text-sm disabled:cursor-not-allowed disabled:opacity-50"
              disabled={busy === 'test'}
              onClick={() => setTestOpen(false)}
            >
              取消
            </button>
            <ActionButton
              icon={<Send size={14} />}
              label="发送测试"
              tone="success"
              busy={busy === 'test'}
              onClick={sendTest}
            />
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </SettingsSplitLayout>
  );
}

function parsePort(value: unknown) {
  const text = String(value ?? '').trim();
  if (!/^\d+$/.test(text)) return 0;
  const port = Number(text);
  return Number.isInteger(port) && port >= 1 && port <= 65535 ? port : 0;
}

// validateEmailForm 启用找回密码时校验 SMTP 必填项，与后端保存校验保持一致
function validateEmailForm(form: EmailForm, secretConfigured: boolean) {
  if (Boolean(form.useTLS) && Boolean(form.startTLS)) {
    return 'TLS 与 STARTTLS 不能同时启用';
  }
  for (const [key, label] of [
    ['smtpHost', 'SMTP 主机'],
    ['smtpPort', 'SMTP 端口'],
    ['username', '用户名'],
  ] as const) {
    if (!String(form[key] ?? '').trim()) {
      return `${label}不能为空`;
    }
    if (key === 'smtpPort' && !parsePort(form.smtpPort)) {
      return 'SMTP 端口需为 1 到 65535 之间的整数';
    }
  }
  if (!String(form.password ?? '').trim() && !secretConfigured) {
    return '密码不能为空';
  }
  if (!String(form.from ?? '').trim()) {
    return '发件人不能为空';
  }
  return '';
}

// prepareEmailConfig 保存前按联动规则归一端口与 TLS 勾选项，保证落库配置自洽
function prepareEmailConfig(form: EmailForm, enabled: boolean) {
  const next: EmailForm = { ...form };
  delete next.hasPassword;
  if (!enabled) return next;
  if (next.useTLS) {
    next.smtpPort = '465';
    next.allowInsecureAuth = false;
  } else if (next.startTLS) {
    next.smtpPort = '587';
    next.allowInsecureAuth = false;
  }
  return next;
}
