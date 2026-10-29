import { createFileRoute, Outlet } from '@tanstack/react-router';

export const Route = createFileRoute('/_authenticated/tasks')({
  validateSearch: (search: Record<string, unknown>): { status?: string; open?: string } => ({
    ...(typeof search.status === 'string' ? { status: search.status } : {}),
    ...(typeof search.open === 'string' ? { open: search.open } : {}),
  }),
  component: TasksLayout,
});

function TasksLayout() {
  return <Outlet />;
}
