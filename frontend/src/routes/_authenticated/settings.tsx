import { createFileRoute } from '@tanstack/react-router';
import { SystemSettingsPage } from '@/features/settings/SystemSettingsPage';

export const Route = createFileRoute('/_authenticated/settings')({ component: SystemSettingsPage });
