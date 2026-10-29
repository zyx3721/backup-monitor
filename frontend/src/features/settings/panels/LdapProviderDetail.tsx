import { CheckCircle2, Network, Save, Trash2 } from 'lucide-react';
import { useCallback, useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';
import { showErrorToast } from '@/lib/toast-errors';
import { fetchAuthProvider, saveAuthProvider, testAuthProvider } from '@/features/settings/api';
import {
  ActionButton,
  ConfigField,
  EnableToggle,
  SettingsDetailHeader,
  SettingsDetailPanel,
  type SettingsField,
} from '../settings-primitives';

type LDAPForm = Record<string, unknown>;

const defaultForm: LDAPForm = {
  host: '',
  port: '',
  baseDN: '',
  bindDN: '',
  bindPassword: '',
  useTLS: false,
  startTLS: false,
  insecureSkipVerify: false,
  userFilter: '',
  timeoutSeconds: '',
  groupFilter: '',
};

const fields: SettingsField[] = [
  { key: 'host', label: '服务器地址', placeholder: 'ldap.example.com', required: true },
  { key: 'port', label: '端口', placeholder: '389', required: true, inputMode: 'numeric' },
  { key: 'baseDN', label: 'Base DN', placeholder: 'DC=example,DC=com', required: true },
  {
    key: 'userFilter',
    label: '用户过滤器',
    placeholder: '(sAMAccountName={username})',
    required: true,
  },
  {
    key: 'bindDN',
    label: '绑定 DN',
    placeholder: 'CN=svc,OU=Users,DC=example,DC=com',
    required: true,
  },
  {
    key: 'bindPassword',
    label: '绑定密码',
    type: 'password',
    placeholder: '请输入绑定账号密码',
    required: true,
  },
  { key: 'useTLS', label: '启用 LDAPS', type: 'checkbox' },
  { key: 'startTLS', label: '启用 STARTTLS', type: 'checkbox' },
  {
    key: 'insecureSkipVerify',
    label: '跳过证书校验',
    type: 'checkbox',
  },
  { key: 'timeoutSeconds', label: '超时时间', placeholder: '8', type: 'number' },
  { key: 'groupFilter', label: '用户组过滤器', placeholder: 'cn=ops,dc=example,dc=com' },
];

// LdapProviderDetail AD/LDAP 认证配置详情面板
export function LdapProviderDetail({
  canManage,
  onChanged,
}: {
  canManage: boolean;
  onChanged?: () => void;
}) {
  const [name, setName] = useState('AD/LDAP');
  const [enabled, setEnabled] = useState(false);
  const [savedEnabled, setSavedEnabled] = useState(false);
  const [form, setForm] = useState<LDAPForm>(defaultForm);
  const [busy, setBusy] = useState('');
  const [error, setError] = useState('');
  const [clearRequested, setClearRequested] = useState(false);
  const loadPromiseRef = useRef<Promise<void> | null>(null);

  const load = useCallback(() => {
    if (loadPromiseRef.current) return loadPromiseRef.current;
    const promise = (async () => {
      setError('');
      try {
        const setting = await fetchAuthProvider();
        setName(setting.name || 'AD/LDAP');
        setEnabled(setting.enabled);
        setSavedEnabled(setting.enabled);
        setForm({ ...defaultForm, ...setting.config });
      } catch (err) {
        const message = err instanceof Error ? err.message : '读取 AD/LDAP 认证配置失败';
        setError(message);
        showErrorToast(err, '读取 AD/LDAP 认证配置失败');
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
        next.port = 636;
      }
      if (field.key === 'startTLS' && value === true) {
        next.useTLS = false;
        next.port = 389;
      }
      return next;
    });
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
      const saved = await saveAuthProvider({
        name: displayName,
        enabled,
        clearConfig: clearRequested,
        config: result.config,
      });
      setName(saved.name);
      setEnabled(saved.enabled);
      setSavedEnabled(saved.enabled);
      setForm({ ...defaultForm, ...saved.config });
      setClearRequested(false);
      toast.success('AD/LDAP 认证配置已保存');
      onChanged?.();
    } catch (err) {
      showErrorToast(err, 'AD/LDAP 认证配置保存失败');
    } finally {
      setBusy('');
    }
  }

  function clearConfig() {
    setEnabled(false);
    setForm(defaultForm);
    setClearRequested(true);
  }

  async function test() {
    setBusy('test');
    try {
      const result = await testAuthProvider();
      toast.success(`认证连接测试通过，成功匹配 ${result.matchedUsers} 个用户`);
    } catch (err) {
      showErrorToast(err, '认证连接测试失败');
    } finally {
      setBusy('');
    }
  }

  return (
    <SettingsDetailPanel
      header={
        <SettingsDetailHeader
          icon={Network}
          color="#38bdf8"
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
            <ActionButton
              icon={<CheckCircle2 size={14} />}
              label="测试"
              tone="success"
              busy={busy === 'test'}
              disabled={!enabled || !savedEnabled || (busy !== '' && busy !== 'test')}
              onClick={test}
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
        通过企业目录服务实现统一身份认证，支持 AD/LDAP 登录。
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
          enabledText="登录页将显示 AD/LDAP 认证登录方式"
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
          <SectionTitle title="必填配置" />
          {fields.slice(0, 6).map(field => (
            <div key={field.key}>
              <ConfigField
                field={field}
                value={form[field.key]}
                secretConfigured={field.key === 'bindPassword' && Boolean(form.hasBindPassword)}
                disabled={!canManage}
                onChange={value => updateField(field, value)}
              />
            </div>
          ))}
        </div>
        <div className="space-y-3">
          <SectionTitle title="可选配置" />
          {fields.slice(6).map(field => (
            <div key={field.key}>
              <ConfigField
                field={field}
                value={form[field.key]}
                secretConfigured={field.key === 'bindPassword' && Boolean(form.hasBindPassword)}
                disabled={!canManage}
                onChange={value => updateField(field, value)}
              />
            </div>
          ))}
        </div>
      </div>
    </SettingsDetailPanel>
  );
}

function SectionTitle({ title }: { title: string }) {
  return <div className="text-xs font-semibold leading-5 text-[var(--bm-text)]">{title}</div>;
}

function prepareConfig(form: LDAPForm, enabled: boolean) {
  const config = Object.fromEntries(
    Object.entries(form).filter(([key, value]) => key !== 'hasBindPassword' && value !== '')
  );
  if (!enabled) return { config, error: '' };
  if (Boolean(config.useTLS) && Boolean(config.startTLS))
    return { config: {}, error: 'LDAPS 与 StartTLS 不能同时启用' };
  for (const field of [
    { key: 'host', label: '服务器地址' },
    { key: 'port', label: '端口', number: true },
    { key: 'baseDN', label: 'Base DN' },
    { key: 'userFilter', label: '用户过滤器' },
    { key: 'bindDN', label: '绑定 DN' },
  ] as const) {
    if (field.number) {
      if (!Number(config[field.key])) return { config: {}, error: `${field.label}不能为空` };
      continue;
    }
    if (!String(config[field.key] || '').trim()) {
      return { config: {}, error: `${field.label}不能为空` };
    }
  }
  if (!config.bindPassword && !form.hasBindPassword) {
    return { config: {}, error: '绑定密码不能为空' };
  }
  const port = Number(config.port || 0);
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    return { config: {}, error: '端口需为 1 到 65535 之间的整数' };
  }
  config.port = port;
  return { config, error: '' };
}
