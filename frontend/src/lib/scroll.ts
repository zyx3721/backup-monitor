// 重置主内容区滚动位置，供路由切换与页内标签切换调用。
export function resetContentScroll() {
  if (typeof document === 'undefined') return;
  document.querySelector<HTMLElement>('[data-bm-scroll-root]')?.scrollTo({ top: 0 });
}
