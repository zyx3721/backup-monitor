import { Image, KeyRound, RotateCcw, Save, UploadCloud } from 'lucide-react';
import { useCallback, useEffect, useRef, useState, type DragEvent } from 'react';
import { toast } from 'sonner';
import { showErrorToast } from '@/lib/toast-errors';
import { fetchSystemBaseConfig, updateSystemBaseConfig } from '@/features/settings/api';
import type { SystemBaseConfig } from '@/features/settings/types';
import { setBrandSettings } from '@/lib/branding';
import { BackupSettingsDetail } from './BackupParamsPanel';
import { backupSections, type BackupSection } from '../backup-sections';
import {
  ActionButton,
  ConfigField,
  NumberControl,
  SettingsDetailHeader,
  SettingsDetailPanel,
  SettingsSplitLayout,
} from '../settings-primitives';
import { cardStyle } from '../settings-style';

type BaseConfigSection = 'brand' | 'security';

type BaseSection = BaseConfigSection | BackupSection;

const sections: Array<{
  id: BaseSection;
  title: string;
  description: string;
  icon: typeof Image;
  color: string;
}> = [
  {
    id: 'brand',
    title: '品牌标识',
    description: '网站名称、登录展示和图标',
    icon: Image,
    color: '#38bdf8',
  },
  {
    id: 'security',
    title: '安全时效',
    description: '验证码时效、发送限流、扫码有效期与登录失败锁定',
    icon: KeyRound,
    color: '#22c55e',
  },
  ...backupSections,
];

