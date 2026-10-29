import { Link, useNavigate } from '@tanstack/react-router';
import {
  ArrowLeft,
  Eye,
  EyeOff,
  KeyRound,
  Loader2,
  Lock,
  Mail,
  Moon,
  RefreshCw,
  Sun,
  User,
} from 'lucide-react';
import { type FormEvent, type ReactNode, useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/feedback/app-tooltip';
import { BrandIcon } from '@/components/layout/brand-mark';
import { useBrandSettings } from '@/lib/branding';
import {
  confirmPasswordReset,
  fetchPasswordResetCaptcha,
  sendPasswordResetCode,
  verifyPasswordResetIdentity,
  type PasswordResetCaptcha,
  type PasswordResetChannel,
} from '@/lib/password-reset';
import { applyTheme, getInitialTheme, persistTheme, toggleTheme, type BmTheme } from '@/lib/utils';

export function ForgotPasswordPage() {
  const navigate = useNavigate();
  const verifyEmailRef = useRef<HTMLInputElement | null>(null);
  const [step, setStep] = useState<'identity' | 'reset'>('identity');
  const [username, setUsername] = useState('');
  const [captcha, setCaptcha] = useState<PasswordResetCaptcha | null>(null);
  const [captchaAnswer, setCaptchaAnswer] = useState('');
  const [verificationToken, setVerificationToken] = useState('');
  const [channel, setChannel] = useState<PasswordResetChannel | null>(null);
  const [verifyEmail, setVerifyEmail] = useState('');
  const [code, setCode] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [cooldown, setCooldown] = useState(0);
  const [busy, setBusy] = useState('');
  const [error, setError] = useState('');
  const [theme, setTheme] = useState<BmTheme>(getInitialTheme);
  const brand = useBrandSettings();

  useEffect(() => {
    applyTheme(theme);
    persistTheme(theme);
  }, [theme]);

  useEffect(() => {
    void loadCaptcha();
  }, []);

  useEffect(() => {
    if (!captcha?.expiresAt || step !== 'identity') return;
    const expiresAt = new Date(captcha.expiresAt).getTime();
    const delay = Number.isNaN(expiresAt) ? 60_000 : Math.max(1000, expiresAt - Date.now());
    const timer = window.setTimeout(() => void loadCaptcha(), delay);
    return () => window.clearTimeout(timer);
  }, [captcha?.expiresAt, step]);

  useEffect(() => {
    if (cooldown <= 0) return;
    const timer = window.setInterval(() => setCooldown(value => Math.max(0, value - 1)), 1000);
    return () => window.clearInterval(timer);
  }, [cooldown]);

  useEffect(() => {
    if (step !== 'reset') return;
    window.requestAnimationFrame(() => verifyEmailRef.current?.focus());
  }, [step]);

  async function loadCaptcha() {
    setBusy('captcha');
    setCaptchaAnswer('');
    try {
      setCaptcha(await fetchPasswordResetCaptcha());
    } catch (err) {
      setError(err instanceof Error ? err.message : '生成验证码失败');
    } finally {
      setBusy('');
    }
  }

  async function verifyIdentity(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalizedUsername = username.trim();
    if (!normalizedUsername) {
      setError('请输入用户名');
      return;
    }
    if (!captcha || !captchaAnswer.trim()) {
      setError('请输入验证码');
      return;
    }
    setBusy('verify');
    setError('');
    try {
      const result = await verifyPasswordResetIdentity({
        username: normalizedUsername,
        captchaToken: captcha.token,
        captchaAnswer: captchaAnswer.trim(),
      });
      const email = result.channels.find(item => item.id === 'email');
      if (!email) {
        setError('当前没有可用的找回密码媒介');
        return;
      }
      setUsername(normalizedUsername);
      setVerificationToken(result.verificationToken);
      setChannel(email);
      setStep('reset');
    } catch (err) {
      setError(err instanceof Error ? err.message : '校验失败，请稍后重试');
      await loadCaptcha();
    } finally {
      setBusy('');
    }
  }

  async function resend() {
    if (cooldown > 0) {
      setError(`验证码已发送，请于 ${cooldown} 秒后再试`);
      return;
    }
    if (!verifyEmail.trim()) {
      setError('请输入验证邮箱');
      return;
    }
    setBusy('send');
    setError('');
    try {
      const result = await sendPasswordResetCode({
        username,
        verificationToken,
        channel: channel?.id ?? 'email',
        verifyEmail: verifyEmail.trim(),
      });
      setCooldown(result.cooldownSeconds);
      toast.success('找回密码验证码已发送');
    } catch (err) {
      setError(err instanceof Error ? err.message : '发送验证码失败');
    } finally {
      setBusy('');
    }
  }

  async function confirm(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!code.trim()) return setError('请输入找回密码验证码');
    if (newPassword.length < 6 || confirmPassword.length < 6) {
      return setError('密码至少 6 个字符');
    }
    if (newPassword !== confirmPassword) return setError('新密码与确认密码不一致');
    setBusy('confirm');
    setError('');
    try {
      await confirmPasswordReset({
        username,
        verificationToken,
        code: code.trim(),
        newPassword,
        confirmPassword,
      });
      toast.success('密码已重置，请重新登录');
      await navigate({ to: '/login', replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : '密码重置失败');
    } finally {
      setBusy('');
    }
  }

  const toggleLabel = theme === 'dark' ? '切换浅色背景' : '切换深色背景';
  const captchaText = captcha?.question.replace(/\s*=\s*\?\s*$/, '') ?? '--';
  return (
    <main
      data-cmp="ForgotPasswordPage"
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
        <Brand brand={brand} />
        <section className="bm-login-frame bm-login-reveal w-full rounded-[28px] p-1">
          <div className="rounded-[24px] border border-[var(--bm-border)] bg-[var(--bm-login-panel-bg)] p-6 shadow-[var(--bm-login-panel-shadow)] backdrop-blur sm:p-8">
            <div className="mb-6 text-center">
              <h1 className="text-2xl font-bold">忘记密码</h1>
              <p className="mt-2 text-sm text-[var(--bm-text-muted)]">
                {step === 'identity'
                  ? '请输入您需要找回密码的用户名'
                  : '验证账号邮箱并接收验证码，然后设置新密码'}
              </p>
            </div>
            {step === 'identity' ? (
              <form className="space-y-5" onSubmit={verifyIdentity} noValidate>
                <AuthField
                  label="用户名"
                  required
                  icon={<User size={17} />}
                  action={
                    <Link
                      to="/login"
                      className="inline-flex items-center gap-1 text-xs font-semibold text-[var(--bm-accent-text)]"
                    >
                      <ArrowLeft size={14} />
                      返回登录
                    </Link>
                  }
                >
                  <input
                    autoComplete="username"
                    value={username}
                    onChange={event => setUsername(event.target.value)}
                    className="bm-auth-input"
                    placeholder="请输入用户名"
                  />
                </AuthField>
                <AuthField label="验证码" required icon={<KeyRound size={17} />}>
                  <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_132px]">
                    <input
                      value={captchaAnswer}
                      onChange={event => setCaptchaAnswer(event.target.value)}
                      className="bm-auth-input"
                      placeholder="请输入验证码"
                    />
                    <button
                      type="button"
                      onClick={() => void loadCaptcha()}
                      className="bm-action-button flex h-12 items-center justify-center gap-2 rounded-2xl border px-4 text-sm font-semibold"
                      style={{ borderColor: 'var(--bm-border)' }}
                      aria-label="刷新图形验证码"
                    >
                      <span className="text-base tracking-[0.18em]">{captchaText}</span>
                      <RefreshCw
                        size={15}
                        className={busy === 'captcha' ? 'animate-spin' : ''}
                        style={{ color: 'var(--bm-text-muted)' }}
                      />
                    </button>
                  </div>
                </AuthField>
                <ErrorLine error={error} />
                <SubmitButton busy={busy === 'verify'} label="提交" />
              </form>
            ) : (
              <form className="space-y-5" onSubmit={confirm} noValidate>
                <AuthField
                  label="验证邮箱"
                  required
                  icon={<Mail size={17} />}
                  action={
                    <span className="text-xs font-semibold text-[var(--bm-text-muted)]">
                      当前账号：<span className="text-[var(--bm-text)]">{username}</span>
                    </span>
                  }
                >
                  <input
                    ref={verifyEmailRef}
                    type="email"
                    autoComplete="email"
                    value={verifyEmail}
                    onChange={event => setVerifyEmail(event.target.value)}
                    className="bm-auth-input"
                    placeholder="请输入当前账号配置的邮箱"
                  />
                </AuthField>
                <AuthField label="重置验证码" required icon={<KeyRound size={17} />}>
                  <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_96px]">
                    <input
                      value={code}
                      onChange={event => setCode(event.target.value)}
                      className="bm-auth-input min-w-0 flex-1"
                      placeholder="请输入收到的验证码"
                    />
                    <button
                      type="button"
                      disabled={Boolean(busy)}
                      onClick={() => void resend()}
                      className="bm-login-submit flex h-12 items-center justify-center gap-2 rounded-2xl text-sm font-semibold text-white disabled:opacity-60"
                    >
                      {busy === 'send' ? <Loader2 size={16} className="animate-spin" /> : null}
                      {busy === 'send' ? '发送中' : cooldown > 0 ? `${cooldown}s` : '发送'}
                    </button>
                  </div>
                </AuthField>
                <AuthField label="新密码" required icon={<Lock size={17} />}>
                  <span className="relative block">
                    <input
                      type={showPassword ? 'text' : 'password'}
                      autoComplete="new-password"
                      value={newPassword}
                      onChange={event => setNewPassword(event.target.value)}
                      className="bm-auth-input pr-12"
                      placeholder="至少 6 个字符"
                    />
                    <button
                      type="button"
                      onClick={() => setShowPassword(value => !value)}
                      className="absolute right-2 top-1/2 grid h-9 w-9 -translate-y-1/2 place-items-center"
                      aria-label={showPassword ? '隐藏密码' : '显示密码'}
                    >
                      {showPassword ? <EyeOff size={17} /> : <Eye size={17} />}
                    </button>
                  </span>
                </AuthField>
                <AuthField label="确认密码" required icon={<Lock size={17} />}>
                  <span className="relative block">
                    <input
                      type={showPassword ? 'text' : 'password'}
                      autoComplete="new-password"
                      value={confirmPassword}
                      onChange={event => setConfirmPassword(event.target.value)}
                      className="bm-auth-input pr-12"
                      placeholder="请再次输入新密码"
                    />
                    <button
                      type="button"
                      onClick={() => setShowPassword(value => !value)}
                      className="absolute right-2 top-1/2 grid h-9 w-9 -translate-y-1/2 place-items-center"
                      aria-label={showPassword ? '隐藏密码' : '显示密码'}
                    >
                      {showPassword ? <EyeOff size={17} /> : <Eye size={17} />}
                    </button>
                  </span>
                </AuthField>
                <ErrorLine error={error} />
                <SubmitButton busy={busy === 'confirm'} label="重置密码" />
              </form>
            )}
          </div>
        </section>
        <p className="text-xs text-[var(--bm-text-muted)]">
          © 2026 {brand.siteName}. Secure backup operations monitoring console.
        </p>
      </section>
    </main>
  );
}

