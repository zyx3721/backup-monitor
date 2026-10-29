import { useEffect, useRef, useState } from 'react';
import { Clock } from 'lucide-react';

const hourItems = Array.from({ length: 24 }, (_, index) => index);
const minuteItems = Array.from({ length: 60 }, (_, index) => index);
const secondItems = Array.from({ length: 60 }, (_, index) => index);
const itemHeight = 28;

// TimePicker 时间选择器：时分秒三列弹层，点击列内数值完成选择
export function TimePicker({
  value,
  onChange,
  ariaLabel,
  align = 'left',
}: {
  value: string;
  onChange: (value: string) => void;
  ariaLabel: string;
  align?: 'left' | 'right';
}) {
  const wrapRef = useRef<HTMLDivElement>(null);
  const hourListRef = useRef<HTMLDivElement>(null);
  const minuteListRef = useRef<HTMLDivElement>(null);
  const secondListRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const parsed = parseTimeParts(value);

  useEffect(() => {
    if (!open) return;
    const closeOnOutside = (event: PointerEvent) => {
      if (!wrapRef.current?.contains(event.target as Node)) setOpen(false);
    };
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false);
    };
    document.addEventListener('pointerdown', closeOnOutside);
    window.addEventListener('keydown', closeOnEscape);
    return () => {
      document.removeEventListener('pointerdown', closeOnOutside);
      window.removeEventListener('keydown', closeOnEscape);
    };
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const match = /^(\d{1,2}):(\d{1,2})(?::(\d{1,2}))?$/.exec(value.trim());
    if (!match) return;
    scrollToItem(hourListRef.current, Number(match[1]));
    scrollToItem(minuteListRef.current, Number(match[2]));
    scrollToItem(secondListRef.current, Number(match[3] ?? 0));
  }, [open]);

  function scrollToItem(list: HTMLDivElement | null, index: number) {
    if (!list || Number.isNaN(index)) return;
    list.scrollTop = Math.max(0, index * itemHeight - itemHeight * 2);
  }

  function pick(part: 'hour' | 'minute' | 'second', item: number) {
    const next = {
      hour: part === 'hour' ? item : Math.max(parsed.hour, 0),
      minute: part === 'minute' ? item : Math.max(parsed.minute, 0),
      second: part === 'second' ? item : Math.max(parsed.second, 0),
    };
    onChange(`${pad(next.hour)}:${pad(next.minute)}:${pad(next.second)}`);
  }

  const columns = [
    {
      key: 'hour' as const,
      label: '时',
      items: hourItems,
      selected: parsed.hour,
      listRef: hourListRef,
    },
    {
      key: 'minute' as const,
      label: '分',
      items: minuteItems,
      selected: parsed.minute,
      listRef: minuteListRef,
    },
    {
      key: 'second' as const,
      label: '秒',
      items: secondItems,
      selected: parsed.second,
      listRef: secondListRef,
    },
  ];

  return (
    <div ref={wrapRef} className="relative min-w-0">
      <button
        type="button"
        onClick={() => setOpen(current => !current)}
        aria-label={ariaLabel}
        aria-expanded={open}
        className="bm-form-control flex h-9 w-full cursor-pointer items-center justify-between gap-1 rounded-md px-2 text-xs"
      >
        <span className={`min-w-0 truncate ${value ? '' : 'text-[var(--bm-text-muted)]'}`}>
          {value || '--:--:--'}
        </span>
        <Clock size={13} className="shrink-0 text-[var(--bm-text-muted)]" />
      </button>
      {open ? (
        <div
          role="dialog"
          aria-label={ariaLabel}
          className={`absolute top-full z-20 mt-1 rounded-lg border p-2 shadow-xl ${align === 'left' ? 'left-0' : 'right-0'}`}
          style={{
            background: 'var(--bm-popover-bg)',
            borderColor: 'var(--bm-popover-border)',
            boxShadow: 'var(--bm-menu-shadow)',
          }}
        >
          <div className="flex gap-1">
            {columns.map(column => (
              <div key={column.key} className="w-14">
                <div className="mb-1 text-center text-xs font-medium text-[var(--bm-text-muted)]">
                  {column.label}
                </div>
                <div ref={column.listRef} className="bm-hidden-scrollbar max-h-44 overflow-y-auto">
                  {column.items.map(item => {
                    const selected = item === column.selected;
                    return (
                      <button
                        key={item}
                        type="button"
                        onClick={() => pick(column.key, item)}
                        className={`grid h-7 w-full place-items-center rounded-md text-xs transition-colors ${
                          selected
                            ? 'bg-blue-500 font-semibold text-white shadow-sm'
                            : 'bm-action-button text-[var(--bm-text)]'
                        }`}
                        aria-pressed={selected}
                      >
                        {pad(item)}
                      </button>
                    );
                  })}
                </div>
              </div>
            ))}
          </div>
          <div className="mt-1 border-t border-[var(--bm-border)] pt-1.5">
            <button
              type="button"
              onClick={() => {
                const now = new Date();
                onChange(
                  `${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}`
                );
              }}
              className="bm-action-button w-full rounded-md py-1 text-xs font-medium text-blue-500"
            >
              此刻
            </button>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function parseTimeParts(value: string) {
  const match = /^(\d{1,2}):(\d{1,2})(?::(\d{1,2}))?$/.exec(value.trim());
  if (!match) return { hour: -1, minute: -1, second: -1 };
  return { hour: Number(match[1]), minute: Number(match[2]), second: Number(match[3] ?? 0) };
}

function pad(input: number) {
  return String(input).padStart(2, '0');
}
