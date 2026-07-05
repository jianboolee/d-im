import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Conversation, Message } from '@/sdk/im'
import { sortConversationsByActivity, applyIncomingMessage, withClearedUnreadForConversation } from '@/utils/im/conversation'

const PAGE_SIZE = 20

export const useConversationListStore = defineStore('conversationList', () => {
  const conversations = ref<Conversation[]>([])
  const loading = ref(false)
  const loadingMore = ref(false)
  const error = ref<string | null>(null)
  const hasMore = ref(true)
  const nextCursor = ref<number>(0)

  let loadPromise: Promise<void> | null = null
  let loadMorePromise: Promise<void> | null = null

  function sort(conv: Conversation[]): Conversation[] {
    return sortConversationsByActivity(conv)
  }

  /** 刷新首屏会话列表 */
  async function loadConversations() {
    if (loadPromise) return loadPromise
    loading.value = true
    error.value = null

    loadPromise = (async () => {
      try {
        // 简易分页：每次加载 PAGE_SIZE 条
        const sdk = useIMStore().imSDK
        if (!sdk) return
        const list = await sdk.getConversations()
        // TODO: 服务端分页后期接入
        conversations.value = sort(list || [])
        hasMore.value = false
      } catch (e: any) {
        error.value = '加载会话失败'
      } finally {
        loading.value = false
        loadPromise = null
      }
    })()
    return loadPromise
  }

  /** 收到 WS 新消息时更新侧栏 */
  function onIncomingMessage(message: Message, activeChatId?: string) {
    conversations.value = applyIncomingMessage(conversations.value, message, activeChatId)
  }

  /** 清除指定会话未读 */
  function clearUnread(chatId: string) {
    conversations.value = withClearedUnreadForConversation(conversations.value, chatId)
  }

  /** 重置全部状态（登出时调用） */
  function reset() {
    conversations.value = []
    loading.value = false
    loadingMore.value = false
    error.value = null
    hasMore.value = true
    nextCursor.value = 0
    loadPromise = null
    loadMorePromise = null
  }

  return {
    conversations,
    loading,
    loadingMore,
    error,
    hasMore,
    loadConversations,
    onIncomingMessage,
    clearUnread,
    reset,
  }
})

import { useIMStore } from './im'