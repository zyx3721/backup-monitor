import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  HeadContent,
  Link,
  Outlet,
  Scripts,
  createRootRouteWithContext,
  useLocation,
  useRouter,
} from '@tanstack/react-router';
import {
  AlertTriangle,
  ArrowLeft,
  CheckCircle2,
  Home,
  Info,
  Loader2,
  RefreshCcw,
  SearchX,
  ShieldAlert,
  XCircle,
} from 'lucide-react';
import { type ReactNode, useEffect, useState } from 'react';
import { BootScreen } from '@/components/layout/boot-screen';
import { initializeBrandSettings, type BrandSettings } from '@/lib/branding';
import { serverEnv } from '@/lib/server-env';
import { Toaster } from '@/components/ui/sonner';

// CaretCapableDocument 为跨浏览器补齐按坐标取插入点的方法
type CaretCapableDocument = Document & {
  caretRangeFromPoint?: (x: number, y: number) => Range | null;
  caretPositionFromPoint?: (x: number, y: number) => { offsetNode: Node; offset: number } | null;
};
type WordCapableSelection = Selection & {
  modify?: (alter: string, direction: string, granularity: string) => void;
};

// ToastTextSelectionBridge 在提示文本上拦截按下与拖动，用命中测试还原选区，使弹窗打开时仍可选择复制提示内容
function ToastTextSelectionBridge() {
  useEffect(() => {
    let dragging = false;
    let dragRoot: HTMLElement | null = null;
    let anchorRange: Range | null = null;
    let lastDownAt = 0;
    let lastDownX = 0;
    let lastDownY = 0;

    const contentOf = (target: EventTarget | null): HTMLElement | null => {
      if (!(target instanceof HTMLElement)) return null;
      if (target.closest('button')) return null;
      const toast = target.closest<HTMLElement>('[data-sonner-toast]');
      if (!toast) return null;
      const content = toast.querySelector<HTMLElement>('[data-content]');
      return content ?? toast;
    };

    const caretAt = (x: number, y: number, root: HTMLElement): Range | null => {
      const doc = document as CaretCapableDocument;
      let range: Range | null = null;
      if (doc.caretRangeFromPoint) {
        range = doc.caretRangeFromPoint(x, y);
      } else if (doc.caretPositionFromPoint) {
        const position = doc.caretPositionFromPoint(x, y);
        if (position) {
          range = document.createRange();
          range.setStart(position.offsetNode, position.offset);
          range.collapse(true);
        }
      }
      if (!range || !root.contains(range.startContainer)) return null;
      return range;
    };

    const applyRange = (range: Range) => {
      const selection = window.getSelection();
      if (!selection) return;
      selection.removeAllRanges();
      selection.addRange(range);
    };

    const selectBetween = (from: Range, to: Range) => {
      const range = document.createRange();
      if (from.compareBoundaryPoints(Range.START_TO_START, to) <= 0) {
        range.setStart(from.startContainer, from.startOffset);
        range.setEnd(to.startContainer, to.startOffset);
      } else {
        range.setStart(to.startContainer, to.startOffset);
        range.setEnd(from.startContainer, from.startOffset);
      }
      applyRange(range);
    };

    const onPointerDown = (event: PointerEvent) => {
      const content = contentOf(event.target);
      if (!content) return;
      const caret = caretAt(event.clientX, event.clientY, content);
      if (!caret) return;
      event.preventDefault();
      const isWordPick =
        Date.now() - lastDownAt < 350 &&
        Math.hypot(event.clientX - lastDownX, event.clientY - lastDownY) < 8;
      lastDownAt = Date.now();
      lastDownX = event.clientX;
      lastDownY = event.clientY;
      if (isWordPick) {
        const selection = window.getSelection() as WordCapableSelection | null;
        applyRange(caret);
        selection?.modify?.('extend', 'forward', 'word');
        selection?.modify?.('extend', 'backward', 'word');
        dragging = false;
        anchorRange = null;
        return;
      }
      dragging = true;
      dragRoot = content;
      anchorRange = caret.cloneRange();
      applyRange(caret);
    };

    const onPointerMove = (event: PointerEvent) => {
      if (!dragging || !anchorRange || !dragRoot) return;
      const caret = caretAt(event.clientX, event.clientY, dragRoot);
      if (!caret) return;
      selectBetween(anchorRange, caret);
    };

    const onPointerEnd = () => {
      dragging = false;
      anchorRange = null;
      dragRoot = null;
    };

    document.addEventListener('pointerdown', onPointerDown, true);
    document.addEventListener('pointermove', onPointerMove, true);
    document.addEventListener('pointerup', onPointerEnd, true);
    document.addEventListener('pointercancel', onPointerEnd, true);
    return () => {
      document.removeEventListener('pointerdown', onPointerDown, true);
      document.removeEventListener('pointermove', onPointerMove, true);
      document.removeEventListener('pointerup', onPointerEnd, true);
      document.removeEventListener('pointercancel', onPointerEnd, true);
    };
  }, []);
  return null;
}

