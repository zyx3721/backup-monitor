import { createFileRoute } from '@tanstack/react-router';
import { TaskDetailPage } from '@/features/backup/TaskDetailPage';

export const Route = createFileRoute('/_authenticated/tasks/$taskID')({
  component: TaskDetailRoute,
});

function TaskDetailRoute() {
  const { taskID } = Route.useParams();
  return <TaskDetailPage taskID={taskID} />;
}
