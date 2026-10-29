import { useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { FolderOpen, Loader2, Monitor, Search, X } from 'lucide-react';
import { useRef, useState } from 'react';
import { fetchHosts, fetchTasks } from '@/features/backup/api';
import { PERM, userHasPermission } from '@/lib/permissions';
import { getStoredUser } from '@/lib/session';

export function HeaderSearch() {
  const navigate = useNavigate();
  const inputRef = useRef<HTMLInputElement>(null);
  const [value, setValue] = useState('');
  const [open, setOpen] = useState(false);
  const canSearchHosts = userHasPermission(getStoredUser(), PERM.hostsRead);
  const canSearchTasks = userHasPermission(getStoredUser(), PERM.tasksRead);
  const scopeLabel = [canSearchHosts ? '主机' : '', canSearchTasks ? '任务' : '']
    .filter(Boolean)
    .join('、');
  const { data: hosts = [], isLoading: hostsLoading } = useQuery({
    queryKey: ['backup', 'hosts'],
    queryFn: () => fetchHosts(),
    staleTime: 0,
    retry: false,
    enabled: open && canSearchHosts,
    refetchInterval: open && canSearchHosts ? 10_000 : false,
  });
  const { data: tasks = [], isLoading: tasksLoading } = useQuery({
    queryKey: ['backup', 'tasks'],
    queryFn: fetchTasks,
    staleTime: 0,
    retry: false,
    enabled: open && canSearchTasks,
    refetchInterval: open && canSearchTasks ? 10_000 : false,
  });
  const keyword = value.trim().toLowerCase();
  const matchedHosts = keyword
    ? hosts
        .filter(
          host => host.hostname.toLowerCase().includes(keyword) || (host.ip ?? '').includes(keyword)
        )
        .slice(0, 5)
    : [];
  const matchedTasks = keyword
    ? tasks
        .filter(
          task =>
            task.name.toLowerCase().includes(keyword) ||
            task.hostName.toLowerCase().includes(keyword)
        )
        .slice(0, 8)
    : [];
  const hasResults = matchedHosts.length + matchedTasks.length > 0;
  const searching = open && Boolean(keyword) && (hostsLoading || tasksLoading);

  if (!canSearchHosts && !canSearchTasks) return null;

  const close = () => {
    setOpen(false);
    setValue('');
  };

  const openHost = (id: string) => {
    close();
    void navigate({ to: '/hosts/$hostID', params: { hostID: id } });
  };

  const openTask = (id: string) => {
    close();
    void navigate({ to: '/tasks/$taskID', params: { taskID: id }, search: {} });
  };

  return (
    <div className="group relative hidden lg:block">
      <Search
        size={15}
        className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--bm-text-muted)] transition-colors group-hover:text-[var(--bm-accent)] group-focus-within:text-[var(--bm-accent)]"
        aria-hidden="true"
      />
      <input
        ref={inputRef}
        value={value}
        onChange={event => {
          setValue(event.target.value);
          setOpen(true);
        }}
        onFocus={() => setOpen(true)}
        onBlur={() => window.setTimeout(() => setOpen(false), 150)}
        onKeyDown={event => {
          if (event.key === 'Escape') {
            close();
            inputRef.current?.blur();
          }
        }}
        placeholder={`搜索${scopeLabel}…`}
        aria-label={`搜索${scopeLabel}`}
        className="h-[42px] w-[340px] rounded-lg border border-[var(--bm-border)] bg-[var(--bm-control-bg)] pl-9 pr-9 text-sm text-[var(--bm-text)] outline-none transition hover:border-[var(--bm-accent)] hover:shadow-[0_0_0_3px_rgba(59,130,246,0.08)] focus:border-[var(--bm-accent)] focus:shadow-[0_0_0_3px_rgba(59,130,246,0.16)]"
      />
      {value ? (
        <button
          type="button"
          onMouseDown={event => event.preventDefault()}
          onClick={() => {
            setValue('');
            inputRef.current?.focus();
          }}
          className="absolute right-2 top-1/2 grid h-7 w-7 -translate-y-1/2 place-items-center rounded-md text-[var(--bm-text-muted)] transition hover:bg-white/[0.06] hover:text-[var(--bm-text)]"
          aria-label="清空搜索"
        >
          <X size={14} />
        </button>
      ) : null}
      {open && keyword ? (
        <div
          className="absolute left-0 top-[calc(100%+8px)] z-[1200] max-h-[calc(100vh-5rem)] w-[340px] overflow-y-auto rounded-xl border p-2"
          style={{
            background:
              'linear-gradient(180deg, rgba(59,130,246,0.09), rgba(59,130,246,0) 96px), var(--bm-popover-bg)',
            borderColor: 'var(--bm-popover-border)',
            boxShadow: 'var(--bm-menu-shadow)',
          }}
          onMouseDown={event => event.preventDefault()}
        >
          {searching ? (
            <div className="flex flex-col items-center gap-2 px-3 py-8">
              <Loader2 size={18} className="animate-spin text-[var(--bm-text-muted)]" />
              <span className="text-xs text-[var(--bm-text-muted)]">搜索中…</span>
            </div>
          ) : hasResults ? (
            <div className="space-y-1.5">
              {matchedHosts.length > 0 ? (
                <div className="rounded-lg bg-[rgba(148,163,184,0.06)] p-1.5">
                  <div className="flex items-center gap-2 px-2 pb-1 pt-1">
                    <span className="h-1.5 w-1.5 rounded-full bg-[#60a5fa]" />
                    <span className="text-[11px] font-bold text-[var(--bm-text)]">主机</span>
                    <span className="ml-auto rounded-full bg-[var(--bm-popover-bg)] px-1.5 py-0.5 text-[10px] font-medium text-[var(--bm-text-muted)]">
                      {matchedHosts.length}
                    </span>
                  </div>
                  {matchedHosts.map(host => (
                    <button
                      key={host.id}
                      type="button"
                      onClick={() => openHost(host.id)}
                      className="flex h-11 w-full items-center gap-2.5 rounded-lg px-2 text-left transition hover:bg-[rgba(96,165,250,0.12)]"
                    >
                      <span className="grid h-7 w-7 shrink-0 place-items-center rounded-md bg-[rgba(96,165,250,0.12)] text-[#60a5fa]">
                        <Monitor size={14} />
                      </span>
                      <span className="min-w-0 flex-1 truncate text-sm text-[var(--bm-text)]">
                        {host.hostname}
                      </span>
                      <span className="max-w-[42%] shrink-0 truncate font-mono text-[11px] text-[var(--bm-text-muted)]">
                        {host.ip}
                      </span>
                    </button>
                  ))}
                </div>
              ) : null}
              {matchedTasks.length > 0 ? (
                <div className="rounded-lg bg-[rgba(148,163,184,0.06)] p-1.5">
                  <div className="flex items-center gap-2 px-2 pb-1 pt-1">
                    <span className="h-1.5 w-1.5 rounded-full bg-[#a78bfa]" />
                    <span className="text-[11px] font-bold text-[var(--bm-text)]">备份任务</span>
                    <span className="ml-auto rounded-full bg-[var(--bm-popover-bg)] px-1.5 py-0.5 text-[10px] font-medium text-[var(--bm-text-muted)]">
                      {matchedTasks.length}
                    </span>
                  </div>
                  {matchedTasks.map(task => (
                    <button
                      key={task.id}
                      type="button"
                      onClick={() => openTask(task.id)}
                      className="flex h-11 w-full items-center gap-2.5 rounded-lg px-2 text-left transition hover:bg-[rgba(167,139,250,0.12)]"
                    >
                      <span className="grid h-7 w-7 shrink-0 place-items-center rounded-md bg-[rgba(167,139,250,0.14)] text-[#a78bfa]">
                        <FolderOpen size={14} />
                      </span>
                      <span className="min-w-0 flex-1 truncate text-sm text-[var(--bm-text)]">
                        {task.name}
                      </span>
                      <span className="max-w-[42%] shrink-0 truncate text-[11px] text-[var(--bm-text-muted)]">
                        {task.hostName}
                      </span>
                    </button>
                  ))}
                </div>
              ) : null}
            </div>
          ) : (
            <div className="flex flex-col items-center gap-2 px-3 py-8">
              <span className="grid h-9 w-9 place-items-center rounded-full bg-[rgba(148,163,184,0.1)] text-[var(--bm-text-muted)]">
                <Search size={16} />
              </span>
              <span className="text-xs text-[var(--bm-text-muted)]">未找到匹配的{scopeLabel}</span>
            </div>
          )}
        </div>
      ) : null}
    </div>
  );
}
