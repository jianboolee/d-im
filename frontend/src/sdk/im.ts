// ============================================================
// 消息类型 — 对齐后端 pkg/types
// ============================================================
export enum MessageType {
  Text = 'text',
  Image = 'image',
  Video = 'video',
  Voice = 'voice',
  Card = 'card',
  Link = 'link',
  Template = 'template',
  File = 'file',
  Location = 'location',
}

export enum MessageStatus {
  Sending = 'sending',
  Sent = 'sent',
  Delivered = 'delivered',
  Read = 'read',
  Failed = 'failed',
  Recalled = 'recalled',
}

export enum ChatType {
  Single = 'single',
  Group = 'group',
  System = 'system',
}

// ============================================================
// 消息内容 — 多态 Content 结构
// ============================================================
export interface TextContent {
  text: string
  mentions?: string[]
  is_at_all?: boolean
}

export interface ImageContent {
  url: string
  thumb_url?: string
  width: number
  height: number
  size: number
  format: string
  md5?: string
  file_name?: string
}

export interface VideoContent {
  url: string
  thumb_url?: string
  duration: number
  width: number
  height: number
  size: number
  format: string
  md5?: string
}

export interface VoiceContent {
  url: string
  duration: number
  size: number
  format: string
  md5?: string
}

export interface CardContent {
  title: string
  description?: string
  image_url?: string
  action_url?: string
}

export interface LinkContent {
  url: string
  title: string
  description?: string
  thumb_url?: string
  favicon?: string
}

export interface TemplateItem {
  label: string
  value: string
  type?: string
  color?: string
  action_url?: string
}

export interface TemplateContent {
  template_id: string
  title?: string
  items: TemplateItem[]
  description?: string
  action_url?: string
  action_text?: string
}

export interface FileContent {
  url: string
  file_name: string
  size: number
  format: string
  md5?: string
}

export interface LocationContent {
  latitude: number
  longitude: number
  address?: string
  name?: string
}

// ============================================================
// 消息模型 — 对齐后端 model.Message
// ============================================================
export interface QuoteMessage {
  msg_id: string
  from_uid: string
  from_name: string
  msg_type: MessageType
  content_preview: string
}

export interface UserInfo {
  id: string
  nickname?: string
  avatar?: string
}

export interface Message {
  id?: string
  msg_id: string
  chat_id: string
  chat_type: ChatType
  from_uid: string
  from_name?: string
  msg_type: MessageType
  content: TextContent | ImageContent | VideoContent | VoiceContent | CardContent | LinkContent | TemplateContent | FileContent | LocationContent
  quote_msg_id?: string
  quote_msg?: QuoteMessage
  status: MessageStatus
  is_recalled?: boolean
  recall_time?: string
  client_time: string
  server_time: string
  created_at: string
  updated_at: string
}

// ============================================================
// 会话 — 对齐后端 model.Conversation（摘要视图）
// ============================================================
export interface LastMessage {
  msg_id: string
  from_uid: string
  msg_type: MessageType
  content_preview: string
  client_time: string
}

export interface Conversation {
  id?: string
  uid: string
  chat_id: string
  chat_type: ChatType
  is_top: boolean
  is_muted: boolean
  unread_count: number
  last_msg?: LastMessage
  joined_at: string
  updated_at: string
}

// ============================================================
// SDK 配置 & API 请求/响应
// ============================================================
export interface IMSDKOptions {
  baseURL: string
  wsURL: string
  token: string
  deviceId: string
}

export interface SendMessageReq {
  chat_id: string
  chat_type: ChatType
  from_name?: string
  msg_type: MessageType
  content: Record<string, unknown>
  target_uids: string[]
  quote_msg_id?: string
}

export interface SendMessageResp {
  msg_id: string
  server_time: string
  status: MessageStatus
}

export interface ConnectionStatus {
  status: 'connected' | 'disconnected' | 'error'
  error?: unknown
}

export type MessageHandler = (message: Message) => void
export type ConnectionHandler = (status: ConnectionStatus) => void

