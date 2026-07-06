import dayjs from 'dayjs'
import type { Conversation, LastMessage } from '@/sdk/im'

const EMPTY_TIME = '0001-01-01T00:00:00Z'

/** 会话列表时间展示：今天显示时分，昨天显示「昨天」，同年显示月日 */
export function formatConversationTime(time?: string): string {
  if (!time || time === EMPTY_TIME) return ''

  const date = dayjs(time)
  const now = dayjs()

  if (date.isSame(now, 'day')) return date.format('HH:mm')
  if (date.isSame(now.subtract(1, 'day'), 'day')) return '昨天'
  if (date.isSame(now, 'year')) return date.format('MM-DD')
  return date.format('YYYY-MM-DD')
}

/** 会话列表最后一条消息预览文案 — 消息摘要由后端生成，前端只处理会话上下文前缀 */
export function formatLastMessagePreview(
  snapshot: LastMessage | undefined,
  conversation?: Conversation,
  currentUserId?: string,
): string {
  if (!snapshot) return ''
  const previewText = snapshot.content_preview || ''
  if (
    conversation &&
    conversation.chat_type === 'group' &&
    snapshot.sender_id &&
    snapshot.sender_id !== currentUserId &&
    snapshot.msg_type !== 'system_event'
  ) {
    const senderName = snapshot.sender_id
    return senderName ? `${senderName}: ${previewText}` : previewText
  }
  return previewText
}

/** 将未读数规范为非负整数 */
export function normalizeUnreadCount(count: number): number {
  if (!Number.isFinite(count)) {
    return 0
  }

  return Math.max(0, Math.floor(count))
}

/** 会话列表未读角标：0 不展示，超过 99 显示 99+ */
export function formatUnreadBadge(count: number): string {
  const normalized = normalizeUnreadCount(count)
  if (normalized <= 0) {
    return ''
  }

  return normalized > 99 ? '99+' : String(normalized)
}
