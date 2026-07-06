import type { ChatMessage } from '@/types/im'

export function formatSystemEventMessage(message: ChatMessage): string {
  return message.content_preview || message.content.text || message.content.title || '群聊状态已更新'
}
