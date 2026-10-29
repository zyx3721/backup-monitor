import type { NotifyChannel } from './types';

export const typeLabels: Record<NotifyChannel['type'], string> = {
  mail: '邮件',
  webhook: 'WebHook',
  wecom: '企业微信',
  dingtalk: '钉钉',
  feishu: '飞书',
};

export const eventOptions = ['backup_failed', 'not_run', 'report_timeout'] as const;

export const levelLabels: Record<string, string> = {
  all: '全部级别',
  info: '提示及以上',
  warning: '警告及以上',
  critical: '仅严重',
};

export const alertLevelLabels: Record<string, string> = {
  info: '提示',
  warning: '警告',
  critical: '严重',
};

export const eventLevelHints: Record<string, string> = {
  backup_failed: '级别：严重',
  not_run: '级别：严重',
  report_timeout: '级别：未达离线阈值时为警告，离线或心跳中断时为严重',
};

export const levelTones: Record<string, 'info' | 'success' | 'warning' | 'danger'> = {
  all: 'info',
  info: 'success',
  warning: 'warning',
  critical: 'danger',
};

export const receiveIdTypeOptions = [
  { value: 'chat_id', label: '群聊 ID' },
  { value: 'open_id', label: 'Open ID' },
  { value: 'user_id', label: 'User ID' },
  { value: 'union_id', label: 'Union ID' },
  { value: 'email', label: '邮箱' },
] as const;

const alertBodyLines = [
  '主机：{{主机}}（{{IP}}）',
  '内容：{{内容}}',
  '级别：{{级别}}',
  '时间：{{时间}}',
  '状态：{{状态}}',
];

export const alertContentTemplate = ['【备份监控】{{告警}}：{{任务}}', ...alertBodyLines].join(
  '\n'
);

const alertBodyTemplate = alertBodyLines.join('\n');

export type MessageType = 'text' | 'markdown' | 'post' | 'interactive';

export const messageTypeOptions: Partial<
  Record<NotifyChannel['type'], { value: MessageType; label: string }[]>
> = {
  wecom: [
    { value: 'text', label: '文本' },
    { value: 'markdown', label: 'Markdown' },
  ],
  dingtalk: [
    { value: 'text', label: '文本' },
    { value: 'markdown', label: 'Markdown' },
  ],
  feishu: [
    { value: 'text', label: '文本' },
    { value: 'post', label: '富文本' },
    { value: 'interactive', label: '卡片' },
  ],
};

export const defaultMessageTypes: Partial<Record<NotifyChannel['type'], MessageType>> = {
  wecom: 'text',
  dingtalk: 'text',
  feishu: 'text',
};

const dingtalkMarkdownText = alertContentTemplate.replace(/\n/g, '\n\n');

function wecomBody(messageType: MessageType): string {
  const msgtype = messageType === 'markdown' ? 'markdown' : 'text';
  return JSON.stringify(
    { touser: '@all', msgtype, agentid: 1, [msgtype]: { content: alertContentTemplate } },
    null,
    2
  );
}

function dingtalkBody(messageType: MessageType): string {
  if (messageType === 'markdown') {
    return JSON.stringify(
      {
        agent_id: '',
        userid_list: '',
        msg: {
          msgtype: 'markdown',
          markdown: {
            title: '【备份监控】{{告警}}：{{任务}}',
            text: dingtalkMarkdownText,
          },
        },
      },
      null,
      2
    );
  }
  return JSON.stringify(
    {
      agent_id: '',
      userid_list: '',
      msg: { msgtype: 'text', text: { content: alertContentTemplate } },
    },
    null,
    2
  );
}

function feishuBody(messageType: MessageType): string {
  if (messageType === 'post') {
    const post = {
      zh_cn: {
        title: '【备份监控】{{告警}}：{{任务}}',
        content: alertBodyTemplate.split('\n').map(line => [{ tag: 'text', text: line }]),
      },
    };
    return JSON.stringify({ receive_id: '', msg_type: 'post', content: post }, null, 2);
  }
  if (messageType === 'interactive') {
    const card = {
      config: { wide_screen_mode: true },
      header: {
        template: '{{颜色}}',
        title: { tag: 'plain_text', content: '【备份监控】{{告警}}：{{任务}}' },
      },
      elements: [{ tag: 'div', text: { tag: 'lark_md', content: alertBodyTemplate } }],
    };
    return JSON.stringify({ receive_id: '', msg_type: 'interactive', content: card }, null, 2);
  }
  return JSON.stringify(
    { receive_id: '', msg_type: 'text', content: { text: alertContentTemplate } },
    null,
    2
  );
}

