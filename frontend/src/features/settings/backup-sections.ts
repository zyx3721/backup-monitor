import { RadioTower, ShieldAlert, TimerReset } from 'lucide-react';

export type BackupSection = 'thresholds' | 'agent' | 'alarm';

export const backupSections: Array<{
  id: BackupSection;
  title: string;
  description: string;
  icon: typeof TimerReset;
  color: string;
}> = [
  {
    id: 'thresholds',
    title: '判定与保留',
    description: '主机超时、任务漏执行、记录与日志保留策略',
    icon: TimerReset,
    color: '#f59e0b',
  },
  {
    id: 'agent',
    title: 'Agent 通道',
    description: '下发给 Agent 的默认上报间隔、超时与重连退避',
    icon: RadioTower,
    color: '#a78bfa',
  },
  {
    id: 'alarm',
    title: '安全与告警',
    description: '告警去重与 Agent 令牌',
    icon: ShieldAlert,
    color: '#fb7185',
  },
];
