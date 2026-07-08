import type { Conversation, Message, LastMessage } from '@/sdk/im'
import { normalizeUnreadCount } from '@/utils/im/format'

function toSnapshot(message: Message): LastMessage {
  const content = getMessagePreview(message)
  return {
    msg_id: message.id ?? '',
    sequence: message.seq ?? 0,
    sender_id: message.sender_id ?? '',
    sender_name: message.sender_profile?.nickname,
    msg_type: message.type,
    content_preview: content,
    client_time: message.created_at ?? new Date().toISOString(),
  }
}

function getMessagePreview(message: Message): string {
  if (message.content_preview) return message.content_preview

  switch (message.type) {
    case 'text':
      return message.content.text || ''
    case 'image':
      return '[图片]'
    case 'video':
      return '[视频]'
    case 'voice':
      return '[语音]'
    case 'file':
      return message.content.file_name ? `[文件] ${message.content.file_name}` : '[文件]'
    case 'card':
      return message.content.title || '[消息]'
    case 'link':
      return message.content.title ? `[链接] ${message.content.title}` : '[链接]'
    default:
      return message.content.text || message.content.title || message.content.file_name || '[消息]'
  }
}

export function getPeerUserId(conversation: Conversation, currentUserId: string): string {
  return conversation.participants.find((id) => id !== currentUserId) ?? ''
}

export function isGroupConversation(conversation?: Conversation | null): boolean {
  return conversation?.chat_type === 'group'
}

export function getConversationDisplayName(conversation: Conversation, currentUserId: string): string {
  if (conversation.display_name) return conversation.display_name
  if (isGroupConversation(conversation)) return conversation.group_info?.name || '群聊'

  const peerId = getPeerUserId(conversation, currentUserId)
  return conversation.peer_user_info?.nickname
    || (peerId ? `用户${peerId.slice(-4)}` : '未知用户')
}

export function getConversationDisplayAvatar(conversation: Conversation): string {
  if (isGroupConversation(conversation)) {
    return conversation.display_avatar || conversation.group_info?.avatar_url || ''
  }
  return conversation.display_avatar
    || conversation.peer_user_info?.avatar
    || ''
}

export function getUnreadCount(conversation: Conversation, currentUserId: string): number {
  if (!currentUserId) return 0
  if (conversation.unread_count != null) {
    return normalizeUnreadCount(conversation.unread_count)
  }
  const lastSequence = conversation.last_message?.sequence ?? 0
  const lastReadSeq = conversation.last_read_sequence ?? 0
  return normalizeUnreadCount(lastSequence - lastReadSeq)
}

export function hasUnreadConversation(conversation: Conversation): boolean {
  const lastSequence = conversation.last_message?.sequence ?? 0
  const lastReadSeq = conversation.last_read_sequence ?? 0
  return lastSequence > lastReadSeq
}

export function sortConversationsByActivity(conversations: Conversation[]): Conversation[] {
  return [...conversations].sort((a, b) => {
    const pinnedA = Boolean(a.pinned)
    const pinnedB = Boolean(b.pinned)
    if (pinnedA !== pinnedB) return pinnedA ? -1 : 1
    const timeA = a.last_activity || a.last_message?.client_time || a.updated_at
    const timeB = b.last_activity || b.last_message?.client_time || b.updated_at
    return new Date(timeB).getTime() - new Date(timeA).getTime()
  })
}

function matchesMessage(conversation: Conversation, message: Message): boolean {
  return Boolean(
    (message.conversation_id && conversation.id === message.conversation_id)
    || (message.chat_id && conversation.chat_id === message.chat_id),
  )
}

function maxTime(...values: Array<string | undefined>): string {
  const timestamps = values
    .map((value) => (value ? new Date(value).getTime() : 0))
    .filter((value) => Number.isFinite(value) && value > 0)
  const max = Math.max(...timestamps)
  return max > 0 ? new Date(max).toISOString() : new Date().toISOString()
}

