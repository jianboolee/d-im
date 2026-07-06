import type { Message } from '@/sdk/im'

export type {
  Message,
  MessageContent,
  MessageType,
  MessageStatus,
  Conversation,
  ConversationPage,
} from '@/sdk/im'

/** 聊天 UI 层扩展的消息状态 */
export type ChatMessageStatus = 'sending' | 'failed' | 'sent' | 'delivered' | 'read' | 'recalled'

/** 纯前端本地上传状态，不在消息 content 中传输 */
export interface UploadState {
  uploading?: boolean
  localFile?: File
  uploadFailed?: boolean
}

export type ChatMessage = Omit<Message, 'status'> & {
  status?: ChatMessageStatus
  /** 本地上传状态，仅前端使用 */
  uploadState?: UploadState
}
