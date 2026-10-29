import { useSyncExternalStore } from 'react';

let current = Date.now();
const listeners = new Set<() => void>();
let timer: number | undefined;

function subscribe(listener: () => void) {
  listeners.add(listener);
  if (timer === undefined) {
    timer = window.setInterval(() => {
      current = Date.now();
      for (const item of listeners) item();
    }, 1_000);
  }
  return () => {
    listeners.delete(listener);
    if (listeners.size === 0 && timer !== undefined) {
      window.clearInterval(timer);
      timer = undefined;
    }
  };
}

function snapshot() {
  return current;
}

// useNow 返回每秒推进的时间戳，供相对时间文案在两次数据刷新之间持续更新
export function useNow() {
  return useSyncExternalStore(subscribe, snapshot, snapshot);
}
