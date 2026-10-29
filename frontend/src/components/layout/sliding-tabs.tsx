import type { LucideIcon } from 'lucide-react';
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';

type SlidingTabItem = {
  id: string;
  label: string;
  icon: LucideIcon;
};

type Indicator = { x: number; y: number; width: number; height: number };

const STEP_DURATION_SECONDS = 1.0;
const MAX_DURATION_SECONDS = 2.0;

export function SlidingTabs({
  items,
  active,
  onChange,
}: {
  items: SlidingTabItem[];
  active: string;
  onChange: (id: string) => void;
}) {
  const containerRef = useRef<HTMLElement | null>(null);
  const buttonRefs = useRef(new Map<string, HTMLButtonElement>());
  const itemsRef = useRef(items);
  itemsRef.current = items;
  const activeIndex = Math.max(
    0,
    items.findIndex(item => item.id === active)
  );
  const activeIndexRef = useRef(activeIndex);
  activeIndexRef.current = activeIndex;
  const previousIndexRef = useRef(activeIndex);
  const hasMeasuredRef = useRef(false);
  const [indicator, setIndicator] = useState<Indicator | null>(null);
  const [duration, setDuration] = useState(0);

  const measure = useCallback((index: number, animate: boolean) => {
    const item = itemsRef.current[index];
    const button = item ? buttonRefs.current.get(item.id) : undefined;
    if (!button) return;
    const step = Math.abs(index - previousIndexRef.current);
    setIndicator({
      x: button.offsetLeft,
      y: button.offsetTop,
      width: button.offsetWidth,
      height: button.offsetHeight,
    });
    setDuration(
      animate && hasMeasuredRef.current
        ? Math.min(step * STEP_DURATION_SECONDS, MAX_DURATION_SECONDS)
        : 0
    );
    previousIndexRef.current = index;
    hasMeasuredRef.current = true;
  }, []);

  useLayoutEffect(() => {
    measure(activeIndex, true);
  }, [activeIndex, items.length, measure]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    const observer = new ResizeObserver(() => measure(activeIndexRef.current, false));
    observer.observe(container);
    return () => observer.disconnect();
  }, [measure]);

  return (
    <section
      ref={containerRef}
      className="bm-card-hover relative flex shrink-0 flex-wrap gap-2 rounded-xl p-3"
      style={{
        background: 'var(--bm-card)',
        border: '1px solid var(--bm-border)',
        boxShadow: 'var(--shadow-card)',
      }}
    >
      {indicator ? (
        <span
          aria-hidden="true"
          className="bm-config-nav-indicator"
          style={{
            width: indicator.width,
            height: indicator.height,
            transform: `translate(${indicator.x}px, ${indicator.y}px)`,
            transitionDuration: `${duration}s`,
          }}
        />
      ) : null}
      {items.map(item => {
        const Icon = item.icon;
        return (
          <button
            key={item.id}
            ref={element => {
              if (element) buttonRefs.current.set(item.id, element);
              else buttonRefs.current.delete(item.id);
            }}
            type="button"
            onClick={() => onChange(item.id)}
            className="bm-config-nav-card relative z-10 flex h-10 items-center gap-2 rounded-lg px-4 text-sm font-medium transition-colors"
            data-active={active === item.id}
            style={{
              color: active === item.id ? 'var(--bm-accent-hover)' : 'var(--bm-text-muted)',
            }}
          >
            <Icon size={16} />
            {item.label}
          </button>
        );
      })}
    </section>
  );
}
