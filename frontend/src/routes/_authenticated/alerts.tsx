import { createFileRoute } from '@tanstack/react-router';
import { AlertsPage } from '@/features/backup/AlertsPage';

export const Route = createFileRoute('/_authenticated/alerts')({ component: AlertsPage });
