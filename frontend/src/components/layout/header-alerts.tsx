import { useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { Bell } from 'lucide-react';
import { AppTooltip } from '@/components/feedback/app-tooltip';
import { fetchAlerts } from '@/features/backup/api';
import { PERM, userHasPermission } from '@/lib/permissions';
import { getStoredUser } from '@/lib/session';

export function HeaderAlerts() {
  const navigate = useNavigate();
  const allowed = userHasPermission(getStoredUser(), PERM.alertsRead);
  const { data: alerts = [] } = useQuery({
    queryKey: ['backup', 'alerts'],
    queryFn: fetchAlerts,
    staleTime: 30_000,
    retry: false,
    enabled: allowed,
  });
  if (!allowed) return null;
  const count = alerts.filter(alert => alert.status === 'active').length;
  const label = count > 0 ? `未处理告警 ${count} 条，点击查看` : '暂无未处理告警';

  return (
    <AppTooltip label={label} placement="bottom">
      <button
        type="button"
        onClick={() => void navigate({ to: '/alerts' })}
        className="bm-action-button relative flex h-[42px] w-[42px] items-center justify-center rounded-lg border"
        aria-label={label}
      >
        <Bell size={17} />
        {count > 0 ? (
          <span className="absolute -right-1.5 -top-1.5 grid min-w-4 h-4 place-items-center rounded-full bg-red-500 px-1 text-[10px] font-semibold leading-none text-white">
            {count > 99 ? '99+' : count}
          </span>
        ) : null}
      </button>
    </AppTooltip>
  );
}
