import type { BackupExecution } from './types';

export function executionDuration(execution: BackupExecution) {
  if (!execution.endTime) return '-';
  const start = new Date(execution.startTime || execution.creationTime).getTime();
  const end = new Date(execution.endTime).getTime();
  if (Number.isNaN(start) || Number.isNaN(end) || end < start) return '-';
  const seconds = Math.round((end - start) / 1000);
  if (seconds < 60) return `${seconds} 秒`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} 分 ${seconds % 60} 秒`;
  return `${Math.floor(minutes / 60)} 时 ${minutes % 60} 分`;
}

export function sizeLabel(value?: string) {
  const text = value?.trim();
  if (!text || text === '-' || text === '--') return '-';
  const match = text.match(/^([\d.]+)\s*(?:[KMGT]?B)?$/i);
  if (match && Number(match[1]) === 0) return '-';
  return text;
}

export function formatExecutionID(value?: string) {
  const text = value?.trim();
  if (!text) return '-';
  return text.startsWith('{') && text.endsWith('}') ? text : `{${text}}`;
}
