import { Link } from '@tanstack/react-router';
import { useId, useMemo, useState } from 'react';
import type { StatusSummary, TaskStatus } from './types';

const meta: Record<TaskStatus, { label: string; color: string }> = {
  normal: { label: '正常', color: '#34d399' },
  running: { label: '运行中', color: '#60a5fa' },
  failed: { label: '失败', color: '#f87171' },
  not_run: { label: '未执行', color: '#fbbf24' },
  no_data: { label: '无数据', color: '#94a3b8' },
};

const order: TaskStatus[] = ['normal', 'running', 'failed', 'not_run', 'no_data'];
const SIZE = 200;
const CENTER = SIZE / 2;
const RADIUS = 73;
const STROKE = 24;
const CIRCUMFERENCE = 2 * Math.PI * RADIUS;

function adjustHexColor(color: string, amount: number) {
  const normalized = color.replace('#', '');
  if (!/^[0-9a-fA-F]{6}$/.test(normalized)) return color;
  const channels = [0, 2, 4].map(start => {
    const value = parseInt(normalized.slice(start, start + 2), 16);
    return Math.max(0, Math.min(255, value + amount))
      .toString(16)
      .padStart(2, '0');
  });
  return `#${channels.join('')}`;
}

export function StatusDonut({ summary, total }: { summary: StatusSummary[]; total: number }) {
  const [active, setActive] = useState<TaskStatus | null>(null);
  const chartId = useId().replace(/:/g, '');
  const items = useMemo(() => {
    const counts = new Map(summary.map(item => [item.status, item.count]));
    return order.map(status => ({ status, ...meta[status], count: counts.get(status) ?? 0 }));
  }, [summary]);
  const chartItems = items.filter(item => item.count > 0);
  let cursor = 0;
  const slices = chartItems.map(item => {
    const length = total ? (item.count / total) * CIRCUMFERENCE : 0;
    const gap = chartItems.length > 1 ? 3 : 0;
    const dash = Math.max(0, length - gap);
    const offset = cursor;
    const startFraction = CIRCUMFERENCE ? offset / CIRCUMFERENCE : 0;
    const endFraction = CIRCUMFERENCE ? (offset + length) / CIRCUMFERENCE : 0;
    const midAngle = -90 + ((startFraction + endFraction) / 2) * 360;
    cursor += length;
    return {
      ...item,
      dash,
      offset,
      midAngle,
      percent: total ? item.count / total : 0,
    };
  });
  const activeSlice = slices.find(item => item.status === active);
  const activePosition = activeSlice
    ? {
        x: CENTER + Math.cos((activeSlice.midAngle * Math.PI) / 180) * RADIUS,
        y: CENTER + Math.sin((activeSlice.midAngle * Math.PI) / 180) * RADIUS,
      }
    : null;

  return (
    <div className="grid grid-cols-1 items-center gap-4 lg:grid-cols-[12.5rem_minmax(0,1fr)]">
      <div className="relative mx-auto h-[12.5rem] w-[12.5rem]">
        <svg viewBox={`0 0 ${SIZE} ${SIZE}`} className="h-full w-full">
          <defs>
            <filter id={`${chartId}-shadow`} x="-50%" y="-50%" width="200%" height="200%">
              <feDropShadow
                dx="0"
                dy="4"
                stdDeviation="4"
                floodColor="#020617"
                floodOpacity="0.3"
              />
            </filter>
            {slices.map(slice => (
              <linearGradient
                key={slice.status}
                id={`${chartId}-slice-${slice.status}`}
                x1="0"
                y1="0"
                x2="1"
                y2="1"
              >
                <stop offset="0%" stopColor={adjustHexColor(slice.color, 24)} stopOpacity="0.92" />
                <stop offset="52%" stopColor={slice.color} stopOpacity="0.84" />
                <stop
                  offset="100%"
                  stopColor={adjustHexColor(slice.color, -24)}
                  stopOpacity="0.78"
                />
              </linearGradient>
            ))}
          </defs>
          <circle
            cx={CENTER}
            cy={CENTER}
            r={RADIUS}
            fill="none"
            stroke="rgba(148,163,184,0.12)"
            strokeWidth={STROKE}
          />
          {slices.map(slice => {
            const isActive = active === slice.status;
            const dimmed = active !== null && !isActive;
            const angle = (slice.midAngle * Math.PI) / 180;
            const offsetX = isActive ? Math.cos(angle) * 3 : 0;
            const offsetY = isActive ? Math.sin(angle) * 3 : 0;
            return (
              <g
                key={slice.status}
                opacity={dimmed ? 0.24 : 1}
                filter={isActive ? `url(#${chartId}-shadow)` : undefined}
                style={{
                  transform: `translate(${offsetX}px, ${offsetY}px)`,
                  transition: 'transform 200ms ease, opacity 200ms ease',
                }}
              >
                {isActive ? (
                  <circle
                    cx={CENTER}
                    cy={CENTER}
                    r={RADIUS}
                    fill="none"
                    stroke="rgba(255,255,255,0.5)"
                    strokeWidth={STROKE + 3}
                    strokeDasharray={`${slice.dash} ${CIRCUMFERENCE - slice.dash}`}
                    strokeDashoffset={-slice.offset}
                    strokeLinecap="butt"
                    transform={`rotate(-90 ${CENTER} ${CENTER})`}
                  />
                ) : null}
                <circle
                  cx={CENTER}
                  cy={CENTER}
                  r={RADIUS}
                  fill="none"
                  stroke={`url(#${chartId}-slice-${slice.status})`}
                  strokeWidth={isActive ? STROKE + 1.5 : STROKE}
                  strokeDasharray={`${slice.dash} ${CIRCUMFERENCE - slice.dash}`}
                  strokeDashoffset={-slice.offset}
                  strokeLinecap="butt"
                  transform={`rotate(-90 ${CENTER} ${CENTER})`}
                  className="cursor-pointer"
                  onMouseEnter={() => setActive(slice.status)}
                  onMouseLeave={() => setActive(null)}
                />
              </g>
            );
          })}
          {slices.map(slice => {
            if (slice.percent < 0.08) return null;
            const angle = (slice.midAngle * Math.PI) / 180;
            return (
              <text
                key={slice.status}
                x={CENTER + Math.cos(angle) * RADIUS}
                y={CENTER + Math.sin(angle) * RADIUS}
                textAnchor="middle"
                dominantBaseline="central"
                fill="var(--bm-text)"
                className="pointer-events-none select-none font-mono text-[11px] font-semibold"
              >
                {slice.count}
              </text>
            );
          })}
        </svg>
        <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center text-center">
          <span className="font-mono text-3xl font-semibold tabular-nums text-[var(--bm-text)]">
            {total}
          </span>
          <span className="mt-1 text-[11px] text-[var(--bm-text-muted)]">任务总数</span>
        </div>
        {activeSlice && activePosition ? (
          <div
            className="pointer-events-none absolute z-10 flex items-center gap-2 whitespace-nowrap rounded-lg border px-2.5 py-1.5 text-xs shadow-xl"
            style={{
              left: `${(activePosition.x / SIZE) * 100}%`,
              top: `${(activePosition.y / SIZE) * 100}%`,
              transform:
                Math.abs(activePosition.y - CENTER) > Math.abs(activePosition.x - CENTER)
                  ? `translate(-50%, ${activePosition.y > CENTER ? '10px' : 'calc(-100% - 10px)'})`
                  : 'translate(10px, -50%)',
              background: 'var(--bm-popover-bg)',
              borderColor: 'var(--bm-popover-border)',
              boxShadow: 'var(--bm-menu-shadow)',
            }}
          >
            <span
              className="h-2 w-2 shrink-0 rounded-full"
              style={{
                background: activeSlice.color,
                boxShadow: `0 0 10px ${activeSlice.color}66`,
              }}
            />
            <span className="font-semibold text-[var(--bm-text)]">{activeSlice.label}</span>
            <span className="font-mono tabular-nums text-[var(--bm-text-muted)]">
              {activeSlice.count} 个
            </span>
          </div>
        ) : null}
      </div>
      <div className="flex min-w-0 flex-col gap-2">
        {items.map(item => {
          const pct = total ? Math.round((item.count / total) * 100) : 0;
          return (
            <Link
              key={item.status}
              to="/tasks"
              search={{ status: item.status, open: undefined }}
              className="flex items-center justify-between gap-1.5 rounded-lg border px-2 py-2 transition hover:border-blue-400/40 hover:bg-white/[0.04]"
              style={{
                background: item.count > 0 ? 'rgba(255,255,255,0.03)' : 'rgba(148,163,184,0.03)',
                borderColor:
                  active === item.status ? 'rgba(96,165,250,0.55)' : 'rgba(148,163,184,0.14)',
              }}
              onMouseEnter={() => setActive(item.status)}
              onMouseLeave={() => setActive(null)}
            >
              <span className="flex min-w-0 items-center gap-1.5">
                <span
                  className="h-2 w-2 shrink-0 rounded-full"
                  style={{ background: item.color, boxShadow: `0 0 12px ${item.color}66` }}
                />
                <span
                  className="truncate text-sm"
                  style={{ color: item.count > 0 ? 'var(--bm-text)' : 'var(--bm-text-muted)' }}
                >
                  {item.label}
                </span>
              </span>
              <span className="flex shrink-0 items-baseline gap-1 font-mono tabular-nums">
                <span className="text-sm text-[var(--bm-text)]">{item.count}</span>
                <span className="w-7 text-right text-xs text-[var(--bm-text-muted)]">{pct}%</span>
              </span>
            </Link>
          );
        })}
      </div>
    </div>
  );
}