export const bodyTemplates: Record<NotifyChannel['type'], string> = {
  mail: '',
  webhook: JSON.stringify({ msgtype: 'text', text: { content: alertContentTemplate } }, null, 2),
  wecom: wecomBody('text'),
  dingtalk: dingtalkBody('text'),
  feishu: feishuBody('text'),
};

export function bodyTemplateFor(
  type: NotifyChannel['type'],
  messageType: MessageType = 'text'
): string {
  if (type === 'wecom') return wecomBody(messageType);
  if (type === 'dingtalk') return dingtalkBody(messageType);
  if (type === 'feishu') return feishuBody(messageType);
  return bodyTemplates[type];
}

export function inferMessageType(channel: NotifyChannel): MessageType {
  const options = messageTypeOptions[channel.type];
  if (!options) return 'text';
  const saved = channel.config.messageType;
  if (typeof saved === 'string' && options.some(option => option.value === saved)) {
    return saved as MessageType;
  }
  const raw = channel.config.body;
  if (typeof raw !== 'string' || !raw.trim()) return 'text';
  const parsed = parseBody(raw);
  if (!parsed) return 'text';
  const value =
    channel.type === 'feishu'
      ? parsed.msg_type
      : channel.type === 'dingtalk'
        ? parsed.msg?.msgtype
        : parsed.msgtype;
  if (typeof value === 'string' && options.some(option => option.value === value)) {
    return value as MessageType;
  }
  return 'text';
}

const targetFields: Partial<Record<NotifyChannel['type'], string>> = {
  wecom: 'touser',
  dingtalk: 'userid_list',
  feishu: 'receive_id',
};

export function preserveTarget(
  type: NotifyChannel['type'],
  currentBody: string,
  nextTemplate: string
): string {
  const field = targetFields[type];
  if (!field) return nextTemplate;
  const value = readStringField(currentBody, field).trim();
  if (!value) return nextTemplate;
  try {
    const parsed = JSON.parse(nextTemplate) as Record<string, unknown>;
    if (!Object.prototype.hasOwnProperty.call(parsed, field)) return nextTemplate;
    parsed[field] = value;
    return JSON.stringify(parsed, null, 2);
  } catch {
    return nextTemplate;
  }
}

function readStringField(raw: string, field: string): string {
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    const value = parsed[field];
    return typeof value === 'string' ? value : '';
  } catch {
    return '';
  }
}

function parseBody(raw: string): {
  msg_type?: string;
  msgtype?: string;
  msg?: { msgtype?: string };
} | null {
  try {
    return JSON.parse(raw) as {
      msg_type?: string;
      msgtype?: string;
      msg?: { msgtype?: string };
    };
  } catch {
    return null;
  }
}

export const defaultHeaders = JSON.stringify({ 'Content-Type': 'application/json' }, null, 2);

export const defaultMailRecipients = JSON.stringify(['123456@qq.com', 'domain@163.com'], null, 4);

export const variableChips = [
  '告警',
  '主机',
  'IP',
  '任务',
  '内容',
  '级别',
  '时间',
  '状态',
  '颜色',
  '告警内容',
] as const;

export type ChannelFormState = {
  id?: string;
  name: string;
  type: NotifyChannel['type'];
  enabled: boolean;
  recover: boolean;
  level: NotifyChannel['level'];
  eventType: string[];
  remark: string;
  config: Record<string, string>;
};

export const emptyChannelForm: ChannelFormState = {
  name: '',
  type: 'mail',
  enabled: true,
  recover: false,
  level: 'all',
  eventType: ['backup_failed'],
  remark: '',
  config: {
    recipients: defaultMailRecipients,
    url: '',
    method: 'POST',
    headers: defaultHeaders,
    body: '',
    corpId: '',
    appSecret: '',
    appKey: '',
    receiveIdType: 'user_id',
    messageType: 'text',
  },
};

const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function parseRecipients(raw: string): { list: string[]; error: string } {
  const text = raw.trim();
  if (!text) return { list: [], error: '请至少填写一个收件邮箱' };
  let parsed: unknown;
  try {
    parsed = JSON.parse(text);
  } catch {
    return { list: [], error: '邮件列表需为合法的 JSON 数组' };
  }
  if (!Array.isArray(parsed)) return { list: [], error: '邮件列表需为合法的 JSON 数组' };
  const list = parsed.map(value => String(value).trim()).filter(Boolean);
  if (list.length === 0) return { list, error: '请至少填写一个收件邮箱' };
  const invalid = list.find(value => !emailPattern.test(value));
  if (invalid) return { list, error: `收件邮箱格式不正确：${invalid}` };
  return { list, error: '' };
}