export function buildConversationFromMessage(message: Message, currentUserId: string): Conversation {
  const senderId = message.sender_id ?? ''
  const timestamp = message.created_at ?? new Date().toISOString()

  return {
    id: message.conversation_id ?? '',
    conversation_id: message.conversation_id ?? '',
    chat_id: message.chat_id ?? '',
    chat_type: 'single',
    participants: [senderId, currentUserId].filter(Boolean),
    last_message: toSnapshot(message),
    preview_image_url: undefined,
    last_read_sequence: message.sender_id === currentUserId ? (message.seq ?? 0) : 0,
    unread_count: message.sender_id === currentUserId ? 0 : 1,
    muted: false,
    pinned: false,
    created_at: timestamp,
    updated_at: timestamp,
    last_activity: timestamp,
  }
}

/** 收到实时消息后，不可变地更新会话列表 */
export function applyIncomingMessage(
  conversations: Conversation[],
  message: Message,
  currentUserId: string,
  activeChatId?: string,
): Conversation[] {
  const index = conversations.findIndex((conversation) => matchesMessage(conversation, message))

  if (index === -1) {
    if (!message.conversation_id) return conversations
    const created = buildConversationFromMessage(message, currentUserId)
    if (activeChatId && created.chat_id === activeChatId && message.sender_id !== currentUserId) {
      created.last_read_sequence = Math.max(created.last_read_sequence ?? 0, message.seq ?? 0)
      created.unread_count = 0
    }
    return sortConversationsByActivity([created, ...conversations])
  }

  const existing = conversations[index]!
  const isFromOther = message.sender_id !== currentUserId
  const isNotActive = existing.chat_id !== activeChatId
  const isSameLastMessage = Boolean(
    (message.id && existing.last_message?.msg_id === message.id)
    || (message.seq != null && existing.last_message?.sequence === message.seq),
  )
  const shouldUpdateUnread = isFromOther && isNotActive
  const nextMessageSeq = message.seq ?? existing.last_message?.sequence ?? 0
  const currentReadSeq = existing.last_read_sequence ?? 0
  const nextReadSeq = shouldUpdateUnread
    ? currentReadSeq
    : Math.max(currentReadSeq, nextMessageSeq)
  const nextUnreadCount = shouldUpdateUnread
    ? getUnreadCount(existing, currentUserId) + (isSameLastMessage ? 0 : 1)
    : 0

  const updated: Conversation = {
    ...existing,
    last_message: toSnapshot(message),
    preview_image_url: message._ws_preview_image_url ?? existing.preview_image_url,
    updated_at: message.created_at ?? existing.updated_at,
    last_activity: maxTime(message.created_at, existing.last_activity, existing.updated_at),
    last_read_sequence: nextReadSeq,
    unread_count: nextUnreadCount,
  }

  const next = [...conversations]
  next[index] = updated
  return sortConversationsByActivity(next)
}

/** 本地推进指定会话的已读指针（进入聊天或当前会话收到消息后同步侧栏） */
export function withClearedUnreadForPeer(
  conversations: Conversation[],
  peerId: string,
  currentUserId: string,
): Conversation[] {
  if (!peerId || !currentUserId) {
    return conversations
  }

  return conversations.map((conversation) => {
    if (getPeerUserId(conversation, currentUserId) !== peerId) {
      return conversation
    }

    return {
      ...conversation,
      last_read_sequence: Math.max(
        conversation.last_read_sequence ?? 0,
        conversation.last_message?.sequence ?? 0,
      ),
      unread_count: 0,
    }
  })
}

export function withClearedUnreadForConversation(
  conversations: Conversation[],
  conversationId: string,
): Conversation[] {
  if (!conversationId) return conversations

  return conversations.map((conversation) => {
    if (conversation.id !== conversationId) {
      return conversation
    }

    return {
      ...conversation,
      last_read_sequence: Math.max(
        conversation.last_read_sequence ?? 0,
        conversation.last_message?.sequence ?? 0,
      ),
      unread_count: 0,
    }
  })
}

export function collectPeerUserIds(conversations: Conversation[], currentUserId: string): string[] {
  const ids = new Set<string>()

  for (const conversation of conversations) {
    const peerId = getPeerUserId(conversation, currentUserId)
    if (peerId) ids.add(peerId)
  }

  return [...ids]
}
