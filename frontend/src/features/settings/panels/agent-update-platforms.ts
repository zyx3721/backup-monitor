import { Monitor, Terminal } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import type { AgentBinaryMeta } from '@/features/backup/types';

export type Platform = 'linux' | 'windows';

export type BinarySlot = {
  arch: 'amd64' | 'arm64';
  variant: 'standard' | 'legacy';
};

type PlatformConfig = {
  id: Platform;
  label: string;
  shortLabel: string;
  file: string;
  format: string;
  icon: LucideIcon;
};

export const platforms: PlatformConfig[] = [
  {
    id: 'linux',
    label: 'Linux',
    shortLabel: 'Linux',
    file: 'backup-agent-linux',
    format: 'ELF',
    icon: Terminal,
  },
  {
    id: 'windows',
    label: 'Windows',
    shortLabel: 'Windows',
    file: 'backup-agent-windows.exe',
    format: 'PE（.exe）',
    icon: Monitor,
  },
];

export function platformLabel(platform: Platform) {
  return platforms.find(item => item.id === platform)?.label ?? platform;
}

// slotsByPlatform 各平台可托管的槽位：平台 + 架构 + 通道
export const slotsByPlatform: Record<Platform, BinarySlot[]> = {
  linux: [
    { arch: 'amd64', variant: 'standard' },
    { arch: 'arm64', variant: 'standard' },
  ],
  windows: [
    { arch: 'amd64', variant: 'standard' },
    { arch: 'amd64', variant: 'legacy' },
    { arch: 'arm64', variant: 'standard' },
  ],
};

// slotLabel 槽位展示名
export function slotLabel(slot: BinarySlot): string {
  return slot.variant === 'legacy' ? `${slot.arch} legacy` : slot.arch;
}

// findSlotMeta 返回槽位已托管的元数据，未上传时为 undefined
export function findSlotMeta(items: AgentBinaryMeta[], platform: Platform, slot: BinarySlot) {
  return items.find(
    item => item.platform === platform && item.arch === slot.arch && item.variant === slot.variant
  );
}

// windowsRequirementLabel 根据安装包的 Go 版本推导最低 Windows 版本
export function windowsRequirementLabel(goVersion: string): string {
  const match = /^go(\d+)\.(\d+)/.exec((goVersion || '').trim());
  if (!match) {
    return '未知（推送时无法校验系统版本）';
  }
  const major = Number(match[1]);
  const minor = Number(match[2]);
  if (major > 1 || (major === 1 && minor >= 21)) {
    return 'Windows 10 / Server 2016 及以上';
  }
  return 'Windows 7 / Server 2008 R2 及以上';
}