export function BaseSettingsPanel({ canManage }: { canManage: boolean }) {
  const [active, setActive] = useState<BaseSection>('brand');
  const [form, setForm] = useState<SystemBaseConfig | null>(null);
  const [saved, setSaved] = useState<SystemBaseConfig | null>(null);
  const [busy, setBusy] = useState(false);
  const [dragging, setDragging] = useState(false);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const load = useCallback(async () => {
    try {
      const config = await fetchSystemBaseConfig();
      setForm(config);
      setSaved(config);
      setBrandSettings(config);
    } catch (err) {
      showErrorToast(err, '读取基础配置失败');
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  if (!form || !saved) {
    return (
      <div className="grid min-h-40 place-items-center text-sm text-[var(--bm-text-muted)]">
        正在加载基础配置
      </div>
    );
  }

  const selected = sections.find(section => section.id === active) ?? sections[0];
  const configSection: BaseConfigSection | null =
    active === 'brand' || active === 'security' ? active : null;

  async function save() {
    if (!form || !saved || !configSection) return;
    const next = { ...saved, ...pickSectionPatch(form, configSection) };
    const error = validateSection(configSection, next);
    if (error) {
      showErrorToast(error);
      return;
    }
    setBusy(true);
    try {
      const updated = await updateSystemBaseConfig({ ...next, section: configSection });
      setForm(updated);
      setSaved(updated);
      setBrandSettings(updated);
      toast.success(`${selected.title}配置已更新`);
    } catch (err) {
      showErrorToast(err, '保存基础配置失败');
    } finally {
      setBusy(false);
    }
  }

  function resetSection() {
    if (configSection) update(pickSectionPatch(defaultBaseConfig, configSection));
  }

  function update(patch: Partial<SystemBaseConfig>) {
    setForm(current => (current ? { ...current, ...patch } : current));
  }

  function selectSection(section: BaseSection) {
    setActive(section);
    if (saved) setForm(saved);
  }

  async function updateFile(file: File | undefined) {
    if (!file) return;
    if (!file.type.startsWith('image/')) {
      toast.error('请选择图片文件');
      return;
    }
    if (file.size > 256 * 1024) {
      toast.error('图标文件不能超过 256KB');
      return;
    }
    update({ iconData: await readFileAsDataURL(file) });
  }

  function onDrop(event: DragEvent<HTMLButtonElement>) {
    event.preventDefault();
    setDragging(false);
    void updateFile(event.dataTransfer.files[0]);
  }

  return (
    <SettingsSplitLayout
      sidebarLabel="基础配置"
      sidebar={sections.map(section => (
        <BaseSectionCard
          key={section.id}
          section={section}
          active={active === section.id}
          onClick={() => selectSection(section.id)}
        />
      ))}
    >
      {active === 'brand' || active === 'security' ? (
        <SettingsDetailPanel
          header={
            <SettingsDetailHeader
              icon={selected.icon}
              color={selected.color}
              title={selected.title}
              subtitle={selected.description}
            />
          }
          actions={
            canManage ? (
              <>
                <ActionButton
                  icon={<RotateCcw size={14} />}
                  label="恢复默认"
                  tone="muted"
                  disabled={busy}
                  onClick={resetSection}
                />
                <ActionButton icon={<Save size={14} />} label="保存" busy={busy} onClick={save} />
              </>
            ) : null
          }
        >
          {active === 'brand' ? (
            <BrandPanel
              form={form}
              dragging={dragging}
              fileInputRef={fileInputRef}
              setDragging={setDragging}
              onDrop={onDrop}
              onFile={updateFile}
              onUpdate={update}
              disabled={!canManage}
            />
          ) : null}
          {active === 'security' ? (
            <SecurityPanel form={form} onUpdate={update} disabled={!canManage} />
          ) : null}
        </SettingsDetailPanel>
      ) : (
        <BackupSettingsDetail section={active} canManage={canManage} />
      )}
    </SettingsSplitLayout>
  );
}

const defaultBaseConfig: SystemBaseConfig = {
  siteName: 'Backup Monitor',
  loginName: 'Backup Monitor',
  appName: 'Backup Monitor',
  appSubtitle: 'Backup Control Plane',
  iconData: '/favicon.svg',
  resetCodeTtlMinutes: 10,
  resetCaptchaTtlMinutes: 1,
  passwordResetSendCooldownMinutes: 0.5,
  passwordResetRateLimitMinutes: 5,
  wecomStateTtlMinutes: 5,
  loginMaxFailures: 5,
  loginLockoutMinutes: 2,
};

function BaseSectionCard({
  section,
  active,
  onClick,
}: {
  section: (typeof sections)[number];
  active: boolean;
  onClick: () => void;
}) {
  const Icon = section.icon;
  return (
    <button
      type="button"
      onClick={onClick}
      className="bm-config-side-card flex w-full items-start gap-3 rounded-lg p-3 text-left transition-all duration-200"
      data-active={active}
      style={cardStyle(active)}
    >
      <span
        className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg"
        style={{
          color: section.color,
          background: 'rgba(255,255,255,0.05)',
          border: '1px solid rgba(255,255,255,0.08)',
        }}
      >
        <Icon size={19} />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-semibold">{section.title}</span>
        <span className="mt-1 block text-xs leading-5" style={{ color: 'var(--bm-text-muted)' }}>
          {section.description}
        </span>
      </span>
    </button>
  );
}

function BrandPanel({
  form,
  dragging,
  fileInputRef,
  setDragging,
  onDrop,
  onFile,
  onUpdate,
  disabled,
}: {
  form: SystemBaseConfig;
  dragging: boolean;
  fileInputRef: React.RefObject<HTMLInputElement | null>;
  setDragging: (value: boolean) => void;
  onDrop: (event: DragEvent<HTMLButtonElement>) => void;
  onFile: (file: File | undefined) => void;
  onUpdate: (patch: Partial<SystemBaseConfig>) => void;
  disabled?: boolean;
}) {
  return (
    <div className="space-y-5">
      <div className="grid gap-4 lg:grid-cols-2">
        <ConfigField
          field={{ key: 'siteName', label: '网站名称', placeholder: 'Backup Monitor' }}
          value={form.siteName}
          disabled={disabled}
          onChange={value => onUpdate({ siteName: String(value) })}
        />
        <ConfigField
          field={{ key: 'loginName', label: '认证页品牌名称', placeholder: 'Backup Monitor' }}
          value={form.loginName}
          disabled={disabled}
          onChange={value => onUpdate({ loginName: String(value) })}
        />
        <ConfigField
          field={{ key: 'appName', label: '控制台品牌名称', placeholder: 'Backup Monitor' }}
          value={form.appName}
          disabled={disabled}
          onChange={value => onUpdate({ appName: String(value) })}
        />
        <ConfigField
          field={{
            key: 'appSubtitle',
            label: '控制台品牌副标题',
            placeholder: 'Backup Control Plane',
          }}
          value={form.appSubtitle}
          disabled={disabled}
          onChange={value => onUpdate({ appSubtitle: String(value) })}
        />
      </div>
      <div className="grid gap-4 lg:grid-cols-[260px_minmax(0,1fr)]">
        <button
          type="button"
          disabled={disabled}
          onClick={() => {
            if (!disabled) fileInputRef.current?.click();
          }}
          onDragOver={event => {
            event.preventDefault();
            setDragging(true);
          }}
          onDragLeave={() => setDragging(false)}
          onDrop={onDrop}
          className="bm-action-button flex min-h-44 flex-col items-center justify-center gap-3 rounded-xl border border-dashed p-4"
          style={{
            borderColor: dragging ? 'rgba(96,165,250,0.78)' : 'var(--bm-border)',
            background: dragging ? 'rgba(59,130,246,0.12)' : 'rgba(255,255,255,0.026)',
            color: 'var(--bm-text-muted)',
          }}
        >
          <img
            src={form.iconData || '/favicon.svg'}
            alt="系统图标预览"
            className="h-16 w-16 rounded-xl object-contain"
          />
          <span className="flex items-center gap-2 text-sm">
            <UploadCloud size={15} />
            拖动或点击上传
          </span>
          <span className="text-xs">支持图片，建议 256KB 内</span>
        </button>
        <BrandPreview config={form} />
      </div>
      <input
        ref={fileInputRef}
        type="file"
        accept="image/*"
        disabled={disabled}
        className="hidden"
        onChange={event => void onFile(event.target.files?.[0])}
      />
    </div>
  );
}

function BrandPreview({ config }: { config: SystemBaseConfig }) {
  return (
    <div
      className="flex min-h-44 flex-col rounded-xl border p-5"
      style={{ borderColor: 'var(--bm-border)', background: 'rgba(255,255,255,0.026)' }}
    >
      <div className="text-sm font-semibold" style={{ color: 'var(--bm-text-muted)' }}>
        实时预览
      </div>
      <div className="flex flex-1 items-center">
        <div className="flex min-w-0 items-center gap-4">
          <img
            src={config.iconData || '/favicon.svg'}
            alt={config.appName}
            className="h-16 w-16 shrink-0 rounded-xl object-contain"
          />
          <div className="min-w-0">
            <div className="bm-gradient-text truncate text-xl font-bold">{config.appName}</div>
            <div
              className="mt-1 truncate text-sm tracking-widest"
              style={{ color: 'var(--bm-text-muted)' }}
            >
              {config.appSubtitle}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function SecurityPanel({
  form,
  onUpdate,
  disabled,
}: {
  form: SystemBaseConfig;
  onUpdate: (patch: Partial<SystemBaseConfig>) => void;
  disabled?: boolean;
}) {
  return (
    <div className="grid gap-3 lg:grid-cols-2">
      <NumberControl
        label="找回密码验证码有效期"
        unit="分钟"
        value={form.resetCodeTtlMinutes}
        min={1}
        max={60}
        disabled={disabled}
        onChange={value => onUpdate({ resetCodeTtlMinutes: value })}
      />
      <NumberControl
        label="图形验证码有效期"
        unit="分钟"
        value={form.resetCaptchaTtlMinutes}
        min={1}
        max={10}
        disabled={disabled}
        onChange={value => onUpdate({ resetCaptchaTtlMinutes: value })}
      />
      <NumberControl
        label="发送冷却时间"
        description={`验证码发送后 ${form.passwordResetSendCooldownMinutes} 分钟内不可重复请求`}
        unit="分钟"
        value={form.passwordResetSendCooldownMinutes}
        min={0.5}
        max={10}
        step={0.5}
        disabled={disabled}
        onChange={value => onUpdate({ passwordResetSendCooldownMinutes: value })}
      />
      <NumberControl
        label="频率限制统计窗口"
        description={`验证码在 ${form.passwordResetRateLimitMinutes} 分钟内最多请求 5 次`}
        unit="分钟"
        value={form.passwordResetRateLimitMinutes}
        min={5}
        max={10}
        disabled={disabled}
        onChange={value => onUpdate({ passwordResetRateLimitMinutes: value })}
      />
      <NumberControl
        label="企业微信扫码有效期"
        description="企微授权 state 的有效窗口，超时需重新扫码登录或绑定"
        unit="分钟"
        value={form.wecomStateTtlMinutes}
        min={1}
        max={60}
        disabled={disabled}
        onChange={value => onUpdate({ wecomStateTtlMinutes: value })}
      />
      <NumberControl
        label="登录失败锁定次数"
        description={`密码连续错误 ${form.loginMaxFailures} 次后锁定账号，admin 不受限`}
        unit="次"
        value={form.loginMaxFailures}
        min={3}
        max={10}
        disabled={disabled}
        onChange={value => onUpdate({ loginMaxFailures: value })}
      />
      <NumberControl
        label="登录锁定等待时长"
        description={`达到锁定次数后需等待 ${form.loginLockoutMinutes} 分钟才能重试`}
        unit="分钟"
        value={form.loginLockoutMinutes}
        min={1}
        max={10}
        disabled={disabled}
        onChange={value => onUpdate({ loginLockoutMinutes: value })}
      />
    </div>
  );
}

function pickSectionPatch(
  config: SystemBaseConfig,
  section: BaseConfigSection
): Partial<SystemBaseConfig> {
  if (section === 'brand') {
    return {
      siteName: withDefault(config.siteName, defaultBaseConfig.siteName),
      loginName: withDefault(config.loginName, defaultBaseConfig.loginName),
      appName: withDefault(config.appName, defaultBaseConfig.appName),
      appSubtitle: withDefault(config.appSubtitle, defaultBaseConfig.appSubtitle),
      iconData: withDefault(config.iconData, defaultBaseConfig.iconData),
    };
  }
  return {
    resetCodeTtlMinutes: config.resetCodeTtlMinutes,
    resetCaptchaTtlMinutes: config.resetCaptchaTtlMinutes,
    passwordResetSendCooldownMinutes: config.passwordResetSendCooldownMinutes,
    passwordResetRateLimitMinutes: config.passwordResetRateLimitMinutes,
    wecomStateTtlMinutes: config.wecomStateTtlMinutes,
    loginMaxFailures: config.loginMaxFailures,
    loginLockoutMinutes: config.loginLockoutMinutes,
  };
}

function validateSection(section: BaseConfigSection, config: SystemBaseConfig) {
  if (section === 'brand') {
    for (const [label, value] of [
      ['网站名称', config.siteName],
      ['认证页品牌名称', config.loginName],
      ['控制台品牌名称', config.appName],
      ['控制台品牌副标题', config.appSubtitle],
    ] as const) {
      if ([...value].length > 60) {
        return `${label}长度不能超过 60 个字符`;
      }
    }
  }
  return '';
}

function withDefault(value: string, fallback: string) {
  return value.trim() || fallback;
}

function readFileAsDataURL(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result || ''));
    reader.onerror = () => reject(new Error('读取图标失败'));
    reader.readAsDataURL(file);
  });
}
