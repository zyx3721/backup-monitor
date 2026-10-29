import { useState } from 'react';
import type { TrendPoint } from './types';

const series = [
  {
    key: 'success',
    label: '成功',
    color: '#34d399',
    bar: 'linear-gradient(180deg, rgba(52,211,153,0.94) 0%, rgba(28,187,129,0.7) 100%)',
  },
  {
    key: 'other',
    label: '运行中',
    color: '#60a5fa',
    bar: 'linear-gradient(180deg, rgba(96,165,250,0.94) 0%, rgba(72,141,226,0.7) 100%)',
  },
  {
    key: 'failed',
    label: '失败',
    color: '#f87171',
    bar: 'linear-gradient(180deg, rgba(248,113,113,0.94) 0%, rgba(226,72,72,0.7) 100%)',
  },
] as const;

export function TrendChart({ points }: { points: TrendPoint[] }) {
  const [hovered, setHovered] = useState<number | null>(null);
  const peak = points.reduce(
    (max, point) => Math.max(max, point.success + point.failed + point.other),
    0
  );
  const step = Math.max(1, Math.ceil(peak / 5));
  const scaleMax = step * 5;
  const ticks = [5, 4, 3, 2, 1, 0].map(index => step * index);
  const active = hovered === null ? null : points[hovered];
  const activeWidth = 100 / points.length;
  const activeLeft = hovered === null ? 0 : (hovered / points.length) * 100;
  const activeCenter = hovered === null ? 0 : ((hovered + 0.5) / points.length) * 100;
  const tooltipOnLeft = hovered !== null && hovered >= points.length - 4;

  return (
    <div className="flex h-full min-h-[16rem] flex-col px-4 pb-3">
      <div className="flex h-10 shrink-0 items-center justify-end">
        <div className="flex items-center gap-4 text-xs text-[var(--bm-text-muted)]">
          {series.map(item => (
            <span key={item.key} className="inline-flex items-center gap-1.5">
              <span className="h-2 w-2 rounded-full" style={{ background: item.color }} />
              {item.label}
            </span>
          ))}
        </div>
      </div>
      <div className="relative min-h-0 flex-1">
        <div className="absolute inset-x-0 top-0 bottom-6">
          {ticks.map(tick => (
            <div
              key={tick}
              className="absolute inset-x-0 flex translate-y-1/2 items-center"
              style={{ bottom: `${(tick / scaleMax) * 100}%` }}
            >
              <span className="w-9 shrink-0 pr-1 text-right text-[10px] leading-none tabular-nums text-[var(--bm-text-muted)]">
                {tick}
              </span>
              <svg
                className="bm-trend-grid-line h-px flex-1"
                viewBox="0 0 100 1"
                preserveAspectRatio="none"
                aria-hidden="true"
              >
                <line
                  x1="0"
                  y1="0.5"
                  x2="100"
                  y2="0.5"
                  stroke="currentColor"
                  strokeWidth="1"
                  vectorEffect="non-scaling-stroke"
                  shapeRendering="crispEdges"
                />
              </svg>
            </div>
          ))}
        </div>
        <div className="absolute top-0 bottom-6 left-9 right-0">
          {hovered !== null ? (
            <div
              className="bm-trend-highlight pointer-events-none absolute top-0 bottom-0 rounded-[3px]"
              style={{
                left: `calc(${activeLeft}% + 2px)`,
                width: `calc(${activeWidth}% - 4px)`,
              }}
            />
          ) : null}
          <div
            className="absolute inset-0 flex items-end justify-between gap-1"
            onMouseLeave={() => setHovered(null)}
          >
            {points.map((point, index) => {
              const total = point.success + point.other + point.failed;
              return (
                <div
                  key={point.date}
                  className="group relative flex h-full flex-1 cursor-pointer items-end justify-center"
                  onMouseEnter={() => setHovered(index)}
                >
                  <div
                    className="flex w-full max-w-[26px] flex-col-reverse overflow-hidden rounded-t-[3px]"
                    style={{ height: `${(total / scaleMax) * 100}%` }}
                  >
                    {total > 0 ? (
                      <>
                        <div
                          style={{
                            height: `${(point.success / total) * 100}%`,
                            background: series[0].bar,
                          }}
                        />
                        <div
                          style={{
                            height: `${(point.other / total) * 100}%`,
                            background: series[1].bar,
                          }}
                        />
                        <div
                          style={{
                            height: `${(point.failed / total) * 100}%`,
                            background: series[2].bar,
                          }}
                        />
                      </>
                    ) : null}
                  </div>
                  <div className="absolute inset-x-0 -bottom-6 text-center text-[10px] text-[var(--bm-text-muted)]">
                    {index % 2 === 0 ? point.date.slice(5) : ''}
                  </div>
                </div>
              );
            })}
          </div>
          {active ? (
            <div
              className="pointer-events-none absolute top-[45%] z-10 w-[132px] rounded-lg border p-2.5 shadow-lg"
              style={{
                left: `calc(${activeCenter}% ${tooltipOnLeft ? '-' : '+'} 8px)`,
                transform: `translate(${tooltipOnLeft ? '-100%' : '0'}, -50%)`,
                background: 'var(--bm-popover-bg)',
                borderColor: 'var(--bm-popover-border)',
                boxShadow: 'var(--bm-menu-shadow)',
              }}
            >
              <div className="mb-1.5 text-[11px] font-medium text-[var(--bm-text)]">
                {active.date}
              </div>
              <div className="space-y-1">
                {series.map(item => {
                  const count =
                    item.key === 'success'
                      ? active.success
                      : item.key === 'other'
                        ? active.other
                        : active.failed;
                  return (
                    <div key={item.key} className="flex items-center gap-2 text-[11px]">
                      <span className="h-2 w-2 rounded-full" style={{ background: item.color }} />
                      <span className="flex-1 text-[var(--bm-text-muted)]">{item.label}</span>
                      <span className="font-mono text-[var(--bm-text)]">{count}</span>
                    </div>
                  );
                })}
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}
