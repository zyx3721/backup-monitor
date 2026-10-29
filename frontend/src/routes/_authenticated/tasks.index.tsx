import { createFileRoute } from '@tanstack/react-router';
import { TasksPage } from '@/features/backup/TasksPage';

export const Route = createFileRoute('/_authenticated/tasks/')({
  component: TasksPage,
});
