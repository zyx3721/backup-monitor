import {
  AlertTriangle,
  FileUp,
  Info,
  Loader2,
  RefreshCw,
  Rocket,
  UploadCloud,
  X,
} from 'lucide-react';
import { useCallback, useEffect, useRef, useState, type ChangeEvent } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import { errorMessageOf, showErrorToast } from '@/lib/toast-errors';
import { usePageRefresh } from '@/lib/page-refresh';
import {
  fetchAgentBinaries,
  fetchHosts,
  pushAgentUpdate,
  uploadAgentBinary,
} from '@/features/backup/api';
import { formatBytes } from '@/features/backup/components';
import type { AgentBinaryMeta, AgentBinaryTargets, PushResult } from '@/features/backup/types';
import {
  HostedSlots,
  PushModeCard,
  PushResultsTable,
  SectionHeading,
  WindowsCompatHint,
  type PushPollState,
} from './AgentUpdateParts';
import { findSlotMeta, platforms, slotsByPlatform, type Platform } from './agent-update-platforms';
import { PlatformTabs } from './AgentUpdatePlatformTabs';

export function AgentUpdatePanel({ canManage }: { canManage: boolean }) {
  const [items, setItems] = useState<AgentBinaryMeta[]>([]);
  const [targets, setTargets] = useState<AgentBinaryTargets | null>(null);
  const [platform, setPlatform] = useState<Platform>('linux');
  const [versions, setVersions] = useState<Record<string, string>>({});
  const [files, setFiles] = useState<Record<Platform, File | null>>({
    linux: null,
    windows: null,
  });
  const [uploading, setUploading] = useState<Platform | null>(null);
  const [loading, setLoading] = useState(false);
  const [all, setAll] = useState(true);
  const [ips, setIps] = useState('');
  const [pushing, setPushing] = useState(false);
  const [results, setResults] = useState<{
    platform: Platform;
    results: PushResult[];
  } | null>(null);
  const [pushOutcome, setPushOutcome] = useState<'idle' | 'pending' | 'success' | 'failed'>('idle');
  const [poll, setPoll] = useState<PushPollState>({
    active: false,
    nextCheckAt: null,
    rounds: 0,
    checking: false,
  });
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const pickTarget = useRef<Platform>('linux');
  const refreshTimers = useRef<number[]>([]);
  const baselines = useRef<Record<string, string>>({});
  const startedBaselines = useRef<Record<string, string>>({});

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await fetchAgentBinaries();
      setItems(data.items);
      setTargets(data.targets);
      return true;
    } catch (error) {
      showErrorToast(error, '读取 Agent 二进制失败');
      return false;
    } finally {
      setLoading(false);
    }
  }, []);

  const refreshList = useCallback(async () => {
    if (await load()) toast.success('清单已刷新');
  }, [load]);

  const trackPush = useCallback((snapshot: { results: PushResult[] }) => {
    const deliveredIPs = snapshot.results.filter(item => item.delivered).map(item => item.ip);
    baselines.current = Object.fromEntries(
      snapshot.results.map(item => [item.ip, item.lastReportTime ?? ''])
    );
    startedBaselines.current = Object.fromEntries(
      snapshot.results.map(item => [item.ip, item.agentStartedAt ?? ''])
    );
    if (deliveredIPs.length === 0) {
      setPushOutcome('failed');
      setPoll({ active: false, nextCheckAt: null, rounds: 0, checking: false });
      return;
    }
    setPushOutcome('pending');
    setPoll({ active: true, nextCheckAt: Date.now() + 3000, rounds: 0, checking: false });
    const startedAt = Date.now();
    const tick = async () => {
      setPoll(prev => ({ ...prev, checking: true, nextCheckAt: null }));
      try {
        const hosts = await fetchHosts();
        const restarted = new Map<string, boolean>();
        const settled = new Map<string, boolean>();
        for (const ip of deliveredIPs) {
          const host = hosts.find(candidate => candidate.ip === ip);
          const baseline = baselines.current[ip] ?? '';
          const baselineStarted = startedBaselines.current[ip] ?? '';
          const lastReport = host?.lastReportTime ?? '';
          const hostStarted = host?.agentStartedAt ?? '';
          const reported =
            baselineStarted !== ''
              ? hostStarted !== '' && hostStarted !== baselineStarted
              : baseline === ''
                ? lastReport !== ''
                : lastReport !== '' && lastReport !== baseline;
          restarted.set(ip, reported);
          settled.set(ip, reported);
        }
        setResults(prev =>
          prev
            ? {
                ...prev,
                results: prev.results.map(item => {
                  const host = hosts.find(candidate => candidate.ip === item.ip);
                  if (!host) return item;
                  return {
                    ...item,
                    agentVersion: host.agentVersion || item.agentVersion,
                    agentStartedAt: host.agentStartedAt ?? item.agentStartedAt,
                    lastReportTime: host.lastReportTime,
                    restarted: restarted.get(item.ip) ?? item.restarted ?? false,
                  };
                }),
              }
            : prev
        );
        if (deliveredIPs.every(ip => settled.get(ip))) {
          setPushOutcome('success');
          setPoll(prev => ({
            active: false,
            nextCheckAt: null,
            rounds: prev.rounds + 1,
            checking: false,
          }));
          return;
        }
      } catch (error) {
        console.warn('刷新推送结果失败', error);
      }
      if (Date.now() - startedAt >= 60000) {
        setPushOutcome('failed');
        setPoll(prev => ({
          active: false,
          nextCheckAt: null,
          rounds: prev.rounds + 1,
          checking: false,
        }));
        return;
      }
      setPoll(prev => ({
        active: true,
        nextCheckAt: Date.now() + 3000,
        rounds: prev.rounds + 1,
        checking: false,
      }));
      refreshTimers.current.push(window.setTimeout(tick, 3000));
    };
    refreshTimers.current.push(window.setTimeout(tick, 3000));
  }, []);

  useEffect(
    () => () => {
      refreshTimers.current.forEach(timer => window.clearTimeout(timer));
    },
    []
  );

  useEffect(() => {
    void load();
  }, [load]);

  usePageRefresh(load);

  const activePlatform = platforms.find(item => item.id === platform) ?? platforms[0];
  const platformSlots = slotsByPlatform[platform];
  const hostedSlots = platformSlots.filter(slot => findSlotMeta(items, platform, slot));
  const hasBinary = hostedSlots.length > 0;
  const standardMeta = findSlotMeta(items, platform, { arch: 'amd64', variant: 'standard' });
  const target = targets?.[platform];
  const noTargets = target?.total === 0;
  const file = files[platform];
  const isUploading = uploading === platform;

  function selectPlatform(next: Platform) {
    if (next === platform) return;
    setPlatform(next);
    setResults(null);
  }

  function pickFile() {
    pickTarget.current = platform;
    fileInputRef.current?.click();
  }

  function handleFileChange(event: ChangeEvent<HTMLInputElement>) {
    const next = event.target.files?.[0] ?? null;
    const targetPlatform = pickTarget.current;
    event.target.value = '';
    if (!next) return;
    setFiles(prev => ({ ...prev, [targetPlatform]: next }));
  }

  function clearFile() {
    setFiles(prev => ({ ...prev, [platform]: null }));
  }

  async function upload() {
    if (!file) {
      toast.error('请选择要上传的二进制文件');
      return;
    }
    setUploading(platform);
    try {
      const uploaded = await uploadAgentBinary(platform, (versions[platform] ?? '').trim(), file);
      const slot = uploaded.variant === 'legacy' ? `${uploaded.arch}/legacy` : uploaded.arch;
      toast.success(`${activePlatform.shortLabel} ${slot} 已上传：v${uploaded.version}`);
      setFiles(prev => ({ ...prev, [platform]: null }));
      setVersions(prev => ({ ...prev, [platform]: '' }));
      await load();
    } catch (error) {
      toast.error(`上传 Agent 二进制失败：${errorMessageOf(error, '未知错误')}`);
    } finally {
      setUploading(null);
    }
  }

  async function push() {
    if (!hasBinary) {
      toast.error('请先上传所选平台的 Agent 二进制');
      return;
    }
    if (noTargets) {
      toast.error('该平台暂无已登记主机，接入 Agent 后再推送');
      return;
    }
    if (!all && !ips.trim()) {
      toast.error('请输入目标主机 IP，或选择全部主机');
      return;
    }
    setPushing(true);
    setResults(null);
    setPushOutcome('idle');
    setPoll({ active: false, nextCheckAt: null, rounds: 0, checking: false });
    try {
      const data = await pushAgentUpdate({ all, ips: ips.trim(), platform });
      setResults(data);
      const delivered = data.results.filter(item => item.delivered).length;
      const total = data.results.length;
      const skipped = data.results.filter(item => item.skipped).length;
      if (delivered === 0 && skipped > 0) {
        toast.warning(`已推送：${skipped} 台主机无匹配安装包，已跳过`);
      } else if (delivered === 0) {
        toast.error(`已推送：${total} 台主机均未下发`);
      } else if (skipped > 0) {
        toast.success(`已推送：已下发 ${delivered} / ${total}，跳过 ${skipped} 台不匹配主机`);
      } else if (delivered === total) {
        toast.success(`已推送：${total} 台主机已下发，等待重启`);
      } else {
        toast.success(`已推送：已下发 ${delivered} / ${total}`);
      }
      refreshTimers.current.forEach(timer => window.clearTimeout(timer));
      refreshTimers.current = [];
      if (delivered === 0 && skipped === total) {
        setPushOutcome('idle');
        setPoll({ active: false, nextCheckAt: null, rounds: 0, checking: false });
      } else {
        setPushOutcome('pending');
        trackPush(data);
      }
      refreshTimers.current.push(window.setTimeout(() => void load(), 6000));
    } catch (error) {
      toast.error(`推送 Agent 更新失败：${errorMessageOf(error, '未知错误')}`);
    } finally {
      setPushing(false);
    }
  }

  async function copySHA(value: string) {
    try {
      await navigator.clipboard.writeText(value);
      toast.success('SHA-256 已复制');
    } catch {
      toast.error('复制失败，请手动选择');
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <span className="grid size-10 shrink-0 place-items-center rounded-xl border border-blue-400/25 bg-blue-500/10 text-[var(--bm-accent-text)]">
            <UploadCloud size={18} />
          </span>
          <div>
            <h2 className="text-base font-semibold text-[var(--bm-text)]">Agent 自动更新</h2>
            <p className="mt-1 max-w-3xl text-xs leading-5 text-[var(--bm-text-muted)]">
              统一托管 Linux / Windows 二进制，上传后可向对应平台在线 Agent 推送；Agent
              会自行下载、校验 SHA-256 并重启
            </p>
          </div>
        </div>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          disabled={loading}
          onClick={() => void refreshList()}
        >
          <RefreshCw size={14} className={loading ? 'animate-spin' : ''} />
          刷新清单
        </Button>
      </div>

      <PlatformTabs
        platform={platform}
        items={items}
        targets={targets}
        disabled={uploading !== null}
        onSelect={selectPlatform}
      />

      <section className="overflow-hidden rounded-xl border border-[var(--bm-border)] bg-[var(--bm-control-bg-soft)]">
        <div className="flex flex-wrap items-start justify-between gap-3 border-b border-[var(--bm-border)]/60 px-4 py-3.5">
          <SectionHeading
            step="01"
            title="上传托管"
            description={
              hasBinary
                ? `服务端按平台校验 ${activePlatform.format} 文件类型，并按 Go 版本自动归档到标准或 legacy 槽位`
                : `尚未托管 ${activePlatform.shortLabel} 二进制；选择文件后上传，服务端会校验 ${activePlatform.format} 并自动识别架构与 Go 版本`
            }
          />
          <span
            className={cn(
              'rounded-full border px-2.5 py-1 text-[11px]',
              hasBinary
                ? 'border-emerald-500/25 bg-emerald-500/10 text-[var(--bm-status-green-text)]'
                : 'border-[var(--bm-border)] text-[var(--bm-text-muted)]'
            )}
          >
            {hasBinary
              ? `已托管 ${hostedSlots.length} / ${platformSlots.length} 个槽位`
              : '尚未托管'}
          </span>
        </div>
        <div className="space-y-3 p-4">
          <HostedSlots platform={platform} items={items} onCopy={value => void copySHA(value)} />

          <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_180px_auto] lg:items-end">
            <div>
              <div className="mb-1 text-[11px] text-[var(--bm-text-muted)]">二进制文件</div>
              <div className="flex min-h-11 items-center gap-3 rounded-lg border border-dashed border-[var(--bm-border)] bg-[var(--bm-control-bg)] px-3 py-2">
                <span className="grid size-8 shrink-0 place-items-center rounded-lg border border-[var(--bm-border)] bg-[var(--bm-control-bg-soft)] text-[var(--bm-accent-text)]">
                  <FileUp size={14} />
                </span>
                <div className="min-w-0 flex-1">
                  <div
                    className={cn(
                      'truncate text-sm',
                      file ? 'text-[var(--bm-text)]' : 'text-[var(--bm-text-muted)]'
                    )}
                  >
                    {file ? file.name : '尚未选择二进制文件'}
                  </div>
                  <div className="mt-0.5 text-[11px] text-[var(--bm-text-muted)]">
                    {file
                      ? `${formatBytes(file.size)} · 待上传`
                      : `支持 ${activePlatform.format} 格式 · ${activePlatform.file}`}
                  </div>
                </div>
                {file ? (
                  <button
                    type="button"
                    aria-label="移除已选文件"
                    onClick={clearFile}
                    className="grid size-6 shrink-0 place-items-center rounded-md text-[var(--bm-text-muted)] transition-colors hover:bg-[var(--bm-control-bg-soft)] hover:text-[var(--bm-text)]"
                  >
                    <X size={13} />
                  </button>
                ) : null}
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="h-9 shrink-0"
                  disabled={!canManage || isUploading}
                  onClick={pickFile}
                >
                  选择文件
                </Button>
              </div>
              <input
                ref={fileInputRef}
                type="file"
                className="hidden"
                disabled={!canManage}
                onChange={handleFileChange}
              />
            </div>
            <label className="block">
              <span className="mb-1 block text-[11px] text-[var(--bm-text-muted)]">
                版本号（可选）
              </span>
              <Input
                value={versions[platform] ?? ''}
                onChange={event =>
                  setVersions(prev => ({ ...prev, [platform]: event.target.value }))
                }
                placeholder="留空自动识别"
                className="h-11 font-mono text-xs"
                disabled={!canManage}
              />
            </label>
            <Button
              type="button"
              className="bm-primary-soft-button h-11 px-5"
              disabled={!canManage || !file || isUploading}
              onClick={() => void upload()}
            >
              {isUploading ? (
                <Loader2 size={15} className="animate-spin" />
              ) : (
                <UploadCloud size={15} />
              )}
              {isUploading ? '上传中…' : '上传托管'}
            </Button>
          </div>
        </div>
      </section>

      <section className="overflow-hidden rounded-xl border border-[var(--bm-border)] bg-[var(--bm-control-bg-soft)]">
        <div className="flex flex-wrap items-start justify-between gap-3 border-b border-[var(--bm-border)]/60 px-4 py-3.5">
          <SectionHeading
            step="02"
            title="推送更新"
            description="按主机架构与 Windows 版本自动选包，无匹配包的主机跳过；也可在采集配置「进阶参数」开启自动更新检查"
          />
          <div className="text-right text-[11px] leading-5 text-[var(--bm-text-muted)]">
            <div className="text-[var(--bm-text)]">{activePlatform.label}</div>
            <div>
              {target ? `登记 ${target.total} 台 · 在线 ${target.online} 台` : '目标统计加载中'}
            </div>
          </div>
        </div>
        <div className="space-y-3 p-4">
          <div className="grid gap-2 sm:grid-cols-2">
            <PushModeCard
              active={all}
              title="全部主机"
              description="向该平台全部已登记主机下发"
              onSelect={() => setAll(true)}
            />
            <PushModeCard
              active={!all}
              title="指定 IP"
              description="按单个 IP、CIDR 网段或 IP 范围筛选"
              onSelect={() => setAll(false)}
            />
          </div>

          {!all ? (
            <div className="rounded-lg border border-[var(--bm-border)] bg-[var(--bm-control-bg)] p-3">
              <textarea
                value={ips}
                onChange={event => setIps(event.target.value)}
                rows={4}
                placeholder={
                  '输入目标主机 IP（必填），逗号、分号或换行分隔，例如：\n10.22.51.47\n10.22.51.0/24\n10.22.51.1-20'
                }
                className="bm-form-control min-h-28 w-full resize-y rounded-md p-2.5 font-mono text-xs leading-6"
              />
              <p className="mt-1.5 text-[11px] leading-5 text-[var(--bm-text-muted)]">
                支持：单个 IP、多个 IP、IP 段（a.b.c.d/24）、IP 范围（a.b.c.1-20 或
                a.b.c.1-a.b.c.20）
              </p>
            </div>
          ) : null}

          {!hasBinary ? (
            <div className="flex items-center gap-2 rounded-lg border border-amber-400/30 bg-amber-500/10 px-3 py-2 text-[11px] text-[var(--bm-status-yellow-text)]">
              <AlertTriangle size={14} className="shrink-0" />
              该平台尚未上传二进制，上传后才能推送
            </div>
          ) : noTargets ? (
            <div className="flex items-center gap-2 rounded-lg border border-amber-400/30 bg-amber-500/10 px-3 py-2 text-[11px] text-[var(--bm-status-yellow-text)]">
              <AlertTriangle size={14} className="shrink-0" />
              该平台暂无已登记主机，接入 Agent 后再推送
            </div>
          ) : null}

          {targets && targets.unknown > 0 ? (
            <div className="flex items-start gap-2 rounded-lg border border-amber-400/30 bg-amber-500/10 px-3 py-2 text-[11px] leading-5 text-[var(--bm-status-yellow-text)]">
              <AlertTriangle size={14} className="mt-0.5 shrink-0" />
              另有 {targets.unknown} 台主机平台未知（操作系统未上报或无法识别），不参与按平台推送
            </div>
          ) : null}

          {platform === 'windows' ? <WindowsCompatHint standard={standardMeta} /> : null}

          <div className="flex flex-wrap items-center justify-between gap-3 border-t border-[var(--bm-border)]/60 pt-3">
            <p className="text-[11px] leading-5 text-[var(--bm-text-muted)]">
              离线主机不会收到推送，上线后会在一轮自动更新检查内补齐
            </p>
            <Button
              type="button"
              className="bm-success-soft-button h-10 px-5"
              disabled={!canManage || pushing || !hasBinary || noTargets}
              onClick={() => void push()}
            >
              {pushing ? <Loader2 size={15} className="animate-spin" /> : <Rocket size={15} />}
              {pushing ? '推送中…' : '推送更新'}
            </Button>
          </div>

          {results ? (
            <PushResultsTable
              platform={results.platform}
              results={results.results}
              outcome={pushOutcome}
              poll={poll}
            />
          ) : null}
        </div>
      </section>

      <div className="flex items-start gap-2.5 rounded-xl border border-[var(--bm-border)]/60 bg-[var(--bm-control-bg-soft)] px-3.5 py-3 text-[11px] leading-5 text-[var(--bm-text-muted)]">
        <Info size={14} className="mt-0.5 shrink-0 text-[var(--bm-accent-text)]" />
        <div className="space-y-1">
          <p>
            推送后 Agent 自行下载并校验 SHA-256，替换二进制（同目录保留{' '}
            <code className="rounded bg-[var(--bm-control-bg)] px-1 py-0.5 font-mono text-[10px] text-[var(--bm-text)]">
              .bak
            </code>
            ）后自动重启。自动更新仅接受比当前更高的版本；手动推送不做版本比较，会直接替换为所选版本。
            新主机首次需手动部署 Agent，之后才可接收手动推送或自动更新。
          </p>
          <p>
            下载地址{' '}
            <code className="rounded bg-[var(--bm-control-bg)] px-1 py-0.5 font-mono text-[10px] text-[var(--bm-text)]">
              /api/agent/download?platform=&arch=&osBuild=
            </code>
            ，需携带{' '}
            <code className="rounded bg-[var(--bm-control-bg)] px-1 py-0.5 font-mono text-[10px] text-[var(--bm-text)]">
              X-Agent-Token
            </code>
            。
          </p>
        </div>
      </div>
    </div>
  );
}
