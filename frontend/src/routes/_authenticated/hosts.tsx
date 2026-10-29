import { createFileRoute, Outlet } from '@tanstack/react-router';

export const Route = createFileRoute('/_authenticated/hosts')({
  component: HostsLayout,
});

function HostsLayout() {
  return <Outlet />;
}
