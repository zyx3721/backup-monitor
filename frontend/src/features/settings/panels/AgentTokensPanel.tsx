import { Copy, Pause, Play, Plus, Search, Trash2 } from 'lucide-react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { ConfirmDialog } from '@/components/dialogs/confirm-dialog';
import { DataTableShell } from '@/components/data/data-table-shell';
import { Pagination } from '@/components/data/pagination';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { errorMessageOf, showErrorToast } from '@/lib/toast-errors';
import { usePageRefresh } from '@/lib/page-refresh';
import { createToken, deleteToken, fetchTokens, updateToken } from '@/features/backup/api';
import type { AgentToken } from '@/features/backup/types';
import { EmptyState, formatTime, Tag } from '@/features/backup/components';

export function AgentTokensPanel({ canManage }: { canManage: boolean }) {
  const [tokens, setTokens] = useState<AgentToken[]>([]);
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(18);
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState('');
  const [hostname, setHostname] = useState('');
  const [manualToken, setManualToken] = useState('');
  const [busy, setBusy] = useState(false);
  const [removing, setRemoving] = useState<AgentToken | null>(null);

  const load = useCallback(
    () =>
      fetchTokens()
        .then(setTokens)
        .catch(error => showErrorToast(error, '读取令牌失败')),
    []
  );

  useEffect(() => {
    void load();
  }, [load]);

  usePageRefresh(load);

  const filtered = useMemo(() => {
    const query = keyword.trim().toLowerCase();
    if (!query) return tokens;
    return tokens.filter(token =>
      `${token.name} ${token.token} ${token.hostname || '不限'}`.toLowerCase().includes(query)
    );
  }, [keyword, tokens]);
  const rowCount = filtered.length;
  const effectivePageSize = pageSize === -1 ? Math.max(rowCount, 1) : pageSize;
  const pageCount = Math.max(1, Math.ceil(rowCount / effectivePageSize));
  const currentPage = Math.min(page, pageCount);
  const pageStart = (currentPage - 1) * effectivePageSize;
  const pageTokens = filtered.slice(pageStart, pageStart + effectivePageSize);

  useEffect(() => setPage(1), [keyword, pageSize]);
  useEffect(() => {
    if (page > pageCount) setPage(pageCount);
  }, [page, pageCount]);

  function openCreate() {
    setName('');
    setHostname('');
    setManualToken('');
    setCreating(true);
  }

  async function create() {
    if (!name.trim()) {
      toast.error('令牌名称不能为空');
      return;
    }
    const custom = manualToken.trim();
    setBusy(true);
    try {
      await createToken({
        name: name.trim(),
        hostname: hostname.trim(),
        ...(custom ? { token: custom } : {}),
      });
      toast.success(`Agent 令牌 ${name.trim()} 已创建，请立即复制保存`);
      setCreating(false);
      await load();
    } catch (error) {
      toast.error(`Agent 令牌 ${name.trim()} 创建失败：${errorMessageOf(error, '未知错误')}`);
    } finally {
      setBusy(false);
    }
  }

  async function toggle(token: AgentToken) {
    try {
      await updateToken(token.id, {
        name: token.name,
        hostname: token.hostname,
        enabled: !token.enabled,
      });
      toast.success(`Agent 令牌 ${token.name} ${token.enabled ? '已停用' : '已启用'}`);
      await load();
    } catch (error) {
      toast.error(
        `Agent 令牌 ${token.name} ${token.enabled ? '停用' : '启用'}失败：${errorMessageOf(error, '未知错误')}`
      );
    }
  }

  async function remove() {
    if (!removing) return;
    setBusy(true);
    try {
      await deleteToken(removing.id);
      toast.success(`Agent 令牌 ${removing.name} 已删除`);
      setRemoving(null);
      await load();
    } catch (error) {
      toast.error(`Agent 令牌 ${removing.name} 删除失败：${errorMessageOf(error, '未知错误')}`);
    } finally {
      setBusy(false);
    }
  }

  async function copy(token: string) {
    try {
      await navigator.clipboard.writeText(token);
      toast.success('令牌已复制');
    } catch {
      toast.error('复制失败，请手动选择文本');
    }
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <DataTableShell
        className="bm-page-card"
        contentClassName="bm-scroll-area flex-1"
        toolbarClassName="bm-hidden-scrollbar flex-nowrap overflow-x-auto"
        footer={
          <Pagination
            page={currentPage}
            pageCount={pageCount}
            pageSize={pageSize}
            total={rowCount}
            onPageChange={setPage}
            onPageSizeChange={setPageSize}
          />
        }
        toolbar={
          <>
            <div className="relative min-w-56 flex-1 sm:max-w-72">
              <Search
                size={15}
                className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--bm-text-muted)]"
              />
              <Input
                value={keyword}
                onChange={event => setKeyword(event.target.value)}
                placeholder="搜索名称、令牌、绑定主机"
                className="pl-9"
              />
            </div>
            <span className="ml-auto text-xs text-[var(--bm-text-muted)]">共 {rowCount} 个</span>
            {canManage ? (
              <Button className="bm-primary-soft-button shrink-0" onClick={openCreate}>
                <Plus size={16} />
                新建令牌
              </Button>
            ) : null}
          </>
        }
      >
        <div className="flex min-h-full flex-col">
          <table className="w-full min-w-[1000px] text-left text-sm">
            <thead className="sticky top-0 z-10 bg-[var(--bm-table-head-bg)] text-center text-xs font-semibold text-[var(--bm-text-muted)]">
              <tr>
                <th className="w-[80px] whitespace-nowrap px-4 py-3 font-semibold">序号</th>
                <th className="w-[200px] whitespace-nowrap px-4 py-3 font-semibold">名称</th>
                <th className="px-4 py-3 font-semibold">令牌</th>
                <th className="w-[200px] whitespace-nowrap px-4 py-3 font-semibold">绑定主机</th>
                <th className="w-[180px] whitespace-nowrap px-4 py-3 font-semibold">最后使用</th>
                <th className="w-[100px] whitespace-nowrap px-4 py-3 font-semibold">状态</th>
                {canManage ? (
                  <th className="w-[180px] whitespace-nowrap px-4 py-3 font-semibold">操作</th>
                ) : null}
              </tr>
            </thead>
            <tbody>
              {pageTokens.map((token, index) => (
                <tr
                  key={token.id}
                  className="border-t border-[var(--bm-border)]/60 bm-table-row-hover"
                >
                  <td className="px-4 py-3 text-center text-sm text-[var(--bm-text)]">
                    {pageStart + index + 1}
                  </td>
                  <td className="px-4 py-3 text-center font-medium text-[var(--bm-text)]">
                    {token.name}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center justify-center gap-2">
                      <code className="bm-token-code rounded px-2 py-0.5 font-mono text-sm text-[var(--bm-text)]">
                        {token.token}
                      </code>
                      <button
                        type="button"
                        className="text-sky-300 hover:text-sky-200"
                        title="复制令牌"
                        onClick={() => void copy(token.token)}
                      >
                        <Copy size={14} />
                      </button>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-center text-sm text-[var(--bm-text)]">
                    {token.hostname || '不限'}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-center text-sm text-[var(--bm-text)]">
                    {token.lastSeenAt ? formatTime(token.lastSeenAt) : '-'}
                  </td>
                  <td className="px-4 py-3 text-center">
                    <Tag tone={token.enabled ? 'success' : 'warning'}>
                      {token.enabled ? '启用' : '停用'}
                    </Tag>
                  </td>
                  {canManage ? (
                    <td className="px-4 py-3">
                      <div className="flex justify-center gap-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          className={`${token.enabled ? 'bm-row-action-warning' : 'bm-row-action-edit'} rounded-lg`}
                          onClick={() => void toggle(token)}
                        >
                          {token.enabled ? <Pause size={14} /> : <Play size={14} />}
                          {token.enabled ? '停用' : '启用'}
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="bm-row-action-delete rounded-lg"
                          onClick={() => setRemoving(token)}
                        >
                          <Trash2 size={14} /> 删除
                        </Button>
                      </div>
                    </td>
                  ) : null}
                </tr>
              ))}
            </tbody>
          </table>
          {pageTokens.length === 0 ? (
            <EmptyState className="flex-1">
              {tokens.length === 0 ? '暂无令牌' : '没有匹配的令牌'}
            </EmptyState>
          ) : null}
        </div>
      </DataTableShell>

      <Dialog open={creating} onOpenChange={open => !open && !busy && setCreating(false)}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>新建 Agent 令牌</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <label className="block">
              <span className="mb-1.5 block text-xs text-[var(--bm-text-muted)]">
                <span className="mr-1 text-red-400">*</span>令牌名称
              </span>
              <Input
                value={name}
                onChange={event => setName(event.target.value)}
                placeholder="如：生产环境"
              />
            </label>
            <label className="block">
              <span className="mb-1.5 block text-xs text-[var(--bm-text-muted)]">
                绑定主机（可选）
              </span>
              <Input
                value={hostname}
                onChange={event => setHostname(event.target.value)}
                placeholder="留空表示不限制"
                className="font-mono"
              />
            </label>
            <label className="block">
              <span className="mb-1.5 block text-xs text-[var(--bm-text-muted)]">
                自定义令牌（可选）
              </span>
              <Input
                value={manualToken}
                onChange={event => setManualToken(event.target.value)}
                placeholder="留空自动生成 48 位随机令牌"
                className="font-mono"
              />
              <span className="mt-1 block text-[11px] leading-5 text-[var(--bm-text-muted)]">
                至少 16 位，仅限字母、数字与常见符号，不允许中文和空格
              </span>
            </label>
          </div>
          <DialogFooter>
            <Button variant="outline" disabled={busy} onClick={() => setCreating(false)}>
              取消
            </Button>
            <Button
              className="bm-primary-soft-button"
              disabled={busy}
              onClick={() => void create()}
            >
              {busy ? '创建中…' : '创建'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={Boolean(removing)}
        title="删除 Agent 令牌"
        description="删除后不可恢复，使用该令牌的 Agent 将立即无法上报"
        detail={`确认删除令牌「${removing?.name ?? ''}」吗？`}
        horizontalHeader
        busy={busy}
        onOpenChange={open => !open && setRemoving(null)}
        onConfirm={() => void remove()}
      />
    </div>
  );
}
