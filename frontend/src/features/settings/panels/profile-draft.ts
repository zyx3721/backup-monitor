import type { AgentProfileConfig } from '@/features/backup/types';

export type ProfileTab = 'basic' | 'collect' | 'filter';

export type ProfileDraft = {
  id?: string;
  name: string;
  hostname: string;
  description: string;
  enabled: boolean;
  priority: number;
  config: AgentProfileConfig;
};

// defaultConfig 与后端 domain.DefaultProfileConfig 保持一致，避免新建配置时收窄采集范围
const defaultConfig: AgentProfileConfig = {
  reportIntervalSec: 0,
  collectCrontab: true,
  collectVeeam: true,
  artifactCheck: true,
  labelPrefix: '# backup-monitor:',
  includeKeywords: [
    'backup',
    'dump',
    'mysqldump',
    'xtrabackup',
    'pg_dump',
    'rsync',
    'rclone',
    'borg',
    'tar',
    'veeam',
  ],
  includePaths: ['/data/scripts', '/opt/backup', '/usr/local/bin'],
  excludeKeywords: ['logrotate', 'ntpdate', 'certbot', 'apt-get', 'yum ', '/usr/lib'],
  maxArtifactAgeHours: 30,
  heartbeatIntervalSec: 30,
  artifactMaxDepth: 3,
  maxSessionsPerJob: 5,
  maxSessionInfoCalls: 10,
  maxJobInfoCalls: 20,
  manualTriggerMinutes: 30,
  artifactPreToleranceMinutes: 10,
  autoUpdateMinutes: 0,
};

export function emptyDraft(): ProfileDraft {
  return {
    name: '',
    hostname: '*',
    description: '',
    enabled: true,
    priority: 0,
    config: {
      ...defaultConfig,
      includeKeywords: [...defaultConfig.includeKeywords],
      includePaths: [...defaultConfig.includePaths],
      excludeKeywords: [...defaultConfig.excludeKeywords],
    },
  };
}
