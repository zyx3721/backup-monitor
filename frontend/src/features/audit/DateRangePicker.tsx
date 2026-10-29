import { useEffect, useRef, useState } from 'react';
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight, Clock, X } from 'lucide-react';
import { createPortal } from 'react-dom';
import { DatePickerField } from './DatePickerField';
import { TimePicker } from './TimePicker';

export type DateRange = {
  start: string;
  end: string;
  startTime: string;
  endTime: string;
};

const emptyRange: DateRange = { start: '', end: '', startTime: '', endTime: '' };
const weekdays = ['一', '二', '三', '四', '五', '六', '日'];
const panelWidth = 640;
const panelHeight = 470;

// DateRangePicker 审计日志的时间范围选择器：双月面板加手动输入行，两次点击或手动填写确定起止时间
export function DateRangePicker({
  value,
  onChange,
}: {
  value: DateRange;
  onChange: (value: DateRange) => void;
}) {
  const triggerRef = useRef<HTMLButtonElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const [pending, setPending] = useState<DateRange>(emptyRange);
  const [hoverDate, setHoverDate] = useState<string>('');
  const [anchorMonth, setAnchorMonth] = useState<Date>(() => new Date());
  const [position, setPosition] = useState({ top: 0, left: 0 });

  useEffect(() => {
    if (!open) return;
    const closeOnOutside = (event: PointerEvent) => {
      const target = event.target as Node;
      if (!triggerRef.current?.contains(target) && !panelRef.current?.contains(target)) {
        setOpen(false);
      }
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
    const rect = triggerRef.current?.getBoundingClientRect();
    if (rect) {
      const width = Math.min(panelWidth, window.innerWidth - 24);
      const showAbove =
        window.innerHeight - rect.bottom - 8 < panelHeight && rect.top > panelHeight + 8;
      const center = rect.left + rect.width / 2 - width / 2;
      setPosition({
        left: Math.max(12, Math.min(center, window.innerWidth - width - 12)),
        top: showAbove ? rect.top - panelHeight - 8 : rect.bottom + 8,
      });
    }
    const base = parseRangeDate(value.start) ?? new Date();
    setAnchorMonth(new Date(base.getFullYear(), base.getMonth(), 1));
    setPending(isCompleteRange(value) ? value : emptyRange);
    setHoverDate('');
    setOpen(true);
  }

  function updatePending(next: DateRange) {
    const normalized = normalizeRange(next);
    setPending(normalized);
    if (isCompleteRange(normalized)) {
      onChange(normalized);
    }
  }

  function pickDate(date: Date) {
    const picked = formatRangeDate(date);
    if (pending.start && !pending.end) {
      const before = pending.start <= picked;
      const start = before ? pending.start : picked;
      const end = before ? picked : pending.start;
      const now = formatTimeOfDay(new Date());
      updatePending({
        start,
        end,
        startTime: before ? pending.startTime || now : now,
        endTime: before ? now : pending.startTime || now,
      });
      return;
    }
    setHoverDate('');
    updatePending({ start: picked, end: '', startTime: formatTimeOfDay(new Date()), endTime: '' });
  }

  function editStartDate(next: string) {
    const patched = { ...pending, start: next };
    if (next && !patched.startTime) patched.startTime = formatTimeOfDay(new Date());
    const date = parseRangeDate(next);
    if (date) setAnchorMonth(new Date(date.getFullYear(), date.getMonth(), 1));
    updatePending(patched);
  }

  function editEndDate(next: string) {
    const patched = { ...pending, end: next };
    if (next && !patched.endTime) patched.endTime = formatTimeOfDay(new Date());
    updatePending(patched);
  }

  function editStartTime(next: string) {
    updatePending({ ...pending, startTime: next });
  }

  function editEndTime(next: string) {
    updatePending({ ...pending, endTime: next });
  }

  function clearValue() {
    setPending(emptyRange);
    setHoverDate('');
    onChange(emptyRange);
    setOpen(false);
  }

  function applySelection() {
    if (isCompleteRange(pending)) {
      onChange(pending);
    }
    setOpen(false);
  }

  const startLabel = value.start ? `${value.start} ${value.startTime}`.trim() : '';
  const endLabel = value.end ? `${value.end} ${value.endTime}`.trim() : '';
  const hasValue = Boolean(startLabel && endLabel);
  const leftMonth = new Date(anchorMonth.getFullYear(), anchorMonth.getMonth(), 1);
  const rightMonth = new Date(anchorMonth.getFullYear(), anchorMonth.getMonth() + 1, 1);

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        onClick={togglePanel}
        aria-label="选择时间范围"
        aria-expanded={open}
        className="bm-form-control flex h-10 w-full cursor-pointer items-center gap-2 rounded-lg px-3 text-sm"
      >
        <Clock size={16} className="shrink-0 text-[var(--bm-text-muted)]" />
        <span className="grid min-w-0 flex-1 grid-cols-[1fr_auto_1fr] items-center gap-2">
          <span
            className={`min-w-0 truncate text-center ${startLabel ? '' : 'text-[var(--bm-text-muted)]'}`}
          >
            {startLabel || '开始时间'}
          </span>
          <span className="shrink-0 font-medium">至</span>
          <span
            className={`min-w-0 truncate text-center ${endLabel ? '' : 'text-[var(--bm-text-muted)]'}`}
          >
            {endLabel || '结束时间'}
          </span>
        </span>
        {hasValue ? (
          <span
            role="button"
            tabIndex={-1}
            aria-label="清空时间范围"
            className="grid h-5 w-5 shrink-0 place-items-center rounded-full text-[var(--bm-text-muted)] hover:bg-[var(--bm-control-bg-soft)] hover:text-[var(--bm-text)]"
            onClick={event => {
              event.stopPropagation();
              clearValue();
            }}
          >
            <X size={13} />
          </span>
        ) : null}
      </button>
      {open && typeof document !== 'undefined'
        ? createPortal(
            <div
              ref={panelRef}
              role="dialog"
              aria-label="选择时间范围"
              className="fixed z-[1400] rounded-xl border p-3 shadow-2xl"
              style={{
                top: position.top,
                left: position.left,
                width: Math.min(panelWidth, window.innerWidth - 24),
                background: 'var(--bm-popover-bg)',
                borderColor: 'var(--bm-popover-border)',
                boxShadow: 'var(--bm-menu-shadow)',
              }}
            >
              <div className="mb-3 grid grid-cols-[1fr_auto_1fr] items-stretch gap-2">
                <div className="grid min-w-0 grid-cols-2 gap-1.5">
                  <DatePickerField
                    value={pending.start}
                    onChange={editStartDate}
                    ariaLabel="开始日期"
                    align="left"
                  />
                  <TimePicker
                    value={pending.startTime}
                    onChange={editStartTime}
                    ariaLabel="开始时间"
                    align="left"
                  />
                </div>
                <ChevronRight
                  size={14}
                  className="shrink-0 self-center text-[var(--bm-text-muted)]"
                />
                <div className="grid min-w-0 grid-cols-2 gap-1.5">
                  <DatePickerField
                    value={pending.end}
                    onChange={editEndDate}
                    ariaLabel="结束日期"
                    align="right"
                  />
                  <TimePicker
                    value={pending.endTime}
                    onChange={editEndTime}
                    ariaLabel="结束时间"
                    align="right"
                  />
                </div>
              </div>
              <div className="flex gap-2">
                <MonthPanel
                  month={leftMonth}
                  pending={pending}
                  hoverDate={hoverDate}
                  onHover={setHoverDate}
                  onPick={pickDate}
                  nav={{
                    prevYear: () =>
                      setAnchorMonth(
                        current => new Date(current.getFullYear() - 1, current.getMonth(), 1)
                      ),
                    prevMonth: () =>
                      setAnchorMonth(
                        current => new Date(current.getFullYear(), current.getMonth() - 1, 1)
                      ),
                  }}
                />
                <div className="w-px bg-[var(--bm-border)]" />
                <MonthPanel
                  month={rightMonth}
                  pending={pending}
                  hoverDate={hoverDate}
                  onHover={setHoverDate}
                  onPick={pickDate}
                  nav={{
                    nextMonth: () =>
                      setAnchorMonth(
                        current => new Date(current.getFullYear(), current.getMonth() + 1, 1)
                      ),
                    nextYear: () =>
                      setAnchorMonth(
                        current => new Date(current.getFullYear() + 1, current.getMonth(), 1)
                      ),
                  }}
                />
              </div>
              <div className="mt-3 flex items-center justify-between border-t border-[var(--bm-border)] pt-3">
                <button
                  type="button"
                  onClick={clearValue}
                  disabled={!value.start && !value.end}
                  className="bm-action-button rounded-md px-2 py-1 text-xs font-medium text-red-500 disabled:cursor-not-allowed disabled:opacity-45"
                >
                  清空
                </button>
                <button
                  type="button"
                  onClick={applySelection}
                  className="bm-action-button rounded-md px-2 py-1 text-xs font-medium text-blue-500"
                >
                  确定
                </button>
              </div>
            </div>,
            document.body
          )
        : null}
    </>
  );
}

// MonthPanel 渲染单个月份网格：nav 提供左侧前进/后退或右侧前进/后退按钮
function MonthPanel({
  month,
  pending,
  hoverDate,
  onHover,
  onPick,
  nav,
}: {
  month: Date;
  pending: DateRange;
  hoverDate: string;
  onHover: (value: string) => void;
  onPick: (date: Date) => void;
  nav: {
    prevYear?: () => void;
    prevMonth?: () => void;
    nextMonth?: () => void;
    nextYear?: () => void;
  };
}) {
  const firstDay = new Date(month.getFullYear(), month.getMonth(), 1);
  const firstWeekday = (firstDay.getDay() + 6) % 7;
  const calendarDays = Array.from(
    { length: 42 },
    (_, index) => new Date(month.getFullYear(), month.getMonth(), index - firstWeekday + 1)
  );
  return (
    <div className="min-w-0 flex-1">
      <div className="relative mb-2 flex h-8 items-center justify-center">
        {nav.prevYear || nav.prevMonth ? (
          <div className="absolute left-0 flex items-center gap-0.5">
            {nav.prevYear ? (
              <button
                type="button"
                onClick={nav.prevYear}
                className="bm-action-button grid h-7 w-7 place-items-center rounded-md"
                aria-label="上一年"
              >
                <ChevronsLeft size={15} />
              </button>
            ) : null}
            {nav.prevMonth ? (
              <button
                type="button"
                onClick={nav.prevMonth}
                className="bm-action-button grid h-7 w-7 place-items-center rounded-md"
                aria-label="上一个月"
              >
                <ChevronLeft size={15} />
              </button>
            ) : null}
          </div>
        ) : null}
        <span className="text-sm font-semibold">
          {month.getFullYear()} 年 {month.getMonth() + 1} 月
        </span>
        {nav.nextMonth || nav.nextYear ? (
          <div className="absolute right-0 flex items-center gap-0.5">
            {nav.nextMonth ? (
              <button
                type="button"
                onClick={nav.nextMonth}
                className="bm-action-button grid h-7 w-7 place-items-center rounded-md"
                aria-label="下一个月"
              >
                <ChevronRight size={15} />
              </button>
            ) : null}
            {nav.nextYear ? (
              <button
                type="button"
                onClick={nav.nextYear}
                className="bm-action-button grid h-7 w-7 place-items-center rounded-md"
                aria-label="下一年"
              >
                <ChevronsRight size={15} />
              </button>
            ) : null}
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
          const key = formatRangeDate(date);
          const currentMonth = date.getMonth() === month.getMonth();
          const isStart = pending.start === key;
          const isEnd = pending.end === key;
          const rangeEnd = pending.end || (pending.start ? hoverDate : '');
          const inRange = Boolean(
            pending.start &&
            rangeEnd &&
            key !== pending.start &&
            key !== rangeEnd &&
            key > (pending.start < rangeEnd ? pending.start : rangeEnd) &&
            key < (pending.start < rangeEnd ? rangeEnd : pending.start)
          );
          return (
            <button
              key={key}
              type="button"
              onClick={() => onPick(date)}
              onMouseEnter={() => onHover(key)}
              onFocus={() => onHover(key)}
              className={`grid h-8 place-items-center rounded-md text-sm transition-colors ${
                isStart || isEnd
                  ? 'bg-blue-500 font-semibold text-white shadow-sm'
                  : inRange
                    ? 'bg-blue-500/12 text-[var(--bm-text)]'
                    : currentMonth
                      ? 'bm-action-button text-[var(--bm-text)]'
                      : 'bm-action-button text-[var(--bm-text-muted)] opacity-55'
              }`}
              aria-pressed={isStart || isEnd}
            >
              {date.getDate()}
            </button>
          );
        })}
      </div>
    </div>
  );
}

function isCompleteRange(range: DateRange) {
  return Boolean(range.start && range.end && range.startTime && range.endTime);
}

function normalizeRange(range: DateRange) {
  if (range.start && range.end && range.start > range.end) {
    return {
      start: range.end,
      end: range.start,
      startTime: range.endTime,
      endTime: range.startTime,
    };
  }
  return range;
}

function parseRangeDate(value: string) {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (!match) return null;
  const [, year, month, day] = match;
  const date = new Date(Number(year), Number(month) - 1, Number(day));
  return Number.isNaN(date.valueOf()) ? null : date;
}

function formatRangeDate(date: Date) {
  const pad = (input: number) => String(input).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

function formatTimeOfDay(date: Date) {
  const pad = (input: number) => String(input).padStart(2, '0');
  return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}
