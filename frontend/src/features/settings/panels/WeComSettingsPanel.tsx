import { MessageCircle, Save, Trash2 } from 'lucide-react';
import { useCallback, useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';
import { showErrorToast } from '@/lib/toast-errors';
import { WECOM_PROVIDERS_CHANGED_EVENT } from '@/lib/wecom';
import { fetchWeComProvider, saveWeComProvider } from '@/features/settings/api';
import {
  ActionButton,
  ConfigField,
  EnableToggle,
  SettingsDetailHeader,
  SettingsDetailPanel,
  type SettingsField,
} from '../settings-primitives';

type WeComForm = Record<string, unknown>;

type WecomAuthMode = 'direct' | 'sso';

const defaultForm: WeComForm = {
  authMode: 'direct',
  corpid: '',
  agentid: '',
  secret: '',
  redirectPrefix: '',
  ssoBaseUrl: '',
  ssoAppID: '',
  ssoAppSecret: '',
};

const directFields: SettingsField[] = [
  { key: 'corpid', label: '企业 ID（corpid）', placeholder: 'ww********', required: true },
  {
    key: 'agentid',
    label: '应用 AgentID',
    placeholder: '1000002',
    required: true,
    inputMode: 'numeric',
  },
  {
    key: 'secret',
    label: '应用 Secret',
    type: 'password',
    placeholder: '请输入自建应用的应用密钥',
    required: true,
  },
  {
    key: 'redirectPrefix',
    label: '回调地址前缀',
    placeholder: '留空则按当前访问地址推断',
    labelHint: '企业微信服务器需能访问，如 https://bm.example.com',
  },
];

const ssoFields: SettingsField[] = [
  {
    key: 'ssoBaseUrl',
    label: '认证中心地址',
    placeholder: 'https://auth.example.com',
    required: true,
    labelHint: '统一认证中心（wecom-auth-center）的外部访问地址',
  },
  {
    key: 'ssoAppID',
    label: '应用标识',
    placeholder: 'backup-monitor',
    required: true,
    labelHint: '认证中心 config.yaml 中 apps 下的条目名',
  },
  {
    key: 'ssoAppSecret',
    label: '应用密钥',
    type: 'password',
    placeholder: '与认证中心 apps 配置的 app_secret 一致',
    required: true,
  },
];

const directGuidance =
  '配置步骤：企业微信管理后台 →「应用管理」→ 自建应用（记录 AgentID 与 Secret）→' +
  '在「网页授权及 JS-SDK」中把回调域名加入可信域名 → 在「企业可信 IP」中加入本服务出口 IP。' +
  '扫码确认后由企微官方登录面板回调授权码，本系统自动完成登录或绑定。';

const ssoGuidance =
  '配置步骤：部署企业微信统一认证中心（wecom-auth-center）→ 在认证中心 config.yaml 的 apps 下为本系统新增条目：' +
  'domain 填本系统外部访问地址、callback_path 填 /login、app_secret 填 32 位以上随机密钥 →' +
  '在上方填写认证中心地址、应用标识与应用密钥（与认证中心保持一致）→ 重启认证中心使配置生效。' +
  '登录时本系统跳转认证中心完成企微扫码，认证中心携带一次性 ticket 回跳本系统 /login 完成登录或绑定。';

// WeComSettingsPanel 企业微信认证配置详情面板
export function WeComSettingsPanel({
  canManage,
  onChanged,
}: {
  canManage: boolean;
  onChanged?: () => void;
}) {
  const [name, setName] = useState('企业微信');
  const [enabled, setEnabled] = useState(false);
  const [form, setForm] = useState<WeComForm>(defaultForm);
  const [busy, setBusy] = useState('');
  const [error, setError] = useState('');
  const [clearRequested, setClearRequested] = useState(false);
  const loadPromiseRef = useRef<Promise<void> | null>(null);
  const authMode: WecomAuthMode = form.authMode === 'sso' ? 'sso' : 'direct';

  const load = useCallback(() => {
    if (loadPromiseRef.current) return loadPromiseRef.current;
    const promise = (async () => {
      setError('');
      try {
        const setting = await fetchWeComProvider();
        setName(setting.name || '企业微信');
        setEnabled(setting.enabled);
        setForm({ ...defaultForm, ...setting.config });
      } catch (err) {
        const message = err instanceof Error ? err.message : '读取企业微信认证配置失败';
        setError(message);
        showErrorToast(err, '读取企业微信认证配置失败');
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
    setForm(current => ({ ...current, [field.key]: value }));
  }

  async function save() {
    const displayName = name.trim();
    if (!displayName) {
      toast.error('显示名称不能为空');
      return;
    }
    const result = prepareConfig(form, enabled);
    if (result.error) {
      toast.error(result.error);
      return;
    }
    setBusy('save');
    try {
      const saved = await saveWeComProvider({
        name: displayName,
        enabled,
        clearConfig: clearRequested,
        config: result.config,
      });
      setName(saved.name);
      setEnabled(saved.enabled);
      setForm({ ...defaultForm, ...saved.config });
      setClearRequested(false);
      toast.success('企业微信认证配置已保存');
      window.dispatchEvent(new Event(WECOM_PROVIDERS_CHANGED_EVENT));
      onChanged?.();
    } catch (err) {
      showErrorToast(err, '企业微信认证配置保存失败');
    } finally {
      setBusy('');
    }
  }

  function clearConfig() {
    setEnabled(false);
    setForm(defaultForm);
    setClearRequested(true);
  }

  return (
    <SettingsDetailPanel
      header={
        <SettingsDetailHeader
          icon={MessageCircle}
          color="#22d3ee"
          title={name}
          subtitle={enabled ? '已启用' : '未启用'}
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
              onClick={clearConfig}
            />
            <ActionButton
              icon={<Save size={14} />}
              label="保存"
              busy={busy === 'save'}
              disabled={busy !== '' && busy !== 'save'}
              onClick={save}
            />
          </>
        ) : null
      }
    >
      {error ? (
        <p className="mb-4 rounded-lg border border-amber-400/25 bg-amber-500/10 p-3 text-sm text-amber-400">
          {error}
        </p>
      ) : null}
      <p className="mb-4 text-sm leading-6 text-[var(--bm-text-muted)]">
        启用后登录页提供企业微信扫码登录，用户可在右上角菜单绑定与解绑企微账号。
      </p>
      <div className="space-y-3">
        <EnableToggle
          enabled={enabled}
          disabled={!canManage}
          onChange={value => {
            setEnabled(value);
            setClearRequested(false);
          }}
          label="启用认证"
          enabledText="登录页将显示企业微信扫码登录方式"
          disabledText="关闭后不会显示在登录页"
        />
        <ConfigField
          field={{ key: 'name', label: '显示名称' }}
          value={name}
          disabled={true}
          onChange={value => {
            setName(String(value ?? ''));
            setClearRequested(false);
          }}
        />
        <div className="space-y-3">
          <SectionTitle title="认证方式" />
          <AuthModeSwitch
            value={authMode}
            disabled={!canManage}
            onChange={value => {
              setForm(current => ({ ...current, authMode: value }));
              setClearRequested(false);
            }}
          />
          <SectionTitle title="应用配置" />
          {(authMode === 'sso' ? ssoFields : directFields).map(field => (
            <div key={field.key}>
              <ConfigField
                field={field}
                value={form[field.key]}
                secretConfigured={
                  (field.key === 'secret' && Boolean(form.hasSecret)) ||
                  (field.key === 'ssoAppSecret' && Boolean(form.hasSsoAppSecret))
                }
                disabled={!canManage}
                onChange={value => updateField(field, value)}
              />
            </div>
          ))}
        </div>
        <p className="rounded-lg border border-[var(--bm-border)] bg-[var(--bm-control-bg-soft)] p-3 text-xs leading-5 text-[var(--bm-text-muted)]">
          {authMode === 'sso' ? ssoGuidance : directGuidance}
        </p>
      </div>
    </SettingsDetailPanel>
  );
}

function SectionTitle({ title }: { title: string }) {
  return <div className="text-xs font-semibold leading-5 text-[var(--bm-text)]">{title}</div>;
}

const authModeOptions: Array<{ value: WecomAuthMode; label: string; description: string }> = [
  { value: 'direct', label: '直连企业微信', description: '本系统直接持有企微应用凭据并完成扫码' },
  { value: 'sso', label: '统一认证中心', description: '经由 wecom-auth-center 完成企微扫码后回跳' },
];

// AuthModeSwitch 企业微信认证方式切换：直连与统一认证中心两种模式
function AuthModeSwitch({
  value,
  disabled,
  onChange,
}: {
  value: WecomAuthMode;
  disabled: boolean;
  onChange: (mode: WecomAuthMode) => void;
}) {
  return (
    <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
      {authModeOptions.map(option => {
        const active = option.value === value;
        return (
          <button
            key={option.value}
            type="button"
            disabled={disabled}
            onClick={() => onChange(option.value)}
            className={[
              'rounded-xl border p-3 text-left transition-all duration-300 ease-out',
              'hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-60 disabled:hover:translate-y-0',
              active
                ? 'border-[var(--bm-accent-text)] bg-[rgba(59,130,246,0.12)] hover:shadow-[0_6px_18px_rgba(37,99,235,0.16)]'
                : 'border-[var(--bm-border)] bg-[var(--bm-control-bg-soft)] hover:border-[var(--bm-accent-text)] hover:bg-[rgba(59,130,246,0.06)]',
            ].join(' ')}
          >
            <span
              className="flex items-center gap-1.5 text-sm font-medium"
              style={{ color: 'var(--bm-text)' }}
            >
              <span
                className="h-2 w-2 rounded-full"
                style={{ background: active ? 'var(--bm-accent-text)' : 'var(--bm-border)' }}
              />
              {option.label}
            </span>
            <span
              className="mt-1 block text-xs leading-4"
              style={{ color: 'var(--bm-text-muted)' }}
            >
              {option.description}
            </span>
          </button>
        );
      })}
    </div>
  );
}

function prepareConfig(form: WeComForm, enabled: boolean) {
  const config = Object.fromEntries(
    Object.entries(form).filter(
      ([key, value]) => key !== 'hasSecret' && key !== 'hasSsoAppSecret' && value !== ''
    )
  );
  if (!enabled) return { config, error: '' };
  if (form.authMode === 'sso') {
    const baseURL = String(config.ssoBaseUrl || '').trim();
    if (!baseURL) {
      return { config: {}, error: '认证中心地址不能为空' };
    }
    if (!/^https?:\/\//i.test(baseURL)) {
      return { config: {}, error: '认证中心地址需以 http:// 或 https:// 开头' };
    }
    if (!String(config.ssoAppID || '').trim()) {
      return { config: {}, error: '应用标识不能为空' };
    }
    if (!config.ssoAppSecret && !form.hasSsoAppSecret) {
      return { config: {}, error: '应用密钥不能为空' };
    }
    return { config, error: '' };
  }
  for (const field of [
    { key: 'corpid', label: '企业 ID（corpid）' },
    { key: 'agentid', label: '应用 AgentID' },
  ] as const) {
    if (!String(config[field.key] || '').trim()) {
      return { config: {}, error: `${field.label}不能为空` };
    }
  }
  if (!config.secret && !form.hasSecret) {
    return { config: {}, error: '应用 Secret 不能为空' };
  }
  return { config, error: '' };
}