import appCss from '../styles.css?url';

const themeScript = `
(function () {
  var root = document.documentElement;
  try {
    var theme = localStorage.getItem('bm.theme') === 'light' ? 'light' : 'dark';
    root.dataset.bmTheme = theme;
    root.style.colorScheme = theme;
  } catch (_) {
    root.dataset.bmTheme = 'dark';
    root.style.colorScheme = 'dark';
  } finally {
    root.style.visibility = 'visible';
  }
})();
`;

// loadPublicBrand 由 SSR 首屏实时拉取公开品牌，客户端 loader 重跑时返回 undefined 以保留已同步快照
async function loadPublicBrand(): Promise<Partial<BrandSettings> | undefined> {
  if (typeof window !== 'undefined') return undefined;
  const configured = await serverEnv('SSR_API_ORIGIN');
  const devOrigin = import.meta.env.DEV ? import.meta.env.VITE_API_BASE_URL : undefined;
  const apiOrigin = configured || devOrigin || 'http://127.0.0.1:8080';
  try {
    const response = await fetch(`${apiOrigin}/api/public/base`, {
      signal: AbortSignal.timeout(2500),
    });
    if (!response.ok) {
      console.warn(
        `[bm-ssr] fetch brand from ${apiOrigin}/api/public/base failed with status ${response.status}`
      );
      return {};
    }
    return (await response.json()) as Partial<BrandSettings>;
  } catch (error) {
    console.warn(
      `[bm-ssr] fetch brand failed: ${error instanceof Error ? error.message : String(error)}`
    );
    return {};
  }
}

