import { Link, useNavigate } from '@tanstack/react-router';
import { Eye, EyeOff, Loader2, Lock, Moon, Sun, User } from 'lucide-react';
import { type FormEvent, useCallback, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/feedback/app-tooltip';
import { BrandIcon } from '@/components/layout/brand-mark';
import { useBrandSettings } from '@/lib/branding';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  consumeAuthExpired,
  fetchCurrentUser,
  fetchPublicAuthProviders,
  getAuthToken,
  login,
  persistUser,
  type AuthSession,
  type PublicAuthProvider,
} from '@/lib/session';
import {
  bindWecomCallback,
  bindWecomSSO,
  loginWithWecomCallback,
  loginWithWecomSSO,
  WECOM_BIND_MESSAGE,
} from '@/lib/wecom';
import { WecomQrLogin, type WecomEmbedSuccess } from '@/features/auth/WecomQrLogin';
import { applyTheme, getInitialTheme, persistTheme, toggleTheme, type BmTheme } from '@/lib/utils';

export function LoginPage() {
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [provider, setProvider] = useState('local');
  const [providers, setProviders] = useState<PublicAuthProvider[]>([]);
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [theme, setTheme] = useState<BmTheme>(getInitialTheme);
  const brand = useBrandSettings();
  const toggleLabel = theme === 'dark' ? '切换浅色背景' : '切换深色背景';
  const [callbackParams] = useState(() => {
    const params = new URLSearchParams(typeof window === 'undefined' ? '' : window.location.search);
    return {
      code: params.get('code') ?? '',
      state: params.get('state') ?? '',
      ticket: params.get('ticket') ?? '',
    };
  });
  const isDirectCallback = Boolean(callbackParams.code && callbackParams.state);
  const isSSOCallback = Boolean(callbackParams.ticket);
  const isCallbackView = isDirectCallback || isSSOCallback;
  const isWecomProvider = provider === 'wecom';
  const [wecomAuthorizing, setWecomAuthorizing] = useState(false);

  const finishWecomLogin = useCallback(
    (session: AuthSession) => {
      toast.success(`欢迎回来，${session.user.displayName || session.user.username}`);
      window.history.replaceState({}, '', window.location.pathname);
      void navigate({ to: '/', replace: true });
    },
    [navigate]
  );

  async function handleWecomEmbedSuccess(payload: WecomEmbedSuccess) {
    setWecomAuthorizing(true);
    setError('');
    try {
      const session =
        payload.authMode === 'sso'
          ? await loginWithWecomSSO(payload.ticket ?? '')
          : await loginWithWecomCallback(payload.code ?? '', payload.state ?? '');
      finishWecomLogin(session);
    } catch (err) {
      setError(err instanceof Error ? err.message : '企业微信登录失败，请稍后重试');
    }
  }

  useEffect(() => {
    applyTheme(theme);
    persistTheme(theme);
  }, [theme]);

  useEffect(() => {
    if (consumeAuthExpired()) {
      toast.error('登录会话已过期，请重新登录');
    }
  }, []);

  useEffect(() => {
    if (isCallbackView) return;
    let cancelled = false;
    if (getAuthToken()) {
      void fetchCurrentUser()
        .then(user => {
          if (cancelled) return;
          persistUser(user);
          void navigate({ to: '/', replace: true });
        })
        .catch(() => undefined);
    }
    return () => {
      cancelled = true;
    };
  }, [isCallbackView, navigate]);

  useEffect(() => {
    let cancelled = false;
    fetchPublicAuthProviders()
      .then(response => {
        if (cancelled) return;
        setProviders(
          response.items.filter(
            item => item.enabled && (item.type === 'ldap' || item.type === 'wecom')
          )
        );
      })
      .catch(() => {
        if (!cancelled) {
          setProviders([]);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!isCallbackView) return;
    let cancelled = false;
    setLoading(true);
    (async () => {
      try {
        if (getAuthToken()) {
          if (isSSOCallback) {
            await bindWecomSSO(callbackParams.ticket);
          } else {
            await bindWecomCallback(callbackParams.code, callbackParams.state);
          }
          window.opener?.postMessage(
            { type: WECOM_BIND_MESSAGE, ok: true },
            window.location.origin
          );
          toast.success('企业微信绑定成功');
          window.history.replaceState({}, '', window.location.pathname);
          window.setTimeout(() => window.close(), 300);
          return;
        }
        const session = isSSOCallback
          ? await loginWithWecomSSO(callbackParams.ticket)
          : await loginWithWecomCallback(callbackParams.code, callbackParams.state);
        if (cancelled) return;
        finishWecomLogin(session);
        return;
      } catch (err) {
        const message = err instanceof Error ? err.message : '企业微信登录失败，请稍后重试';
        if (!cancelled) setError(message);
        if (getAuthToken()) {
          window.opener?.postMessage(
            { type: WECOM_BIND_MESSAGE, ok: false, message },
            window.location.origin
          );
          if (window.opener) {
            window.setTimeout(() => window.close(), 1500);
          }
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [isCallbackView, isSSOCallback, callbackParams, navigate, finishWecomLogin]);

  useEffect(() => {
    const handlePageShow = (event: PageTransitionEvent) => {
      if (!event.persisted) return;
      if (isCallbackView) {
        window.location.replace('/login');
        return;
      }
      setLoading(false);
    };
    window.addEventListener('pageshow', handlePageShow);
    return () => window.removeEventListener('pageshow', handlePageShow);
  }, [isCallbackView]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalizedUsername = username.trim();
    if (!normalizedUsername || !password) {
      setError('用户名或密码不能为空');
      return;
    }
    setLoading(true);
    setError('');
    try {
      const session = await login(normalizedUsername, password, provider);
      toast.success(`欢迎回来，${session.user.displayName || session.user.username}`);
      void navigate({ to: '/', replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : '登录失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  }

  if (isCallbackView || wecomAuthorizing) {
    return (
      <main
        data-cmp="Login"
        className="relative flex min-h-dvh items-center justify-center overflow-hidden px-4 py-8 sm:px-6"
        style={{
          background:
            'radial-gradient(circle at 50% 0%, rgba(59,130,246,0.24), transparent 30%), var(--bm-login-bg)',
          color: 'var(--bm-text)',
        }}
      >
        <div className="bm-login-grid absolute inset-0" aria-hidden="true" />
        <section className="bm-login-frame relative z-10 w-full max-w-[420px] rounded-[24px] p-1">
          <div
            className="flex flex-col items-center gap-4 rounded-[20px] px-8 py-10 text-center"
            style={{
              background: 'var(--bm-login-panel-bg)',
              border: '1px solid var(--bm-border)',
              backdropFilter: 'blur(18px)',
              boxShadow: 'var(--bm-login-panel-shadow)',
            }}
          >
            {error ? (
              <>
                <p className="text-sm leading-6" style={{ color: '#fca5a5' }} role="alert">
                  {error}
                </p>
                <button
                  type="button"
                  onClick={() => window.location.replace('/login')}
                  className="bm-action-button rounded-xl px-4 py-2 text-sm font-medium"
                  style={{
                    borderColor: 'var(--bm-border)',
                    background: 'var(--bm-control-bg)',
                    color: 'var(--bm-accent-text)',
                  }}
                >
                  返回登录
                </button>
              </>
            ) : (
              <>
                <Loader2 size={28} className="bm-spinner" />
                <p className="text-sm font-medium" style={{ color: 'var(--bm-text)' }}>
                  正在处理企业微信授权，请稍候…
                </p>
              </>
            )}
          </div>
        </section>
      </main>
    );
  }

  return (
    <main
      data-cmp="Login"
      className="relative flex min-h-dvh items-center justify-center overflow-hidden px-4 py-8 sm:px-6"
      style={{
        background:
          'radial-gradient(circle at 50% 0%, rgba(59,130,246,0.24), transparent 30%), radial-gradient(circle at 12% 22%, rgba(6,182,212,0.18), transparent 28%), radial-gradient(circle at 88% 82%, rgba(16,185,129,0.14), transparent 30%), var(--bm-login-bg)',
        color: 'var(--bm-text)',
      }}
    >
      <AppTooltip
        label={toggleLabel}
        placement="bottom"
        className="absolute right-5 top-5 z-20 sm:right-6 sm:top-6"
      >
        <button
          type="button"
          onClick={() => setTheme(toggleTheme)}
          className="bm-action-button grid h-11 w-11 place-items-center rounded-lg border transition-transform hover:-translate-y-0.5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--bm-accent-text)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--bm-login-bg)]"
          style={{
            background: 'var(--bm-control-bg)',
            borderColor: 'var(--bm-border)',
            color: 'var(--bm-text-muted)',
            boxShadow: 'var(--bm-menu-shadow)',
          }}
          aria-label={toggleLabel}
        >
          {theme === 'dark' ? <Sun size={18} /> : <Moon size={18} />}
        </button>
      </AppTooltip>
      <div className="bm-login-grid absolute inset-0" aria-hidden="true" />
      <div
        className="bm-login-orb absolute left-[-6rem] top-20 h-64 w-64 rounded-full"
        aria-hidden="true"
      />
      <div
        className="bm-login-orb absolute bottom-[-4rem] right-[-5rem] h-80 w-80 rounded-full"
        aria-hidden="true"
      />

      <section className="relative z-10 flex w-full max-w-[460px] flex-col items-center gap-5">
        <ConsoleBrand brand={brand} />
        <section
          className="bm-login-frame bm-login-reveal w-full rounded-[28px] p-1"
          style={{ animationDelay: '80ms' }}
        >
          <div
            className="rounded-[24px] p-6 sm:p-8"
            style={{
              background: 'var(--bm-login-panel-bg)',
              border: '1px solid var(--bm-border)',
              backdropFilter: 'blur(18px)',
              boxShadow: 'var(--bm-login-panel-shadow)',
            }}
          >
            <div className="mb-7 text-center">
              <h1
                className="text-2xl font-bold"
                style={{ color: 'var(--bm-text)', textShadow: '0 0 22px rgba(6,182,212,0.16)' }}
              >
                欢迎回来
              </h1>
              <p className="mt-2 text-sm" style={{ color: 'var(--bm-text-muted)' }}>
                登录您的账户以继续
              </p>
            </div>

            <form className="space-y-5" onSubmit={handleSubmit} noValidate>
              {providers.length > 0 ? (
                <LoginProviderSelect
                  value={provider}
                  providers={providers}
                  onChange={setProvider}
                />
              ) : null}
              {isWecomProvider ? (
                <WecomQrLogin onSuccess={payload => void handleWecomEmbedSuccess(payload)} />
              ) : (
                <>
                  <AuthInput id="username" label="用户名" icon={<User size={17} />}>
                    <input
                      id="username"
                      autoComplete="username"
                      value={username}
                      onChange={event => setUsername(event.target.value)}
                      className="h-12 w-full rounded-2xl py-3 pl-12 pr-4 text-sm outline-none transition-all"
                      style={inputStyle}
                      placeholder="请输入用户名"
                    />
                  </AuthInput>
                  <div>
                    <div className="mb-2 flex items-center justify-between">
                      <label className="block text-sm font-medium" htmlFor="password">
                        密码
                      </label>
                      <Link
                        to="/forgot-password"
                        className="text-xs font-semibold"
                        style={{ color: 'var(--bm-accent-text)' }}
                      >
                        忘记密码?
                      </Link>
                    </div>
                    <div className="relative">
                      <Lock
                        className="absolute left-4 top-1/2 -translate-y-1/2"
                        size={17}
                        style={{ color: 'var(--bm-text-muted)' }}
                      />
                      <input
                        id="password"
                        autoComplete="current-password"
                        type={showPassword ? 'text' : 'password'}
                        value={password}
                        onChange={event => setPassword(event.target.value)}
                        className="h-12 w-full rounded-2xl py-3 pl-12 pr-12 text-sm outline-none transition-all"
                        style={inputStyle}
                        placeholder="请输入密码"
                      />
                      <button
                        type="button"
                        onClick={() => setShowPassword(value => !value)}
                        className="absolute right-2 top-1/2 grid h-9 w-9 -translate-y-1/2 place-items-center rounded-xl"
                        style={{ color: 'var(--bm-text-muted)' }}
                        aria-label={showPassword ? '隐藏密码' : '显示密码'}
                      >
                        {showPassword ? <EyeOff size={17} /> : <Eye size={17} />}
                      </button>
                    </div>
                  </div>
                </>
              )}
              <div aria-live="polite" className="min-h-6">
                {error ? (
                  <p
                    role="alert"
                    className="rounded-xl px-3 py-2 text-xs"
                    style={{
                      background: 'rgba(239,68,68,0.12)',
                      border: '1px solid rgba(239,68,68,0.28)',
                      color: '#fca5a5',
                    }}
                  >
                    {error}
                  </p>
                ) : null}
              </div>
              {!isWecomProvider ? (
                <button
                  type="submit"
                  disabled={loading}
                  className="bm-login-submit flex h-12 w-full items-center justify-center gap-2 rounded-2xl text-sm font-semibold transition-all disabled:cursor-not-allowed disabled:opacity-60"
                  style={{
                    background: 'linear-gradient(135deg, #2563eb, #06b6d4)',
                    color: '#fff',
                    boxShadow: '0 18px 48px rgba(37,99,235,0.35)',
                  }}
                >
                  {loading ? <Loader2 size={17} className="bm-spinner" /> : null}
                  {loading ? '处理中...' : '登录'}
                </button>
              ) : null}
            </form>
          </div>
        </section>
        <p className="text-xs" style={{ color: 'var(--bm-text-muted)' }}>
          © 2026 {brand.siteName}. Secure backup operations monitoring console.
        </p>
      </section>
    </main>
  );
}

const inputStyle = {
  background: 'var(--bm-control-bg)',
  border: '1px solid var(--bm-border)',
  color: 'var(--bm-text)',
};

function ConsoleBrand({ brand }: { brand: ReturnType<typeof useBrandSettings> }) {
  return (
    <div className="bm-login-reveal flex flex-col items-center text-center">
      <BrandIcon
        iconData={brand.iconData}
        alt={brand.loginName}
        className="bm-brand-float mb-4 h-20 w-20 drop-shadow-[0_18px_42px_rgba(6,182,212,0.18)]"
      />
      <p className="bm-gradient-text text-2xl font-bold tracking-wide">{brand.loginName}</p>
      <p
        className="mt-1 text-xs uppercase tracking-[0.28em]"
        style={{ color: 'var(--bm-text-muted)' }}
      >
        {brand.appSubtitle}
      </p>
    </div>
  );
}

function AuthInput({
  id,
  label,
  icon,
  children,
}: {
  id: string;
  label: string;
  icon: ReactNode;
  children: ReactNode;
}) {
  return (
    <div>
      <label className="mb-2 block text-sm font-medium" htmlFor={id}>
        {label}
      </label>
      <div className="relative">
        <span
          className="absolute left-4 top-1/2 -translate-y-1/2"
          style={{ color: 'var(--bm-text-muted)' }}
        >
          {icon}
        </span>
        {children}
      </div>
    </div>
  );
}

function LoginProviderSelect({
  value,
  providers,
  onChange,
}: {
  value: string;
  providers: PublicAuthProvider[];
  onChange: (value: string) => void;
}) {
  const options = [
    { id: 'local', name: '本地账号' },
    ...providers.map(item => ({
      id: item.id,
      name: item.name || (item.type === 'wecom' ? '企业微信' : 'AD/LDAP'),
    })),
  ];
  return (
    <div>
      <label className="mb-2 block text-sm font-medium" htmlFor="login-provider-select">
        登录方式
      </label>
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger id="login-provider-select" className="h-12 rounded-2xl px-4 font-normal">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {options.map(item => (
            <SelectItem key={item.id} value={item.id} className="font-normal">
              {item.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
