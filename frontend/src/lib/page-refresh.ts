import { useEffect } from 'react';

// 页面刷新事件协议：顶栏刷新按钮通过 dispatchPageRefresh 触发，各页面用 usePageRefresh 注册重载任务

export const PAGE_REFRESH_EVENT = 'bm:refresh';

type PageRefreshEvent = CustomEvent<{ tasks: Promise<unknown>[] }>;

// dispatchPageRefresh 派发页面刷新事件并等待所有已注册的页面刷新任务完成
export async function dispatchPageRefresh() {
  const event: PageRefreshEvent = new CustomEvent(PAGE_REFRESH_EVENT, {
    detail: { tasks: [] },
  });
  window.dispatchEvent(event);
  await Promise.allSettled(event.detail.tasks);
}

// usePageRefresh 注册当前页面的数据重载任务，顶栏刷新按钮触发时重新执行 onRefresh
export function usePageRefresh(onRefresh: () => unknown) {
  useEffect(() => {
    const handler = (event: Event) => {
      (event as PageRefreshEvent).detail?.tasks?.push(Promise.resolve(onRefresh()));
    };
    window.addEventListener(PAGE_REFRESH_EVENT, handler);
    return () => window.removeEventListener(PAGE_REFRESH_EVENT, handler);
  }, [onRefresh]);
}
