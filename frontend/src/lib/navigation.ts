import {
  Bot,
  CircleGauge,
  Folder,
  History,
  Monitor,
  ScrollText,
  Send,
  Settings,
  ShieldAlert,
  type LucideIcon,
} from 'lucide-react';
import { PERM, SETTINGS_READ_PERMISSIONS, userHasAnyPermission } from '@/lib/permissions';
import type { AuthUser } from '@/lib/session';

export type NavItem = {
  to: string;
  label: string;
  description: string;
  permissions: string[];
  icon: LucideIcon;
};

export const navigation: NavItem[] = [
  {
    to: '/',
    label: '概览',
    description: '备份监控运行态势',
    permissions: [PERM.dashboardRead],
    icon: CircleGauge,
  },
  {
    to: '/hosts',
    label: '备份主机',
    description: '主机清单与在线状态',
    permissions: [PERM.hostsRead],
    icon: Monitor,
  },
  {
    to: '/tasks',
    label: '备份任务',
    description: '任务、计划与健康状态',
    permissions: [PERM.tasksRead],
    icon: Folder,
  },
  {
    to: '/executions',
    label: '执行记录',
    description: '执行结果、体积与产物',
    permissions: [PERM.executionsRead],
    icon: History,
  },
  {
    to: '/alerts',
    label: '告警中心',
    description: '备份失败、未执行与主机异常',
    permissions: [PERM.alertsRead],
    icon: ShieldAlert,
  },
  {
    to: '/history',
    label: '审计日志',
    description: '追踪身份认证、资源与配置安全事件',
    permissions: [PERM.auditRead],
    icon: ScrollText,
  },
  {
    to: '/notifications',
    label: '通知管理',
    description: '通知渠道与发送记录',
    permissions: [PERM.notificationsRead],
    icon: Send,
  },
  {
    to: '/agents',
    label: 'Agent 管理',
    description: '采集配置、访问令牌与版本更新',
    permissions: [PERM.agentsRead],
    icon: Bot,
  },
  {
    to: '/settings',
    label: '系统配置',
    description: '基础参数、用户、认证与找密配置',
    permissions: SETTINGS_READ_PERMISSIONS,
    icon: Settings,
  },
];

export function matchesPath(to: string, path: string) {
  return to === '/' ? path === '/' : path === to || path.startsWith(`${to}/`);
}

function isNavigationAllowed(user: AuthUser | null, item: NavItem) {
  return item.permissions.length === 0 || userHasAnyPermission(user, item.permissions);
}

// visibleNavigation 返回当前账号有权访问的侧边栏导航项
export function visibleNavigation(user: AuthUser | null) {
  return navigation.filter(item => isNavigationAllowed(user, item));
}

// firstAccessiblePath 返回概览之外首个可访问集合的路径，用于缺少概览权限时的落地跳转
export function firstAccessiblePath(user: AuthUser | null) {
  return navigation.find(item => item.to !== '/' && isNavigationAllowed(user, item))?.to ?? null;
}