export function recipientsToText(value: unknown): string {
  if (Array.isArray(value)) return JSON.stringify(value.map(String), null, 4);
  if (typeof value === 'string' && value.trim()) {
    const parsed = (() => {
      try {
        return JSON.parse(value);
      } catch {
        return null;
      }
    })();
    if (Array.isArray(parsed)) return JSON.stringify(parsed.map(String), null, 4);
    const list = value
      .split(/[\n,;，；]/)
      .map(item => item.trim())
      .filter(Boolean);
    if (list.length > 0) return JSON.stringify(list, null, 4);
  }
  return '[]';
}

export function channelToForm(channel: NotifyChannel): ChannelFormState {
  const config: Record<string, string> = {};
  for (const key of [
    'url',
    'method',
    'headers',
    'body',
    'corpId',
    'appKey',
    'appSecret',
    'messageType',
  ]) {
    const value = channel.config[key];
    config[key] = typeof value === 'string' ? value : '';
  }
  if (!config.appSecret && typeof channel.config.secret === 'string') {
    config.appSecret = channel.config.secret;
  }
  config.recipients = recipientsToText(channel.config.recipients);
  config.headers = config.headers || defaultHeaders;
  const messageType = inferMessageType(channel);
  config.messageType = messageType;
  config.body = config.body || bodyTemplateFor(channel.type, messageType);
  config.receiveIdType =
    typeof channel.config.receiveIdType === 'string' && channel.config.receiveIdType
      ? channel.config.receiveIdType
      : 'user_id';
  return {
    id: channel.id,
    name: channel.name,
    type: channel.type,
    enabled: channel.enabled,
    recover: channel.recover,
    level: channel.level,
    eventType: channel.eventType,
    remark: channel.remark ?? '',
    config,
  };
}

// copyChannelForm 以现有渠道生成副本表单，去掉 id 并默认追加“副本”后缀
export function copyChannelForm(channel: NotifyChannel): ChannelFormState {
  const form = channelToForm(channel);
  return { ...form, id: undefined, name: `${channel.name} 副本` };
}

export function formToPayload(state: ChannelFormState): Partial<NotifyChannel> {
  const config: Record<string, unknown> = {};
  if (state.type === 'mail') {
    config.recipients = parseRecipients(state.config.recipients).list;
  } else if (state.type === 'webhook') {
    config.method = state.config.method || 'POST';
    config.url = state.config.url.trim();
    config.headers = state.config.headers.trim();
    config.body = state.config.body.trim();
  } else if (state.type === 'wecom') {
    config.corpId = state.config.corpId.trim();
    config.appSecret = state.config.appSecret.trim();
    config.messageType = state.config.messageType || 'text';
    config.body = state.config.body.trim();
  } else if (state.type === 'dingtalk') {
    config.appKey = state.config.appKey.trim();
    config.appSecret = state.config.appSecret.trim();
    config.messageType = state.config.messageType || 'text';
    config.body = state.config.body.trim();
  } else {
    config.appKey = state.config.appKey.trim();
    config.appSecret = state.config.appSecret.trim();
    config.receiveIdType = state.config.receiveIdType || 'user_id';
    config.messageType = state.config.messageType || 'text';
    config.body = state.config.body.trim();
  }
  return {
    name: state.name.trim(),
    type: state.type,
    enabled: state.enabled,
    recover: state.recover,
    level: state.level,
    eventType: state.eventType,
    remark: state.remark.trim(),
    triggerGroup: '默认分组',
    config,
  };
}

// testChannelSuccessText 渠道测试成功的提示文案，邮件渠道补充去重后的收件邮箱
export function testChannelSuccessText(channel: NotifyChannel): string {
  const base = `通知渠道 ${channel.name} 已发送`;
  if (channel.type !== 'mail') return base;
  const recipients = uniqueRecipients(channel.config.recipients);
  return recipients.length > 0 ? `${base}至 ${recipients.join('、')}` : base;
}

// testChannelFailureText 渠道测试失败的提示文案
export function testChannelFailureText(channel: NotifyChannel, reason: string): string {
  return `通知渠道 ${channel.name} 发送失败：${reason.trim() || '未知错误'}`;
}

function uniqueRecipients(raw: unknown): string[] {
  const seen = new Set<string>();
  const recipients: string[] = [];
  for (const recipient of parseRecipients(recipientsToText(raw)).list) {
    const key = recipient.toLowerCase();
    if (seen.has(key)) continue;
    seen.add(key);
    recipients.push(recipient);
  }
  return recipients;
}
