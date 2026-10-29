import { createFileRoute } from '@tanstack/react-router';
import { AgentManagementPage } from '@/features/agents/AgentManagementPage';

export const Route = createFileRoute('/_authenticated/agents')({ component: AgentManagementPage });
