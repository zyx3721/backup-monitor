/* 找回密码流程：算术验证码、身份校验、邮件验证码发送与密码重置 */

import { api } from '@/lib/api';

export type PasswordResetCaptcha = {
  token: string;
  question: string;
  expiresAt: string;
};

export type PasswordResetChannel = {
  id: string;
  name: string;
  maskedTo: string;
  requiresTo: boolean;
};

export const fetchPasswordResetCaptcha = () =>
  api<PasswordResetCaptcha>('/api/auth/password-reset/captcha', { auth: false });

export const verifyPasswordResetIdentity = (body: {
  username: string;
  captchaToken: string;
  captchaAnswer: string;
}) =>
  api<{ verificationToken: string; channels: PasswordResetChannel[] }>(
    '/api/auth/password-reset/verify',
    {
      method: 'POST',
      auth: false,
      body: JSON.stringify(body),
    }
  );

export const sendPasswordResetCode = (body: {
  username: string;
  verificationToken: string;
  channel: string;
  verifyEmail: string;
}) =>
  api<{ status: string; cooldownSeconds: number }>('/api/auth/password-reset/send', {
    method: 'POST',
    auth: false,
    body: JSON.stringify(body),
  });

export const confirmPasswordReset = (body: {
  username: string;
  verificationToken: string;
  code: string;
  newPassword: string;
  confirmPassword: string;
}) =>
  api<{ status: string }>('/api/auth/password-reset/confirm', {
    method: 'POST',
    auth: false,
    body: JSON.stringify(body),
  });
