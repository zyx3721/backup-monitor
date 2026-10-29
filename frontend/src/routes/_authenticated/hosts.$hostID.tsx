import { createFileRoute } from '@tanstack/react-router';
import { HostDetailPage } from '@/features/backup/HostDetailPage';

export const Route = createFileRoute('/_authenticated/hosts/$hostID')({
  component: HostDetailRoute,
});

function HostDetailRoute() {
  const { hostID } = Route.useParams();
  return <HostDetailPage hostID={hostID} />;
}
