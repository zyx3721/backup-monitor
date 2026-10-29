import { createFileRoute } from '@tanstack/react-router';
import { NotificationsPage } from '@/features/backup/NotificationsPage';

export const Route = createFileRoute('/_authenticated/notifications')({
  component: NotificationsPage,
});
