import type { Conversation, LastMessage } from '@/sdk/im'

const EMPTY_TIME = '0001-01-01T00:00:00Z'

/** 会话列表时间展示 */
export function formatConversationTime(time?: string): string {
  if (!time || time === EMPTY_TIME) return ''
  const date = new Date(time)
  if (isNaN(date.getTime())) return ''
  const now = new Date()
  if (isSameDay(date, now)) return `${pad2(date.getHours())}:${pad2(date.getMinutes())}`
  const yesterday = new Date(now)
  yesterday.setDate(yesterday.getDate() - 1)
  if (isSameDay(date, yesterday)) return '昨天'
  if (date.getFullYear() === now.getFullYear()) return `${date.getMonth() + 1}-${pad2(date.getDate())}`
  return `${date.getFullYear()}-${date.getMonth() + 1}-${pad2(date.getDate())}`
}

/** 消息时间展示（时间轴分割线用） */
export function formatMessageDividerTime(time: string): string {
  const date = new Date(time)
  if (isNaN(date.getTime())) return ''
  const now = new Date()
  const t = `${pad2(date.getHours())}:${pad2(date.getMinutes())}`
  if (isSameDay(date, now)) return t
  if (date.getFullYear() === now.getFullYear()) return `${date.getMonth() + 1}-${pad2(date.getDate())} ${t}`
  return `${date.getFullYear()}-${date.getMonth() + 1}-${pad2(date.getDate())} ${t}`
}

/** 会话列表最后一条消息预览文案 */
export function formatLastMessagePreview(
  snapshot: LastMessage | undefined,
  conversation?: Conversation,
  currentUserId?: string,
): string {
  if (!snapshot) return ''
  const previewText = snapshot.content_preview || ''
  if (conversation?.chat_type === 'group' && snapshot.from_uid && snapshot.from_uid !== currentUserId) {
    return `${snapshot.from_uid}: ${previewText}`
  }
  return previewText
}

/** 未读数 */
export function normalizeUnreadCount(count: number): number {
  return Math.max(0, Math.floor(count))
}

/** 未读角标文案 */
export function formatUnreadBadge(count: number): string {
  const n = normalizeUnreadCount(count)
  return n <= 0 ? '' : n > 99 ? '99+' : String(n)
}

const pad2 = (v: number) => String(v).padStart(2, '0')
const isSameDay = (a: Date, b: Date) =>
  a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate()