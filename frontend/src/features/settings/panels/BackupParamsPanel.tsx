import { RotateCcw, Save } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';
import { showErrorToast } from '@/lib/toast-errors';
import { fetchBackupSettings, saveBackupSettings } from '@/features/backup/api';
import type { BackupSettings } from '@/features/backup/types';
import { backupSections, type BackupSection } from '../backup-sections';
import {
  ActionButton,
  EnableToggle,
  NumberControl,
  SettingsDetailHeader,
  SettingsDetailPanel,
} from '../settings-primitives';

// DEFAULT_BACKUP_SETTINGS 与后端 domain.DefaultAppSettings 保持一致，恢复默认后保存即写入数据库并下发给 Agent
const DEFAULT_BACKUP_SETTINGS: BackupSettings = {
  reportTimeoutMinutes: 3,
  offlineTimeoutMinutes: 6,
  notRunGraceMinutes: 30,
  scheduleMatchWindowMinutes: 60,
  schedulerIntervalSec: 30,
  dedupeAlerts: true,
  retentionDays: 90,
  auditRetentionDays: 90,
  notifyRetentionDays: 90,
  requireAgentToken: false,
  agent: {
    reportIntervalSec: 60,
    httpTimeoutSec: 0,
    wsBackoffMinSec: 0,
    wsBackoffMaxSec: 0,
  },
};

