import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export type BmTheme = 'dark' | 'light';

const themeStorageKey = 'bm.theme';

export function getInitialTheme(): BmTheme {
  if (typeof window === 'undefined') return 'dark';
  return window.localStorage.getItem(themeStorageKey) === 'light' ? 'light' : 'dark';
}

export function applyTheme(theme: BmTheme) {
  if (typeof document === 'undefined') return;
  document.documentElement.dataset.bmTheme = theme;
  document.documentElement.style.colorScheme = theme;
}

export function persistTheme(theme: BmTheme) {
  if (typeof window === 'undefined') return;
  window.localStorage.setItem(themeStorageKey, theme);
}

export function toggleTheme(theme: BmTheme): BmTheme {
  return theme === 'dark' ? 'light' : 'dark';
}
