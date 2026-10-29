import { useQuery } from '@tanstack/react-query';
import { History, Search } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { AppTooltip } from '@/components/feedback/app-tooltip';
import { Pagination } from '@/components/data/pagination';
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { fetchNotificationRecords } from './api';
import { BackupSelect, EventTag, Tag, formatTime } from './components';
import { alertLevelLabels, levelTones } from './notify-config';

const eventOptions = [
  { value: 'test', label: '测试' },
  { value: 'backup_failed', label: '备份失败' },
  { value: 'not_run', label: '未按时执行' },
  { value: 'report_timeout', label: '主机异常/离线' },
];

const levelOptions = Object.entries(alertLevelLabels).map(([value, label]) => ({ value, label }));

const statusOptions = [
  { value: 'sent', label: '已发送' },
  { value: 'failed', label: '失败' },
];

export function NotificationRecordsDialog({
  open,
  onOpenChange,
  channelId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  channelId?: string;
}) {
  const { data = [], isLoading } = useQuery({
    queryKey: ['backup', 'notifications', channelId ?? 'all'],
    queryFn: () => fetchNotificationRecords(500, channelId),
    enabled: open,
  });
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(18);
  const [keyword, setKeyword] = useState('');
  const [event, setEvent] = useState('');
  const [level, setLevel] = useState('');
  const [status, setStatus] = useState('');

  useEffect(() => {
    if (open) setPage(1);
  }, [open]);

  useEffect(() => {
    setPage(1);
  }, [keyword, event, level, status, pageSize]);

  const records = useMemo(() => {
    const key = keyword.trim().toLowerCase();
    return data.filter(record => {
      const hit =
        !key ||
        [record.recipients ?? '', record.error ?? '', record.subject ?? ''].some(value =>
          value.toLowerCase().includes(key)
        );
      return (
        hit &&
        (event === '' || record.type === event) &&
        (level === '' || (record.level ?? 'all') === level) &&
        (status === '' || record.status === status)
      );
    });
  }, [data, keyword, event, level, status]);

  const effectivePageSize = pageSize === -1 ? Math.max(records.length, 1) : pageSize;
  const pageCount = Math.max(1, Math.ceil(records.length / effectivePageSize));
  const currentPage = Math.min(page, pageCount);
  const pageStart = (currentPage - 1) * effectivePageSize;
  const pageRecords = records.slice(pageStart, pageStart + effectivePageSize);

  useEffect(() => {
    if (page > pageCount) setPage(pageCount);
  }, [page, pageCount]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[88vh] w-[94vw] max-w-7xl flex-col gap-0 overflow-hidden p-0">
        <div className="flex shrink-0 flex-wrap items-start gap-2 border-b border-[var(--bm-border)] px-6 py-4 pr-14">
          <History size={18} className="mt-0.5 text-[var(--bm-accent-text)]" />
          <div className="min-w-0">
            <DialogTitle className="text-base font-semibold">发送记录</DialogTitle>
            <DialogDescription className="mt-1 text-xs">
              查看告警与测试通知的投递结果
            </DialogDescription>
          </div>
          <div className="ml-auto flex flex-wrap items-center gap-2">
            <div className="relative min-w-56">
              <Search
                size={15}
                className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--bm-text-muted)]"
              />
              <Input
                value={keyword}
                onChange={changeEvent => setKeyword(changeEvent.target.value)}
                placeholder="搜索接收目标、说明"
                className="h-9 pl-9"
              />
            </div>
            <BackupSelect
              value={event}
              onValueChange={setEvent}
              className="h-9 w-35"
              ariaLabel="通知事件"
              placeholder="事件"
              clearable
              options={eventOptions}
            />
            <BackupSelect
              value={level}
              onValueChange={setLevel}
              className="h-9 w-32"
              ariaLabel="通知级别"
              placeholder="级别"
              clearable
              options={levelOptions}
            />
            <BackupSelect
              value={status}
              onValueChange={setStatus}
              className="h-9 w-28"
              ariaLabel="发送状态"
              placeholder="状态"
              clearable
              options={statusOptions}
            />
          </div>
        </div>
        <div className="bm-hidden-scrollbar flex min-h-0 flex-1 flex-col overflow-auto">
          <table className="w-full min-w-[960px] shrink-0 table-fixed text-center text-sm">
            <colgroup>
              <col className="w-[180px]" />
              <col className="w-[132px]" />
              <col className="w-[104px]" />
              <col />
              <col className="w-[104px]" />
              <col />
            </colgroup>
            <thead className="sticky top-0 z-10 bg-[var(--bm-table-head-bg)] text-xs font-semibold text-[var(--bm-text-muted)]">
              <tr>
                <th className="whitespace-nowrap px-4 py-3 font-semibold">时间</th>
                <th className="whitespace-nowrap px-4 py-3 font-semibold">事件</th>
                <th className="whitespace-nowrap px-4 py-3 font-semibold">级别</th>
                <th className="px-4 py-3 font-semibold">接收目标</th>
                <th className="whitespace-nowrap px-4 py-3 font-semibold">状态</th>
                <th className="px-4 py-3 font-semibold">说明</th>
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                <tr>
                  <td colSpan={6} className="px-4 py-12 text-xs text-[var(--bm-text-muted)]">
                    正在读取发送记录
                  </td>
                </tr>
              ) : null}
              {pageRecords.map(record => (
                <tr
                  key={record.id}
                  className="border-t border-[var(--bm-border)]/60 bm-table-row-hover"
                >
                  <td className="whitespace-nowrap px-4 py-3 text-sm text-[var(--bm-text)]">
                    {formatTime(record.createdAt)}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3">
                    {record.type === 'test' ? (
                      <Tag tone="test">测试</Tag>
                    ) : (
                      <EventTag event={record.type} />
                    )}
                  </td>
                  <td className="whitespace-nowrap px-4 py-3">
                    {record.type === 'test' || !record.level ? (
                      <span className="text-[var(--bm-text-muted)]">-</span>
                    ) : (
                      <Tag tone={levelTones[record.level] ?? 'muted'}>
                        {alertLevelLabels[record.level] ?? record.level}
                      </Tag>
                    )}
                  </td>
                  <td className="px-4 py-3 text-sm text-[var(--bm-text)]">
                    <AppTooltip label={record.recipients || undefined} wrap>
                      <span className="mx-auto block max-w-64 truncate">
                        {record.recipients || '-'}
                      </span>
                    </AppTooltip>
                  </td>
                  <td className="whitespace-nowrap px-4 py-3">
                    <Tag tone={record.status === 'sent' ? 'success' : 'danger'}>
                      {record.status === 'sent' ? '已发送' : '失败'}
                    </Tag>
                  </td>
                  <td className="px-4 py-3 text-xs text-[var(--bm-text-muted)]">
                    <AppTooltip label={record.error || record.subject || undefined} wrap>
                      <span
                        className={`mx-auto block max-w-72 truncate ${record.error ? 'text-red-300' : ''}`}
                      >
                        {record.error || record.subject || '-'}
                      </span>
                    </AppTooltip>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {!isLoading && records.length === 0 ? (
            <div className="flex flex-1 items-center justify-center px-4 py-10 text-xs text-[var(--bm-text-muted)]">
              暂无发送记录
            </div>
          ) : null}
        </div>
        <Pagination
          page={currentPage}
          pageCount={pageCount}
          pageSize={pageSize}
          total={records.length}
          onPageChange={setPage}
          onPageSizeChange={size => {
            setPageSize(size);
            setPage(1);
          }}
        />
      </DialogContent>
    </Dialog>
  );
}