export function BackupSettingsDetail({
  section,
  canManage,
}: {
  section: BackupSection;
  canManage: boolean;
}) {
  const [form, setForm] = useState<BackupSettings | null>(null);
  const [saved, setSaved] = useState<BackupSettings | null>(null);
  const [busy, setBusy] = useState(false);
  const meta = backupSections.find(item => item.id === section) ?? backupSections[0];

  const load = useCallback(async () => {
    try {
      const value = await fetchBackupSettings();
      setForm(value);
      setSaved(value);
    } catch (error) {
      showErrorToast(error, '读取备份参数失败');
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  if (!form || !saved) {
    return (
      <div className="grid min-h-40 place-items-center rounded-lg border border-[var(--bm-border)] bg-white/[0.035] text-sm text-[var(--bm-text-muted)]">
        正在加载备份参数
      </div>
    );
  }

  function update<K extends keyof BackupSettings>(key: K, value: BackupSettings[K]) {
    setForm(current => (current ? { ...current, [key]: value } : current));
  }

  function updateAgent(key: keyof BackupSettings['agent'], value: number) {
    setForm(current =>
      current ? { ...current, agent: { ...current.agent, [key]: value } } : current
    );
  }

  function resetSection() {
    setForm(current => (current ? { ...current, ...sectionDefaults(section) } : current));
  }

  async function save() {
    if (!form) return;
    if (form.offlineTimeoutMinutes < form.reportTimeoutMinutes) {
      toast.error('主机离线阈值不能小于主机上报超时');
      return;
    }
    setBusy(true);
    try {
      const next = await saveBackupSettings(form, section);
      setForm(next);
      setSaved(next);
      toast.success(`${meta.title}配置已更新`);
    } catch (error) {
      showErrorToast(error, '保存备份参数失败');
    } finally {
      setBusy(false);
    }
  }

  return (
    <SettingsDetailPanel
      header={
        <SettingsDetailHeader
          icon={meta.icon}
          color={meta.color}
          title={meta.title}
          subtitle={meta.description}
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
      {section === 'thresholds' ? (
        <div className="grid gap-3 lg:grid-cols-2">
          <NumberControl
            label="主机上报超时"
            description="超过该时长未收到上报，主机判为「异常」并告警"
            unit="分钟"
            min={1}
            max={1440}
            value={form.reportTimeoutMinutes}
            disabled={!canManage}
            onChange={value => update('reportTimeoutMinutes', value)}
          />
          <NumberControl
            label="主机离线阈值"
            description="超过该时长未上报，主机判为「离线」（应不小于上报超时）"
            unit="分钟"
            min={1}
            max={4320}
            value={form.offlineTimeoutMinutes}
            disabled={!canManage}
            onChange={value => update('offlineTimeoutMinutes', value)}
          />
          <NumberControl
            label="未执行宽限期"
            description="计划时刻后多久仍未执行才判为「未执行」"
            unit="分钟"
            min={1}
            max={1440}
            value={form.notRunGraceMinutes}
            disabled={!canManage}
            onChange={value => update('notRunGraceMinutes', value)}
          />
          <NumberControl
            label="计划匹配窗口"
            description="执行时间晚于「计划时刻 + 窗口」视为「未按时执行」"
            unit="分钟"
            min={1}
            max={1440}
            value={form.scheduleMatchWindowMinutes}
            disabled={!canManage}
            onChange={value => update('scheduleMatchWindowMinutes', value)}
          />
          <NumberControl
            label="巡检间隔"
            description="服务端兜底巡检周期"
            unit="秒"
            min={5}
            max={600}
            step={1}
            value={form.schedulerIntervalSec}
            disabled={!canManage}
            onChange={value => update('schedulerIntervalSec', value)}
          />
          <NumberControl
            label="执行记录保留"
            description="超过保留期的执行记录自动清理；0 表示不自动清理"
            unit="天"
            min={0}
            max={3650}
            value={form.retentionDays}
            disabled={!canManage}
            onChange={value => update('retentionDays', value)}
          />
          <NumberControl
            label="发送记录保留"
            description="超过保留期的发送记录自动清理；0 表示不自动清理"
            unit="天"
            min={0}
            max={3650}
            value={form.notifyRetentionDays}
            disabled={!canManage}
            onChange={value => update('notifyRetentionDays', value)}
          />
          <NumberControl
            label="审计日志保留"
            description="超过保留期的审计日志自动清理；0 表示不自动清理"
            unit="天"
            min={0}
            max={365}
            value={form.auditRetentionDays}
            disabled={!canManage}
            onChange={value => update('auditRetentionDays', value)}
          />
        </div>
      ) : null}

      {section === 'agent' ? (
        <div className="grid gap-3 lg:grid-cols-2">
          <NumberControl
            label="默认上报间隔"
            description="采集配置未单独指定上报间隔时使用"
            unit="秒"
            min={10}
            max={86400}
            value={form.agent.reportIntervalSec}
            disabled={!canManage}
            onChange={value => updateAgent('reportIntervalSec', value)}
          />
          <NumberControl
            label="HTTP 超时"
            description="Agent 上报 / 拉取配置的超时时间；0 表示默认 20 秒"
            unit="秒"
            min={0}
            max={600}
            value={form.agent.httpTimeoutSec}
            disabled={!canManage}
            onChange={value => updateAgent('httpTimeoutSec', value)}
          />
          <NumberControl
            label="WS 重连退避下限"
            description="心跳断线重连的最小等待时间；0 表示默认 2 秒"
            unit="秒"
            min={0}
            max={120}
            value={form.agent.wsBackoffMinSec}
            disabled={!canManage}
            onChange={value => updateAgent('wsBackoffMinSec', value)}
          />
          <NumberControl
            label="WS 重连退避上限"
            description="心跳断线重连的最大等待时间；0 表示默认 30 秒"
            unit="秒"
            min={0}
            max={600}
            value={form.agent.wsBackoffMaxSec}
            disabled={!canManage}
            onChange={value => updateAgent('wsBackoffMaxSec', value)}
          />
        </div>
      ) : null}

      {section === 'alarm' ? (
        <div className="space-y-3">
          <EnableToggle
            enabled={form.dedupeAlerts}
            disabled={!canManage}
            onChange={value => update('dedupeAlerts', value)}
            label="告警去重"
            enabledText="同一对象同类型告警仅保留一条未处理记录"
            disabledText="同类型告警按产生时间逐条保留"
          />
          <EnableToggle
            enabled={form.requireAgentToken}
            disabled={!canManage}
            onChange={value => update('requireAgentToken', value)}
            label="要求 Agent 令牌"
            enabledText="Agent 上报与配置拉取必须携带有效令牌"
            disabledText="Agent 无需令牌即可上报与拉取配置"
          />
        </div>
      ) : null}
    </SettingsDetailPanel>
  );
}

function sectionDefaults(section: BackupSection): Partial<BackupSettings> {
  if (section === 'thresholds') {
    return {
      reportTimeoutMinutes: DEFAULT_BACKUP_SETTINGS.reportTimeoutMinutes,
      offlineTimeoutMinutes: DEFAULT_BACKUP_SETTINGS.offlineTimeoutMinutes,
      notRunGraceMinutes: DEFAULT_BACKUP_SETTINGS.notRunGraceMinutes,
      scheduleMatchWindowMinutes: DEFAULT_BACKUP_SETTINGS.scheduleMatchWindowMinutes,
      schedulerIntervalSec: DEFAULT_BACKUP_SETTINGS.schedulerIntervalSec,
      retentionDays: DEFAULT_BACKUP_SETTINGS.retentionDays,
      auditRetentionDays: DEFAULT_BACKUP_SETTINGS.auditRetentionDays,
      notifyRetentionDays: DEFAULT_BACKUP_SETTINGS.notifyRetentionDays,
    };
  }
  if (section === 'agent') {
    return { agent: { ...DEFAULT_BACKUP_SETTINGS.agent } };
  }
  return {
    dedupeAlerts: DEFAULT_BACKUP_SETTINGS.dedupeAlerts,
    requireAgentToken: DEFAULT_BACKUP_SETTINGS.requireAgentToken,
  };
}
