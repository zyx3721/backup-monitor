import { createFileRoute } from '@tanstack/react-router';
import { useEffect } from 'react';
import { Loader2 } from 'lucide-react';

// WecomQrCallbackPage 企微扫码中转页：iframe 内静默占位由父页面接管，顶层整页回跳转发登录页现有回调逻辑
function WecomQrCallbackPage() {
  const embedded = typeof window !== 'undefined' && window.self !== window.top;
  useEffect(() => {
    if (!embedded) {
      window.location.replace('/login' + window.location.search);
    }
  }, [embedded]);
  return (
    <main
      className="grid min-h-screen place-items-center px-6"
      style={{ background: 'var(--bm-login-bg)', color: 'var(--bm-text)' }}
    >
      <div
        className="flex flex-col items-center gap-3 rounded-3xl px-10 py-8 text-center"
        style={{
          background: 'var(--bm-login-panel-bg)',
          border: '1px solid var(--bm-border)',
          boxShadow: 'var(--bm-login-panel-shadow)',
        }}
      >
        <Loader2 size={26} className="bm-spinner" style={{ color: 'var(--bm-accent-text)' }} />
        <p className="text-sm font-medium" style={{ color: 'var(--bm-text)' }}>
          {embedded ? '扫码成功，正在登录' : '企业微信授权回跳处理中'}
        </p>
        <p className="text-xs" style={{ color: 'var(--bm-text-muted)' }}>
          {embedded ? '请勿关闭当前页面' : '即将返回登录页继续'}
        </p>
      </div>
    </main>
  );
}

export const Route = createFileRoute('/wecom-qr-callback')({
  component: WecomQrCallbackPage,
});
