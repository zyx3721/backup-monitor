import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  CircleCheck,
  CircleX,
  Copy,
  History,
  Pencil,
  Plus,
  Search,
  Send,
  Trash2,
} from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/feedback/app-tooltip';
import { Button } from '@/components/ui/button';
import { DataTableShell } from '@/components/data/data-table-shell';
import { Pagination } from '@/components/data/pagination';
import { ConfirmDialog } from '@/components/dialogs/confirm-dialog';
import { Input } from '@/components/ui/input';
import { userHasPermission } from '@/lib/permissions';
import { getStoredUser } from '@/lib/session';
import { errorMessageOf } from '@/lib/toast-errors';
import { createChannel, deleteChannel, fetchChannels, testChannel, updateChannel } from './api';
import { ChannelDialog } from './ChannelDialog';
import { BackupSelect, EmptyState, EventTag, PageHeader, Tag } from './components';
import { NotificationRecordsDialog } from './NotificationRecordsDialog';
import {
  type ChannelFormState,
  channelToForm,
  copyChannelForm,
  emptyChannelForm,
  formToPayload,
  levelLabels,
  levelTones,
  testChannelFailureText,
  testChannelSuccessText,
  typeLabels,
} from './notify-config';
import type { NotifyChannel } from './types';

export function NotificationsPage() {
  const queryClient = useQueryClient();
  const user = getStoredUser();
  const canManage = userHasPermission(user, 'backup.notifications.manage');
  const [keyword, setKeyword] = useState('');
  const [eventType, setEventType] = useState('');
  const [level, setLevel] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(18);
  const [form, setForm] = useState<ChannelFormState | null>(null);
  const [copyMode, setCopyMode] = useState(false);
  const [deleting, setDeleting] = useState<NotifyChannel | null>(null);
  const [recordsOpen, setRecordsOpen] = useState(false);
  const [recordsChannelId, setRecordsChannelId] = useState('');

  const channels = useQuery({ queryKey: ['backup', 'channels'], queryFn: fetchChannels });

  const save = useMutation({
    mutationFn: (state: ChannelFormState) => {
      const payload = formToPayload(state);
      return state.id ? updateChannel(state.id, payload) : createChannel(payload);
    },
    onSuccess: (_result, state) => {
      toast.success(`通知渠道 ${state.name} ${state.id ? '已更新' : '已创建'}`);
      setForm(null);
      void queryClient.invalidateQueries({ queryKey: ['backup'] });
    },
    onError: (error, state) =>
      toast.error(
        `通知渠道 ${state.name} ${state.id ? '更新' : '创建'}失败：${errorMessageOf(error, '未知错误')}`
      ),
  });
  const test = useMutation({
    mutationFn: (channel: NotifyChannel) => testChannel(channel.id),
    onSuccess: (result, channel) => {
      if (result.ok) toast.success(testChannelSuccessText(channel));
      else toast.error(testChannelFailureText(channel, result.message));
      void queryClient.invalidateQueries({ queryKey: ['backup'] });
    },
    onError: (error, channel) =>
      toast.error(testChannelFailureText(channel, errorMessageOf(error, '未知错误'))),
  });
  const remove = useMutation({
    mutationFn: (id: string) => deleteChannel(id),
    onSuccess: () => {
      toast.success(`通知渠道 ${deleting?.name ?? ''} 已删除`);
      setDeleting(null);
      void queryClient.invalidateQueries({ queryKey: ['backup'] });
    },
    onError: error =>
      toast.error(
        `通知渠道 ${deleting?.name ?? ''} 删除失败：${errorMessageOf(error, '未知错误')}`
      ),
  });
  const toggle = useMutation({
    mutationFn: (channel: NotifyChannel) =>
      updateChannel(channel.id, { ...channel, enabled: !channel.enabled }),
    onSuccess: (_result, channel) => {
      toast.success(`通知渠道 ${channel.name} ${channel.enabled ? '已停用' : '已启用'}`);
      void queryClient.invalidateQueries({ queryKey: ['backup'] });
    },
    onError: (error, channel) =>
      toast.error(
        `通知渠道 ${channel.name} ${channel.enabled ? '停用' : '启用'}失败：${errorMessageOf(error, '未知错误')}`
      ),
  });

  const channelList = useMemo(() => channels.data ?? [], [channels.data]);
  const filteredChannels = useMemo(() => {
    const key = keyword.trim().toLowerCase();
    return channelList.filter(channel => {
      const hit =
        !key ||
        [channel.name, channel.remark ?? '', typeLabels[channel.type]].some(value =>
          value.toLowerCase().includes(key)
        );
      return (
        hit &&
        (eventType === '' || channel.eventType.includes(eventType)) &&
        (level === '' || channel.level === level)
      );
    });
  }, [channelList, keyword, eventType, level]);
  const rowCount = filteredChannels.length;
  const effectivePageSize = pageSize === -1 ? Math.max(rowCount, 1) : pageSize;
  const pageCount = Math.max(1, Math.ceil(rowCount / effectivePageSize));
  const currentPage = Math.min(page, pageCount);
  const pageStart = (currentPage - 1) * effectivePageSize;
  const pageChannels = filteredChannels.slice(pageStart, pageStart + effectivePageSize);

  useEffect(() => setPage(1), [keyword, eventType, level, pageSize]);
  useEffect(() => {
    if (page > pageCount) setPage(pageCount);
  }, [page, pageCount]);

  return (
    <div className="bm-page-fill">
      <PageHeader title="通知管理" description="告警通知渠道、事件订阅与级别过滤" />
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
                placeholder="搜索通知名称、通知方式、备注"
                className="pl-9"
              />
            </div>
            <BackupSelect
              value={eventType}
              onValueChange={setEventType}
              className="h-10 w-36"
              ariaLabel="事件类型"
              placeholder="事件类型"
              clearable
              options={[
                { value: 'backup_failed', label: '备份失败' },
                { value: 'not_run', label: '未按时执行' },
                { value: 'report_timeout', label: '主机异常/离线' },
              ]}
            />
            <BackupSelect
              value={level}
              onValueChange={setLevel}
              className="h-10 w-32"
              ariaLabel="告警级别"
              placeholder="级别"
              clearable
              options={[
                { value: 'info', label: '提示及以上' },
                { value: 'warning', label: '警告及以上' },
                { value: 'critical', label: '仅严重' },
              ]}
            />
            <span className="ml-auto text-xs text-[var(--bm-text-muted)]">
              共 {filteredChannels.length} 条
            </span>
            {canManage ? (
              <Button
                className="bm-primary-soft-button shrink-0"
                onClick={() => {
                  setCopyMode(false);
                  setForm({ ...emptyChannelForm });
                }}
              >
                <Plus size={16} />
                添加通知
              </Button>
            ) : null}
          </>
        }
      >
        <div className="flex min-h-full flex-col">
          <table className="w-full min-w-[1080px] text-center text-sm">
            <thead className="sticky top-0 z-10 bg-[var(--bm-table-head-bg)] text-center text-xs font-semibold text-[var(--bm-text-muted)]">
              <tr>
                <th className="px-4 py-3 font-semibold">序号</th>
                <th className="px-4 py-3 font-semibold">名称</th>
                <th className="px-4 py-3 font-semibold">通知方式</th>
                <th className="px-4 py-3 font-semibold">事件类型</th>
                <th className="px-4 py-3 font-semibold">告警级别</th>
                <th className="px-4 py-3 font-semibold">恢复通知</th>
                <th className="px-4 py-3 font-semibold">备注</th>
                <th className="px-4 py-3 font-semibold">操作</th>
              </tr>
            </thead>
            <tbody>
              {pageChannels.map((channel, index) => (
                <tr
                  key={channel.id}
                  className="border-t border-[var(--bm-border)]/60 bm-table-row-hover"
                >
                  <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                    {pageStart + index + 1}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center justify-center gap-2">
                      <span className="truncate font-medium text-[var(--bm-text)]">
                        {channel.name}
                      </span>
                    </div>
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-sm text-[var(--bm-text)]">
                    {typeLabels[channel.type]}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap justify-center gap-1">
                      {channel.eventType.map(event => (
                        <EventTag key={event} event={event} />
                      ))}
                    </div>
                  </td>
                  <td className="whitespace-nowrap px-4 py-3">
                    <Tag tone={levelTones[channel.level] ?? 'muted'}>
                      {levelLabels[channel.level] ?? channel.level}
                    </Tag>
                  </td>
                  <td className="whitespace-nowrap px-4 py-3">
                    <Tag tone={channel.recover ? 'success' : 'orange'}>
                      {channel.recover ? '已开启' : '未开启'}
                    </Tag>
                  </td>
                  <td
                    className="max-w-64 truncate px-4 py-3 text-sm text-[var(--bm-text)]"
                    title={channel.remark ?? ''}
                  >
                    {channel.remark || '-'}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center justify-center gap-2">
                      {canManage ? (
                        <>
                          <AppTooltip label="编辑">
                            <Button
                              variant="ghost"
                              size="sm"
                              aria-label="编辑通知"
                              className="bm-row-action-edit h-8 w-8 rounded-lg px-0"
                              onClick={() => {
                                setCopyMode(false);
                                setForm(channelToForm(channel));
                              }}
                            >
                              <Pencil size={14} />
                            </Button>
                          </AppTooltip>
                          <AppTooltip label="复制">
                            <Button
                              variant="ghost"
                              size="sm"
                              aria-label="复制通知"
                              className="bm-row-action-copy h-8 w-8 rounded-lg px-0"
                              onClick={() => {
                                setCopyMode(true);
                                setForm(copyChannelForm(channel));
                              }}
                            >
                              <Copy size={14} />
                            </Button>
                          </AppTooltip>
                          <AppTooltip label="测试">
                            <Button
                              variant="ghost"
                              size="sm"
                              aria-label="测试通知"
                              className="bm-row-action-test h-8 w-8 rounded-lg px-0"
                              onClick={() => test.mutate(channel)}
                              disabled={test.isPending}
                            >
                              <Send size={14} />
                            </Button>
                          </AppTooltip>
                          <AppTooltip
                            label={channel.enabled ? '已启用（点击停用）' : '已停用（点击启用）'}
                          >
                            <Button
                              variant="ghost"
                              size="sm"
                              aria-label={channel.enabled ? '停用通知' : '启用通知'}
                              className={`${
                                channel.enabled ? 'bm-row-action-success' : 'bm-row-action-warning'
                              } h-8 w-8 rounded-lg px-0`}
                              onClick={() => toggle.mutate(channel)}
                              disabled={toggle.isPending}
                            >
                              {channel.enabled ? <CircleCheck size={14} /> : <CircleX size={14} />}
                            </Button>
                          </AppTooltip>
                          <AppTooltip label="删除">
                            <Button
                              variant="ghost"
                              size="sm"
                              aria-label="删除通知"
                              className="bm-row-action-delete h-8 w-8 rounded-lg px-0"
                              onClick={() => setDeleting(channel)}
                            >
                              <Trash2 size={14} />
                            </Button>
                          </AppTooltip>
                        </>
                      ) : null}
                      <AppTooltip label="查看发送记录">
                        <Button
                          variant="ghost"
                          size="sm"
                          aria-label="查看发送记录"
                          className="h-8 w-8 rounded-lg px-0 text-[var(--bm-text-muted)]"
                          onClick={() => {
                            setRecordsChannelId(channel.id);
                            setRecordsOpen(true);
                          }}
                        >
                          <History size={14} />
                        </Button>
                      </AppTooltip>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {rowCount === 0 ? <EmptyState className="flex-1">暂无数据</EmptyState> : null}
        </div>
      </DataTableShell>

      {form ? (
        <ChannelDialog
          form={form}
          busy={save.isPending}
          title={copyMode ? '复制通知' : undefined}
          onChange={setForm}
          onClose={() => setForm(null)}
          onSubmit={() => save.mutate(form)}
        />
      ) : null}

      <NotificationRecordsDialog
        open={recordsOpen}
        onOpenChange={setRecordsOpen}
        channelId={recordsChannelId}
      />

      <ConfirmDialog
        open={Boolean(deleting)}
        title="删除通知渠道"
        description="删除后不可恢复，且该渠道不再接收新的告警通知"
        detail={`确认删除通知渠道「${deleting?.name ?? ''}」吗？`}
        horizontalHeader
        busy={remove.isPending}
        onOpenChange={open => !open && setDeleting(null)}
        onConfirm={() => deleting && remove.mutate(deleting.id)}
      />
    </div>
  );
}
