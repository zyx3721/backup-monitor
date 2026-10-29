import { createFileRoute } from '@tanstack/react-router';
import { AppLayout, RequireAuth } from '@/components/layout/app-layout';

export const Route = createFileRoute('/_authenticated')({
  component: AuthenticatedShell,
});

function AuthenticatedShell() {
  return (
    <RequireAuth>
      <AppLayout />
    </RequireAuth>
  );
}
