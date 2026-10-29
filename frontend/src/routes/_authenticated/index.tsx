import { createFileRoute } from '@tanstack/react-router';
import { DashboardPage } from '@/features/backup/DashboardPage';

export const Route = createFileRoute('/_authenticated/')({ component: DashboardPage });
