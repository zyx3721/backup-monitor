import { createFileRoute } from '@tanstack/react-router';
import { ExecutionsPage } from '@/features/backup/ExecutionsPage';

export const Route = createFileRoute('/_authenticated/executions')({ component: ExecutionsPage });
