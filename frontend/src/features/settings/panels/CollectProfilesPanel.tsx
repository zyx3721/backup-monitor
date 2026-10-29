import { Pencil, Plus, Search, Trash2 } from 'lucide-react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { ConfirmDialog } from '@/components/dialogs/confirm-dialog';
import { DataTableShell } from '@/components/data/data-table-shell';
import { Pagination } from '@/components/data/pagination';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { errorMessageOf, showErrorToast } from '@/lib/toast-errors';
import { ApiError } from '@/lib/api';
import { usePageRefresh } from '@/lib/page-refresh';
import { createProfile, deleteProfile, fetchProfiles, updateProfile } from '@/features/backup/api';
import type { AgentProfile } from '@/features/backup/types';
import { EmptyState, formatTime, Tag } from '@/features/backup/components';
import { CollectProfileDialog } from './CollectProfileDialog';
import { emptyDraft, type ProfileDraft, type ProfileTab } from './profile-draft';

export function CollectProfilesPanel({ canManage = true }: { canManage?: boolean }) {
  const [profiles, setProfiles] = useState<AgentProfile[]>([]);
  const [keyword, setKeyword] = useState('');
  const [draft, setDraft] = useState<ProfileDraft | null>(null);
  const [tab, setTab] = useState<ProfileTab>('basic');
  const [busy, setBusy] = useState(false);
  const [removing, setRemoving] = useState<AgentProfile | null>(null);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(18);

  const load = useCallback(
    () =>
      fetchProfiles()
        .then(setProfiles)
        .catch(error => showErrorToast(error, '读取采集配置失败')),
    []
  );

  useEffect(() => {
    void load();
  }, [load]);

  usePageRefresh(load);

  const filtered = useMemo(() => {
    const query = keyword.trim().toLowerCase();
    if (!query) return profiles;
    return profiles.filter(item =>
      `${item.name} ${item.hostname} ${item.description ?? ''}`.toLowerCase().includes(query)
    );
  }, [keyword, profiles]);
  const rowCount = filtered.length;
  const effectivePageSize = pageSize === -1 ? Math.max(rowCount, 1) : pageSize;
  const pageCount = Math.max(1, Math.ceil(rowCount / effectivePageSize));
  const currentPage = Math.min(page, pageCount);
  const pageStart = (currentPage - 1) * effectivePageSize;
  const pageProfiles = filtered.slice(pageStart, pageStart + effectivePageSize);

  useEffect(() => setPage(1), [keyword, pageSize]);
  useEffect(() => {
    if (page > pageCount) setPage(pageCount);
  }, [page, pageCount]);

  function openCreate() {
    setDraft(emptyDraft());
    setTab('basic');
  }

  function openEdit(profile: AgentProfile) {
    setDraft({
      id: profile.id,
      name: profile.name,
      hostname: profile.hostname,
      description: profile.description ?? '',
      enabled: profile.enabled,
      priority: profile.priority,
      config: { ...profile.config },
    });
    setTab('basic');
  }

  async function save() {
    if (!draft) return;
    if (!draft.name.trim()) {
      toast.error('配置名称不能为空');
      setTab('basic');
      return;
    }
    if (!draft.hostname.trim()) {
      toast.error('匹配主机不能为空');
      setTab('basic');
      return;
    }
    const payload = {
      ...draft,
      config: {
        ...draft.config,
        reportIntervalSec:
          draft.config.reportIntervalSec > 0 && draft.config.reportIntervalSec < 10
            ? 10
            : draft.config.reportIntervalSec,
        includeKeywords: draft.config.includeKeywords.filter(Boolean),
        includePaths: draft.config.includePaths.filter(Boolean),
        excludeKeywords: draft.config.excludeKeywords.filter(Boolean),
      },
    };
    setBusy(true);
    try {
      if (draft.id) await updateProfile(draft.id, payload);
      else await createProfile(payload);
      toast.success(`采集配置 ${draft.name} ${draft.id ? '已更新' : '已创建'}`);
      setDraft(null);
      await load();
    } catch (error) {
      if (error instanceof ApiError && error.status === 409) {
        toast.error(`采集配置 ${draft.name} 已存在`);
        return;
      }
      toast.error(
        `采集配置 ${draft.name} ${draft.id ? '更新' : '创建'}失败：${errorMessageOf(error, '未知错误')}`
      );
    } finally {
      setBusy(false);
    }
  }

  async function remove() {
    if (!removing) return;
    setBusy(true);
    try {
      await deleteProfile(removing.id);
      toast.success(`采集配置 ${removing.name} 已删除`);
      setRemoving(null);
      await load();
    } catch (error) {
      toast.error(`采集配置 ${removing.name} 删除失败：${errorMessageOf(error, '未知错误')}`);
    } finally {
      setBusy(false);
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
                placeholder="搜索配置名称、匹配主机、描述"
                className="pl-9"
              />
            </div>
            <span className="ml-auto text-xs text-[var(--bm-text-muted)]">共 {rowCount} 条</span>
            {canManage ? (
              <Button className="bm-primary-soft-button shrink-0" onClick={openCreate}>
                <Plus size={16} />
                新建配置
              </Button>
            ) : null}
          </>
        }
      >
        <div className="flex min-h-full flex-col">
          <table className="w-full min-w-[900px] text-left text-sm">
            <thead className="sticky top-0 z-10 bg-[var(--bm-table-head-bg)] text-center text-xs font-semibold text-[var(--bm-text-muted)]">
              <tr>
                <th className="w-[64px] whitespace-nowrap px-4 py-3 font-semibold">序号</th>
                <th className="w-[240px] px-4 py-3 font-semibold">配置名称</th>
                <th className="w-[140px] whitespace-nowrap px-4 py-3 font-semibold">匹配主机</th>
                <th className="w-[96px] whitespace-nowrap px-4 py-3 font-semibold">优先级</th>
                <th className="w-[112px] whitespace-nowrap px-4 py-3 font-semibold">上报间隔</th>
                <th className="w-[96px] whitespace-nowrap px-4 py-3 font-semibold">状态</th>
                <th className="w-[176px] whitespace-nowrap px-4 py-3 font-semibold">更新时间</th>
                {canManage ? (
                  <th className="w-[176px] whitespace-nowrap px-4 py-3 font-semibold">操作</th>
                ) : null}
              </tr>
            </thead>
            <tbody>
              {pageProfiles.map((profile, index) => (
                <tr
                  key={profile.id}
                  className="border-t border-[var(--bm-border)]/60 bm-table-row-hover"
                >
                  <td className="px-4 py-3 text-center text-sm text-[var(--bm-text)]">
                    {pageStart + index + 1}
                  </td>
                  <td className="px-4 py-3 text-center font-medium text-[var(--bm-text)]">
                    {profile.name}
                  </td>
                  <td className="px-4 py-3 text-center font-mono text-sm text-[var(--bm-text)]">
                    {profile.hostname}
                  </td>
                  <td className="px-4 py-3 text-center text-sm text-[var(--bm-text)]">
                    {profile.priority}
                  </td>
                  <td className="px-4 py-3 text-center text-sm text-[var(--bm-text)]">
                    {profile.config.reportIntervalSec
                      ? `${profile.config.reportIntervalSec}s`
                      : '默认'}
                  </td>
                  <td className="px-4 py-3 text-center">
                    <Tag tone={profile.enabled ? 'success' : 'warning'}>
                      {profile.enabled ? '启用' : '停用'}
                    </Tag>
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-center text-sm text-[var(--bm-text)]">
                    {formatTime(profile.updatedAt)}
                  </td>
                  {canManage ? (
                    <td className="px-4 py-3">
                      <div className="flex justify-center gap-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          className="bm-row-action-edit rounded-lg"
                          onClick={() => openEdit(profile)}
                        >
                          <Pencil size={14} /> 编辑
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="bm-row-action-delete rounded-lg"
                          onClick={() => setRemoving(profile)}
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
          {pageProfiles.length === 0 ? (
            <EmptyState className="flex-1">
              {profiles.length === 0 ? '暂无采集配置' : '没有匹配的采集配置'}
            </EmptyState>
          ) : null}
        </div>
      </DataTableShell>

      <CollectProfileDialog
        draft={draft}
        tab={tab}
        busy={busy}
        onTabChange={setTab}
        onDraftChange={setDraft}
        onClose={() => setDraft(null)}
        onSave={() => void save()}
      />

      <ConfirmDialog
        open={Boolean(removing)}
        title="删除采集配置"
        description="删除后不可恢复，关联的采集规则也会一并移除"
        detail={`确认删除采集配置「${removing?.name ?? ''}」吗？`}
        horizontalHeader
        busy={busy}
        onOpenChange={open => !open && setRemoving(null)}
        onConfirm={() => void remove()}
      />
    </div>
  );
}
