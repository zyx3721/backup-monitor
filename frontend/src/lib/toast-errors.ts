import { toast } from 'sonner';
import { ApiError } from '@/lib/api';
import { isAuthRedirecting } from '@/lib/session';

// errorMessageOf 提取错误文本的首行，用于拼装带动作前缀的失败提示
export function errorMessageOf(error: unknown, fallback = '操作失败') {
  const message =
    typeof error === 'string' && error.trim()
      ? error
      : error instanceof Error && error.message
        ? error.message
        : fallback;
  const lines = String(message ?? '')
    .split('\n')
    .map(line => line.trim())
    .filter(Boolean);
  return lines[0] ?? fallback;
}

// showErrorToast 展示后端错误消息，多行时拆成多条提示；401 由全局跳转登录页，这里不重复弹出
export function showErrorToast(error: unknown, fallback = '操作失败') {
  if (error instanceof ApiError && error.status === 401 && isAuthRedirecting()) return;
  const message =
    typeof error === 'string' && error.trim()
      ? error
      : error instanceof Error && error.message
        ? error.message
        : fallback;
  const lines = String(message ?? '')
    .split('\n')
    .map(line => line.trim())
    .filter(Boolean);
  if (lines.length <= 1) {
    toast.error(lines[0] ?? fallback);
    return;
  }
  lines.forEach(line => toast.error(line));
}
