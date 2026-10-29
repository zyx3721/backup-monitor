import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';

export type HostRefreshGroup = {
  intervalSec: number;
  hostIds: string[];
};

export type HostRefreshPlan = {
  defaultIntervalSec: number;
  minIntervalSec: number;
  groups: HostRefreshGroup[];
};

export const hostRefreshPlanQueryKey = ['backup', 'hosts', 'refresh-plan'] as const;

export async function fetchHostRefreshPlan() {
  return api<HostRefreshPlan>('/api/backup/hosts/refresh-plan');
}

// useHostRefreshPlan 读取服务端按上报间隔分组的主机刷新计划
export function useHostRefreshPlan(enabled = true) {
  return useQuery({
    queryKey: hostRefreshPlanQueryKey,
    queryFn: fetchHostRefreshPlan,
    enabled,
    staleTime: 60_000,
    refetchInterval: 300_000,
  });
}
