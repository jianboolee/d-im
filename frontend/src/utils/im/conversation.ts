import type { Conversation, Message, LastMessage } from '@/sdk/im'
import { normalizeUnreadCount } from './format'

function toLastMessage(message: Message): LastMessage {
  return {
    msg_id: message.msg_id || '',
    from_uid: message.from_uid,
    msg_type: message.msg_type,
    content_preview: getContentPreview(message),
    client_time: message.client_time,
  }
}

function getContentPreview(msg: Message): string {
  const content = msg.content as unknown as Record<string, unknown> | undefined
  switch (msg.msg_type) {
    case 'text': return (content?.text as string) || ''
    case 'image': return '[图片]'
    case 'video': return '[视频]'
    case 'voice': return '[语音]'
    case 'file': return '[文件]'
    case 'card': return `[卡片] ${content?.title || ''}`
    case 'link': return `[链接] ${content?.title || ''}`
    case 'template': return `[模板] ${content?.title || ''}`
    case 'location': return '[位置]'
    default: return '[消息]'
  }
}

export function getUnreadCount(conversation: Conversation): number {
  return normalizeUnreadCount(conversation.unread_count ?? 0)
}

export function sortConversationsByActivity(conversations: Conversation[]): Conversation[] {
  return [...conversations].sort((a, b) => {
    if (a.is_top !== b.is_top) return a.is_top ? -1 : 1
    const tA = a.last_msg?.client_time || a.updated_at
    const tB = b.last_msg?.client_time || b.updated_at
    return new Date(tB).getTime() - new Date(tA).getTime()
  })
}

interface ConversationMap {
  [chatId: string]: Conversation
}

/** 收到实时消息后更新侧栏会话列表 */
export function applyIncomingMessage(
  conversations: Conversation[],
  message: Message,
  activeChatId?: string,
): Conversation[] {
  const existing = conversations.find(c => c.chat_id === message.chat_id)
  const isFromOther = message.from_uid !== '' // TODO: need currentUserId
  const isNotActive = message.chat_id !== activeChatId

  const updated: Conversation = existing
    ? {
        ...existing,
        last_msg: toLastMessage(message),
        updated_at: message.server_time || message.created_at,
        unread_count: isFromOther && isNotActive
          ? (existing.unread_count || 0) + 1
          : existing.unread_count,
      }
    : {
        uid: '',
        chat_id: message.chat_id,
        chat_type: message.chat_type as Conversation['chat_type'],
        is_top: false,
        is_muted: false,
        unread_count: isFromOther ? 1 : 0,
        last_msg: toLastMessage(message),
        joined_at: '',
        updated_at: message.server_time || message.created_at,
      }

  const others = conversations.filter(c => c.chat_id !== message.chat_id)
  return sortConversationsByActivity([updated, ...others])
}

/** 清除会话未读 */
export function withClearedUnreadForConversation(
  conversations: Conversation[],
  chatId: string,
): Conversation[] {
  return conversations.map(c => c.chat_id === chatId ? { ...c, unread_count: 0 } : c)
}