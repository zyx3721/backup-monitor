import { type ReactNode, useRef, useState } from 'react';
import { createPortal } from 'react-dom';

type TooltipPlacement = 'top' | 'bottom' | 'left' | 'right';
type TooltipAlign = 'start' | 'center' | 'end';

export function AppTooltip({
  label,
  children,
  className,
  placement,
  align = 'center',
  disabled,
  wrap = false,
}: {
  label?: ReactNode;
  children: ReactNode;
  className?: string;
  placement?: TooltipPlacement;
  align?: TooltipAlign;
  disabled?: boolean;
  wrap?: boolean;
}) {
  const triggerRef = useRef<HTMLSpanElement | null>(null);
  const [tooltip, setTooltip] = useState({
    open: false,
    top: 0,
    left: 0,
    placement: 'top' as TooltipPlacement,
    align,
  });
  const active = !disabled && Boolean(label);

  function showTooltip() {
    if (!active || !triggerRef.current) return;
    const rect = triggerRef.current.getBoundingClientRect();
    const gap = 8;
    const nextPlacement =
      placement ??
      (rect.top < 50 && window.innerHeight - rect.bottom > rect.top ? 'bottom' : 'top');
    const minLeft = 72;
    const maxLeft = window.innerWidth - minLeft;
    const centeredLeft = Math.min(maxLeft, Math.max(minLeft, rect.left + rect.width / 2));
    const edgeLeft = align === 'start' ? rect.left : align === 'end' ? rect.right : centeredLeft;
    const left =
      nextPlacement === 'right'
        ? rect.right + gap
        : nextPlacement === 'left'
          ? rect.left - gap
          : edgeLeft;
    const top =
      nextPlacement === 'top'
        ? rect.top - gap
        : nextPlacement === 'bottom'
          ? rect.bottom + gap
          : rect.top + rect.height / 2;
    setTooltip({ open: true, top, left, placement: nextPlacement, align });
  }

  function hideTooltip() {
    setTooltip(current => ({ ...current, open: false }));
  }

  const horizontalTransform =
    tooltip.align === 'start' ? '0' : tooltip.align === 'end' ? '-100%' : '-50%';
  const transform =
    tooltip.placement === 'top'
      ? `translate(${horizontalTransform}, -100%)`
      : tooltip.placement === 'bottom'
        ? `translate(${horizontalTransform}, 0)`
        : tooltip.placement === 'right'
          ? 'translate(0, -50%)'
          : 'translate(-100%, -50%)';

  const bubble =
    active && tooltip.open && typeof document !== 'undefined'
      ? createPortal(
          <span
            className="pointer-events-none fixed text-left shadow-2xl"
            style={{ left: tooltip.left, top: tooltip.top, transform, zIndex: 1000 }}
          >
            <span
              className={`block rounded-lg border px-2.5 py-1.5 text-xs font-semibold ${
                wrap ? 'max-w-xs whitespace-pre-wrap text-left leading-5' : 'whitespace-nowrap'
              }`}
              style={{
                background: 'var(--bm-popover-bg)',
                borderColor: 'var(--bm-popover-border)',
                color: 'var(--bm-text)',
                boxShadow: 'var(--bm-menu-shadow)',
              }}
            >
              {label}
            </span>
          </span>,
          document.body
        )
      : null;

  return (
    <span
      ref={triggerRef}
      className={className}
      onMouseEnter={showTooltip}
      onMouseLeave={hideTooltip}
      onFocus={showTooltip}
      onBlur={hideTooltip}
    >
      {children}
      {bubble}
    </span>
  );
}
