import { createFileRoute } from '@tanstack/react-router';
import { HostsPage } from '@/features/backup/HostsPage';

export const Route = createFileRoute('/_authenticated/hosts/')({
  component: HostsPage,
});