// ============================================================
// API 请求工具
// ============================================================
async function apiPost<T>(baseURL: string, path: string, token: string, body: unknown): Promise<T> {
  const resp = await fetch(`${baseURL}${path}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
    body: JSON.stringify(body),
  })
  if (!resp.ok) {
    const err = await resp.json().catch(() => ({}))
    throw new Error((err as { error?: string }).error || `HTTP ${resp.status}`)
  }
  return resp.json()
}

async function apiGet<T>(baseURL: string, path: string, token: string): Promise<T> {
  const resp = await fetch(`${baseURL}${path}`, {
    headers: { 'Authorization': `Bearer ${token}` },
  })
  if (!resp.ok) {
    throw new Error(`HTTP ${resp.status}`)
  }
  return resp.json()
}

// ============================================================
// IM SDK
// ============================================================
class IMSDK {
  private baseURL: string
  private wsURL: string
  private token: string
  private deviceId: string
  private ws: WebSocket | null = null
  private messageHandlers: MessageHandler[] = []
  private connectionHandlers: ConnectionHandler[] = []
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null

  constructor(options: IMSDKOptions) {
    this.baseURL = options.baseURL
    this.wsURL = options.wsURL
    this.token = options.token
    this.deviceId = options.deviceId
  }

  // ---- WebSocket ----

  connect(): Promise<void> {
    if (this.ws?.readyState === WebSocket.OPEN) return Promise.resolve()
    this.ws?.close()

    return new Promise((resolve, reject) => {
      const url = `${this.wsURL}?token=${encodeURIComponent(this.token)}&device_id=${encodeURIComponent(this.deviceId)}`
      const ws = new WebSocket(url)
      this.ws = ws
      let settled = false

      ws.onopen = () => {
        this._notifyConnection({ status: 'connected' })
        this.startHeartbeat()
        if (!settled) { settled = true; resolve() }
      }

      ws.onclose = () => {
        if (this.ws === ws) this.ws = null
        this.stopHeartbeat()
        this._notifyConnection({ status: 'disconnected' })
        if (!settled) { settled = true; reject(new Error('WebSocket closed')) }
      }

      ws.onmessage = (event: MessageEvent) => {
        try {
          const msg = JSON.parse(event.data) as Message
          this._notifyMessage(msg)
        } catch (e) {
          console.warn('[im-sdk] invalid WS message', e)
        }
      }

      ws.onerror = () => {
        this.stopHeartbeat()
        this._notifyConnection({ status: 'error' })
        if (!settled) { settled = true; reject(new Error('WebSocket error')) }
      }
    })
  }

  disconnect(): void {
    this.stopHeartbeat()
    this.ws?.close()
    this.ws = null
  }

  updateToken(token: string): void {
    this.token = token
  }

  private startHeartbeat(): void {
    this.heartbeatTimer = setInterval(() => {
      // WebSocket 层自带 ping/pong，无需额外心跳
    }, 30_000)
  }

  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
    }
  }

  private _notifyMessage(msg: Message): void {
    this.messageHandlers.forEach(h => h(msg))
  }

  private _notifyConnection(status: ConnectionStatus): void {
    this.connectionHandlers.forEach(h => h(status))
  }

  // ---- HTTP API ----

  /** 发送消息 */
  async sendMessage(
    chatId: string,
    chatType: ChatType,
    msgType: MessageType,
    content: Record<string, unknown>,
    targetUIDs: string[],
    fromName?: string,
  ): Promise<SendMessageResp> {
    return apiPost<SendMessageResp>(this.baseURL, '/api/v1/message/send', this.token, {
      chat_id: chatId,
      chat_type: chatType,
      from_name: fromName || '',
      msg_type: msgType,
      content,
      target_uids: targetUIDs,
    })
  }

  /** 发送文本消息快捷方法 */
  async sendTextMessage(chatId: string, chatType: ChatType, text: string, targetUIDs: string[], fromName?: string): Promise<SendMessageResp> {
    return this.sendMessage(chatId, chatType, MessageType.Text, { text }, targetUIDs, fromName)
  }

  /** 撤回消息 */
  async recallMessage(msgId: string): Promise<void> {
    await apiPost(this.baseURL, '/api/v1/message/recall', this.token, { msg_id: msgId })
  }

  /** 获取会话列表 */
  async getConversations(): Promise<Conversation[]> {
    return apiGet<Conversation[]>(this.baseURL, '/api/v1/conversation/list', this.token)
  }

  /** 已读标记 */
  async markRead(chatId: string): Promise<void> {
    await apiPost(this.baseURL, '/api/v1/conversation/read', this.token, { chat_id: chatId })
  }

  // ---- 事件监听 ----

  onMessage(handler: MessageHandler): void { this.messageHandlers.push(handler) }
  offMessage(handler: MessageHandler): void {
    const idx = this.messageHandlers.indexOf(handler)
    if (idx !== -1) this.messageHandlers.splice(idx, 1)
  }

  onConnection(handler: ConnectionHandler): void { this.connectionHandlers.push(handler) }
  offConnection(handler: ConnectionHandler): void {
    const idx = this.connectionHandlers.indexOf(handler)
    if (idx !== -1) this.connectionHandlers.splice(idx, 1)
  }
}

export default IMSDK