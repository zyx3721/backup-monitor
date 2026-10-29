import { useEffect, useRef, useState } from 'react';
import { CalendarDays, ChevronDown, ChevronLeft, ChevronRight } from 'lucide-react';

const weekdays = ['一', '二', '三', '四', '五', '六', '日'];

// DatePickerField 单月日历选择字段：点击弹出项目风格日历，替代浏览器原生日期弹层
export function DatePickerField({
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
  const [open, setOpen] = useState(false);
  const [monthPickerOpen, setMonthPickerOpen] = useState(false);
  const [viewDate, setViewDate] = useState(() => new Date());
  const selectedDate = parseCalendarDate(value);
  const firstDay = new Date(viewDate.getFullYear(), viewDate.getMonth(), 1);
  const firstWeekday = (firstDay.getDay() + 6) % 7;
  const calendarDays = Array.from(
    { length: 42 },
    (_, index) => new Date(viewDate.getFullYear(), viewDate.getMonth(), index - firstWeekday + 1)
  );

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

  function togglePanel() {
    if (open) {
      setOpen(false);
      return;
    }
    const base = parseCalendarDate(value) ?? new Date();
    setViewDate(new Date(base.getFullYear(), base.getMonth(), 1));
    setMonthPickerOpen(false);
    setOpen(true);
  }

  function selectDate(date: Date) {
    onChange(formatCalendarDate(date));
    setOpen(false);
    setMonthPickerOpen(false);
  }

  function selectMonth(month: number) {
    setViewDate(current => new Date(current.getFullYear(), month, 1));
    setMonthPickerOpen(false);
  }

  return (
    <div ref={wrapRef} className="relative min-w-0">
      <button
        type="button"
        onClick={togglePanel}
        aria-label={ariaLabel}
        aria-expanded={open}
        className="bm-form-control flex h-9 w-full cursor-pointer items-center justify-between gap-1 rounded-md px-2 text-xs"
      >
        <span className={`min-w-0 truncate ${value ? '' : 'text-[var(--bm-text-muted)]'}`}>
          {value || '年/月/日'}
        </span>
        <CalendarDays size={13} className="shrink-0 text-[var(--bm-text-muted)]" />
      </button>
      {open ? (
        <div
          role="dialog"
          aria-label={ariaLabel}
          className={`absolute top-full z-20 mt-1 w-72 rounded-xl border p-3 shadow-xl ${align === 'left' ? 'left-0' : 'right-0'}`}
          style={{
            background: 'var(--bm-popover-bg)',
            borderColor: 'var(--bm-popover-border)',
            boxShadow: 'var(--bm-menu-shadow)',
          }}
        >
          <div className="relative mb-3 flex items-center justify-between">
            <button
              type="button"
              onClick={() => setMonthPickerOpen(current => !current)}
              className="bm-action-button inline-flex items-center gap-1 rounded-md px-2 py-1 text-sm font-semibold"
              aria-label="选择年份和月份"
              aria-expanded={monthPickerOpen}
            >
              {viewDate.getFullYear()}年{viewDate.getMonth() + 1}月
              <ChevronDown size={14} className="text-[var(--bm-text-muted)]" />
            </button>
            <div className="flex items-center gap-1">
              <button
                type="button"
                onClick={() =>
                  setViewDate(current => new Date(current.getFullYear(), current.getMonth() - 1, 1))
                }
                className="bm-action-button grid h-7 w-7 place-items-center rounded-md"
                aria-label="上一个月"
              >
                <ChevronLeft size={16} />
              </button>
              <button
                type="button"
                onClick={() =>
                  setViewDate(current => new Date(current.getFullYear(), current.getMonth() + 1, 1))
                }
                className="bm-action-button grid h-7 w-7 place-items-center rounded-md"
                aria-label="下一个月"
              >
                <ChevronRight size={16} />
              </button>
            </div>
            {monthPickerOpen ? (
              <div
                className="absolute left-0 top-9 z-10 w-full rounded-lg border p-3 shadow-xl"
                style={{
                  background: 'var(--bm-popover-bg)',
                  borderColor: 'var(--bm-popover-border)',
                  boxShadow: 'var(--bm-menu-shadow)',
                }}
              >
                <div className="mb-2 flex items-center justify-between rounded-md bg-[var(--bm-control-bg-soft)] px-2 py-1.5">
                  <button
                    type="button"
                    onClick={() =>
                      setViewDate(
                        current => new Date(current.getFullYear() - 1, current.getMonth(), 1)
                      )
                    }
                    className="bm-action-button grid h-6 w-6 place-items-center rounded-md"
                    aria-label="上一年"
                  >
                    <ChevronLeft size={15} />
                  </button>
                  <span className="text-sm font-semibold">{viewDate.getFullYear()} 年</span>
                  <button
                    type="button"
                    onClick={() =>
                      setViewDate(
                        current => new Date(current.getFullYear() + 1, current.getMonth(), 1)
                      )
                    }
                    className="bm-action-button grid h-6 w-6 place-items-center rounded-md"
                    aria-label="下一年"
                  >
                    <ChevronRight size={15} />
                  </button>
                </div>
                <div className="grid grid-cols-3 gap-1.5">
                  {Array.from({ length: 12 }, (_, month) => {
                    const selected = month === viewDate.getMonth();
                    return (
                      <button
                        key={month}
                        type="button"
                        onClick={() => selectMonth(month)}
                        className={`rounded-md px-2 py-2 text-sm transition-colors ${
                          selected
                            ? 'bg-blue-500 font-semibold text-white shadow-sm'
                            : 'bm-action-button text-[var(--bm-text)]'
                        }`}
                      >
                        {month + 1}月
                      </button>
                    );
                  })}
                </div>
              </div>
            ) : null}
          </div>
          <div className="grid grid-cols-7 gap-1 text-center text-xs text-[var(--bm-text-muted)]">
            {weekdays.map(day => (
              <span key={day} className="grid h-7 place-items-center font-medium">
                {day}
              </span>
            ))}
            {calendarDays.map(date => {
              const selected = sameCalendarDate(date, selectedDate);
              const currentMonth = date.getMonth() === viewDate.getMonth();
              return (
                <button
                  key={date.toISOString()}
                  type="button"
                  onClick={() => selectDate(date)}
                  className={`grid h-8 place-items-center rounded-md text-sm transition-colors ${
                    selected
                      ? 'bg-blue-500 font-semibold text-white shadow-sm'
                      : currentMonth
                        ? 'bm-action-button text-[var(--bm-text)]'
                        : 'bm-action-button text-[var(--bm-text-muted)] opacity-55'
                  }`}
                  aria-pressed={selected}
                >
                  {date.getDate()}
                </button>
              );
            })}
          </div>
          <div className="mt-3 flex items-center justify-between border-t border-[var(--bm-border)] pt-3">
            <button
              type="button"
              onClick={() => {
                onChange('');
                setOpen(false);
                setMonthPickerOpen(false);
              }}
              disabled={!value}
              className="bm-action-button rounded-md px-2 py-1 text-xs font-medium text-red-500 disabled:cursor-not-allowed disabled:opacity-45"
            >
              清空
            </button>
            <button
              type="button"
              onClick={() => selectDate(new Date())}
              className="bm-action-button rounded-md px-2 py-1 text-xs font-medium text-blue-500"
            >
              今天
            </button>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function parseCalendarDate(value: string) {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (!match) return null;
  const [, year, month, day] = match;
  const date = new Date(Number(year), Number(month) - 1, Number(day));
  return Number.isNaN(date.valueOf()) ? null : date;
}

function formatCalendarDate(date: Date) {
  const pad = (input: number) => String(input).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

function sameCalendarDate(left: Date, right: Date | null) {
  return Boolean(
    right &&
    left.getFullYear() === right.getFullYear() &&
    left.getMonth() === right.getMonth() &&
    left.getDate() === right.getDate()
  );
}