function Brand({ brand }: { brand: ReturnType<typeof useBrandSettings> }) {
  return (
    <div className="bm-login-reveal flex flex-col items-center text-center">
      <BrandIcon
        iconData={brand.iconData}
        alt={brand.loginName}
        className="bm-brand-float mb-4 h-20 w-20 drop-shadow-[0_18px_42px_rgba(6,182,212,0.18)]"
      />
      <p className="bm-gradient-text text-2xl font-bold tracking-wide">{brand.loginName}</p>
      <p className="mt-1 text-xs uppercase tracking-[0.28em] text-[var(--bm-text-muted)]">
        {brand.appSubtitle}
      </p>
    </div>
  );
}

function AuthField({
  label,
  icon,
  required = false,
  action,
  children,
}: {
  label: string;
  icon: ReactNode;
  required?: boolean;
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <label className="block">
      <span className="mb-2 flex items-center justify-between gap-3 text-sm font-medium">
        <span>
          {label}
          {required ? <span className="ml-1 text-red-400">*</span> : null}
        </span>
        {action}
      </span>
      <span className="relative block">
        <span className="absolute left-4 top-6 z-10 -translate-y-1/2 text-[var(--bm-text-muted)]">
          {icon}
        </span>
        <span className="block [&_.bm-auth-input]:h-12 [&_.bm-auth-input]:w-full [&_.bm-auth-input]:rounded-2xl [&_.bm-auth-input]:border [&_.bm-auth-input]:border-[var(--bm-border)] [&_.bm-auth-input]:bg-[var(--bm-control-bg)] [&_.bm-auth-input]:py-3 [&_.bm-auth-input]:pl-12 [&_.bm-auth-input]:text-sm [&_.bm-auth-input]:text-[var(--bm-text)] [&_.bm-auth-input]:outline-none">
          {children}
        </span>
      </span>
    </label>
  );
}

function ErrorLine({ error }: { error: string }) {
  return (
    <div aria-live="polite" className="min-h-6">
      {error ? (
        <p
          role="alert"
          className="rounded-xl border border-red-500/25 bg-red-500/10 px-3 py-2 text-xs text-red-300"
        >
          {error}
        </p>
      ) : null}
    </div>
  );
}

function SubmitButton({ busy, label }: { busy: boolean; label: string }) {
  return (
    <button
      type="submit"
      disabled={busy}
      className="bm-login-submit flex h-12 w-full items-center justify-center gap-2 rounded-2xl bg-[linear-gradient(135deg,#2563eb,#06b6d4)] text-sm font-semibold text-white shadow-[0_18px_48px_rgba(37,99,235,0.35)] disabled:opacity-60"
    >
      {busy ? <Loader2 size={17} className="animate-spin" /> : null}
      {busy ? '处理中...' : label}
    </button>
  );
}
