export function renderErrorPage(): string {
  return `<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="utf-8" />
    <title>页面加载失败</title>
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <style>
      :root {
        color-scheme: dark;
        --bm-text: #e2e8f0;
        --bm-text-muted: #94a3b8;
        --bm-border: rgba(76, 103, 150, 0.5);
        --bm-card: linear-gradient(145deg, rgba(20, 34, 58, 0.96), rgba(8, 15, 29, 0.98));
        --bm-bg:
          radial-gradient(circle at 14% 8%, rgba(59, 130, 246, 0.13), transparent 32%),
          radial-gradient(circle at 92% 10%, rgba(6, 182, 212, 0.09), transparent 28%),
          linear-gradient(135deg, #050914 0%, #0b1220 46%, #07101d 100%);
      }
      * { box-sizing: border-box; }
      body {
        display: grid;
        place-items: center;
        min-height: 100vh;
        margin: 0;
        padding: 2.5rem 1.5rem;
        overflow: hidden;
        background: var(--bm-bg);
        color: var(--bm-text);
        font: 15px/1.6 system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
      }
      body::before {
        position: fixed;
        inset: 0;
        pointer-events: none;
        content: '';
        background-image:
          linear-gradient(rgba(148, 163, 184, 0.052) 1px, transparent 1px),
          linear-gradient(90deg, rgba(148, 163, 184, 0.04) 1px, transparent 1px);
        background-size: 44px 44px;
        mask-image: radial-gradient(circle at 50% 46%, #000 0%, transparent 72%);
      }
      .card {
        position: relative;
        width: min(100%, 56rem);
        overflow: hidden;
        border: 1px solid var(--bm-border);
        border-radius: 1rem;
        background: var(--bm-card);
        padding: 2.5rem;
        text-align: center;
        box-shadow:
          0 32px 70px rgba(0, 0, 0, 0.5),
          0 14px 30px rgba(2, 8, 23, 0.4),
          0 1px 0 rgba(255, 255, 255, 0.16) inset,
          0 -22px 42px rgba(0, 0, 0, 0.5) inset;
      }
      .card::before {
        position: absolute;
        inset: 0;
        pointer-events: none;
        content: '';
        background:
          linear-gradient(135deg, rgba(255, 255, 255, 0.12), transparent 34%),
          radial-gradient(circle at 18% 0%, rgba(96, 165, 250, 0.14), transparent 38%);
      }
      .content { position: relative; z-index: 1; }
      .icon {
        display: grid;
        place-items: center;
        width: 3.5rem;
        height: 3.5rem;
        margin: 0 auto 1.5rem;
        border: 1px solid rgba(239, 68, 68, 0.34);
        border-radius: 1rem;
        background: rgba(239, 68, 68, 0.11);
        color: #fca5a5;
        box-shadow: 0 16px 36px rgba(239, 68, 68, 0.14);
      }
      h1 { margin: 0; font-size: clamp(1.5rem, 3vw, 1.875rem); line-height: 1.2; }
      p { max-width: 36rem; margin: 0.75rem auto 0; color: var(--bm-text-muted); }
      .actions { display: flex; gap: 0.75rem; justify-content: center; flex-wrap: wrap; margin-top: 2rem; }
      a, button {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        min-height: 2.5rem;
        padding: 0.55rem 1.25rem;
        border-radius: 0.5rem;
        font: inherit;
        font-weight: 600;
        cursor: pointer;
        text-decoration: none;
        transition: transform 0.18s ease, border-color 0.18s ease, filter 0.18s ease;
      }
      a:hover, button:hover { transform: translateY(-1px); }
      .primary {
        border: 0;
        background: linear-gradient(135deg, #3b82f6, #06b6d4);
        color: #fff;
        box-shadow: 0 18px 36px rgba(37, 99, 235, 0.26);
      }
      .primary:hover { filter: brightness(1.06); }
      .secondary {
        border: 1px solid var(--bm-border);
        background: rgba(255, 255, 255, 0.04);
        color: var(--bm-text);
      }
      .secondary:hover { border-color: rgba(96, 165, 250, 0.7); }
      @media (max-width: 640px) {
        body { padding: 1rem; }
        .card { padding: 2rem 1.25rem; }
      }
    </style>
  </head>
  <body>
    <div class="card">
      <div class="content">
        <div class="icon" aria-hidden="true">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none">
            <path d="M12 9v4" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            <path d="M12 17h.01" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0Z" stroke="currentColor" stroke-width="2" stroke-linejoin="round"/>
          </svg>
        </div>
        <h1>页面加载失败</h1>
        <p>页面加载过程中出现异常，可以重试或返回首页。</p>
        <div class="actions">
          <button class="primary" onclick="location.reload()">重试</button>
          <a class="secondary" href="/">返回首页</a>
        </div>
      </div>
    </div>
  </body>
</html>`;
}