function NotFoundComponent() {
  return (
    <main className="relative grid min-h-screen place-items-center overflow-hidden bg-[var(--bm-bg)] px-4 py-10 sm:px-6">
      <div className="pointer-events-none absolute inset-x-0 top-1/2 h-px -translate-y-1/2 bg-gradient-to-r from-transparent via-[rgba(96,165,250,0.28)] to-transparent" />
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(rgba(148,163,184,0.055)_1px,transparent_1px),linear-gradient(90deg,rgba(148,163,184,0.045)_1px,transparent_1px)] bg-[size:46px_46px] [mask-image:radial-gradient(circle_at_50%_50%,#000_0%,transparent_70%)]" />

      <section className="bm-surface-3d w-full max-w-4xl rounded-2xl px-6 py-8 text-center sm:px-10 sm:py-10 lg:px-12">
        <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl border border-[rgba(96,165,250,0.34)] bg-[rgba(59,130,246,0.12)] text-[var(--bm-accent-hover)] shadow-[0_16px_36px_rgba(37,99,235,0.18)]">
          <SearchX size={28} aria-hidden="true" />
        </div>
        <div className="bm-gradient-text mt-7 text-[clamp(5.25rem,15vw,9rem)] font-black leading-none tracking-normal drop-shadow-[0_14px_32px_rgba(37,99,235,0.18)]">
          404
        </div>
        <h1 className="bm-gradient-text mt-4 text-2xl font-semibold tracking-normal sm:text-3xl">
          页面不存在
        </h1>
        <p className="mx-auto mt-3 max-w-xl text-sm leading-6 text-muted-foreground sm:text-base">
          当前访问的页面不存在，或已经被移动。请返回首页后重新选择需要访问的功能。
        </p>
        <div className="mt-8 flex flex-wrap justify-center gap-3">
          <Link
            to="/"
            className="bm-login-submit inline-flex h-10 items-center justify-center gap-2 rounded-lg px-5 text-sm font-medium text-white"
          >
            <Home size={16} aria-hidden="true" />
            返回首页
          </Link>
          <button
            type="button"
            onClick={() => history.back()}
            className="bm-action-button inline-flex h-10 items-center justify-center gap-2 rounded-lg border px-5 text-sm font-medium"
          >
            <ArrowLeft size={16} aria-hidden="true" />
            返回上一页
          </button>
        </div>
      </section>
    </main>
  );
}

function ErrorComponent({ error, reset }: { error: unknown; reset: () => void }) {
  console.error(error);
  const router = useRouter();

  return (
    <main className="relative grid min-h-screen place-items-center overflow-hidden bg-[var(--bm-bg)] px-4 py-10 sm:px-6">
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(rgba(148,163,184,0.052)_1px,transparent_1px),linear-gradient(90deg,rgba(148,163,184,0.04)_1px,transparent_1px)] bg-[size:44px_44px] [mask-image:radial-gradient(circle_at_50%_46%,#000_0%,transparent_72%)]" />

      <section className="bm-surface-3d w-full max-w-4xl rounded-2xl px-6 py-8 text-center sm:px-10 sm:py-10">
        <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl border border-[rgba(239,68,68,0.34)] bg-[rgba(239,68,68,0.11)] text-[var(--bm-status-red-text)] shadow-[0_16px_36px_rgba(239,68,68,0.14)]">
          <ShieldAlert size={28} aria-hidden="true" />
        </div>
        <h1 className="mt-6 text-2xl font-semibold tracking-tight text-foreground sm:text-3xl">
          页面加载失败
        </h1>
        <p className="mx-auto mt-3 max-w-xl text-sm leading-6 text-muted-foreground sm:text-base">
          页面加载过程中出现异常，可以重试或返回首页。
        </p>
        {import.meta.env.DEV ? (
          <pre className="bm-error-detail-pre bm-hidden-scrollbar mx-auto mt-6 max-h-80 w-full max-w-3xl overflow-auto rounded-xl border p-5 text-left text-xs leading-5 sm:text-[0.8125rem]">
            {(error as Error)?.stack ?? (error as Error)?.message}
          </pre>
        ) : null}
        <div className="mt-8 flex flex-wrap justify-center gap-3">
          <button
            type="button"
            onClick={() => {
              router.invalidate();
              reset();
            }}
            className="bm-login-submit inline-flex h-10 items-center justify-center gap-2 rounded-lg px-5 text-sm font-medium text-white"
          >
            <RefreshCcw size={16} aria-hidden="true" />
            重试
          </button>
          <a
            href="/"
            className="bm-action-button inline-flex h-10 items-center justify-center gap-2 rounded-lg border px-5 text-sm font-medium"
          >
            <Home size={16} aria-hidden="true" />
            返回首页
          </a>
        </div>
      </section>
    </main>
  );
}

export const Route = createRootRouteWithContext<{ queryClient: QueryClient }>()({
  loader: () => loadPublicBrand(),
  head: ({ match }) => {
    const brand = match.loaderData as Partial<BrandSettings> | undefined;
    const siteName = brand?.siteName?.trim() || 'Backup Monitor';
    const icon = brand?.iconData?.trim() || '/favicon.svg';
    return {
      meta: [
        { charSet: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { title: siteName },
        {
          name: 'description',
          content: 'Backup task and host monitoring platform',
        },
        { name: 'author', content: siteName },
        { property: 'og:title', content: `${siteName} 控制台` },
        {
          property: 'og:description',
          content: 'Backup task and host monitoring platform',
        },
        { property: 'og:type', content: 'website' },
        { name: 'twitter:card', content: 'summary' },
        { name: 'twitter:title', content: `${siteName} 控制台` },
        {
          name: 'twitter:description',
          content: 'Backup task and host monitoring platform',
        },
      ],
      links: [
        {
          rel: 'stylesheet',
          href: appCss,
        },
        {
          rel: 'icon',
          href: icon,
          type: 'image/svg+xml',
        },
      ],
    };
  },
  shellComponent: RootShell,
  component: RootComponent,
  notFoundComponent: NotFoundComponent,
  errorComponent: ErrorComponent,
});

function RootShell({ children }: { children: ReactNode }) {
  const initialStyle =
    typeof document === 'undefined' ? { visibility: 'hidden' as const } : undefined;

  return (
    <html lang="zh-CN" data-bm-theme="dark" style={initialStyle} suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
        <HeadContent />
      </head>
      <body>
        {children}
        <Scripts />
      </body>
    </html>
  );
}

function RootComponent() {
  const { queryClient } = Route.useRouteContext();
  const brand = Route.useLoaderData() as Partial<BrandSettings> | undefined;
  if (brand !== undefined) initializeBrandSettings(brand);
  const location = useLocation();
  const search = (location.search ?? {}) as Record<string, string | undefined>;
  const skipBoot =
    location.pathname.endsWith('/wecom-qr-callback') ||
    (location.pathname.endsWith('/login') && Boolean(search.ticket || search.code));
  const [booting, setBooting] = useState(!skipBoot);

  useEffect(() => {
    if (!booting) return;
    const timer = window.setTimeout(() => setBooting(false), 520);
    return () => window.clearTimeout(timer);
  }, [booting]);

  return (
    <QueryClientProvider client={queryClient}>
      <ToastTextSelectionBridge />
      {booting ? <BootScreen /> : <Outlet />}
      <Toaster
        position="top-right"
        theme="system"
        closeButton
        expand
        visibleToasts={8}
        gap={10}
        duration={3000}
        offset={{ top: 28, right: 20 }}
        mobileOffset={{ top: 16, right: 12, left: 12 }}
        swipeDirections={[]}
        icons={{
          success: <CheckCircle2 size={18} />,
          error: <XCircle size={18} />,
          warning: <AlertTriangle size={18} />,
          info: <Info size={18} />,
          loading: <Loader2 className="bm-toast-spin" size={18} />,
        }}
      />
    </QueryClientProvider>
  );
}
