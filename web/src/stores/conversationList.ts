import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { useIMStore } from '@/stores/im'
import { useIMTabStore } from '@/stores/imTab'
import { useUserStore } from '@/stores/user'
import type { Conversation, Message } from '@/sdk/im'
import {
  applyIncomingMessage,
  collectPeerUserIds,
  sortConversationsByActivity,
  withClearedUnreadForConversation,
  withClearedUnreadForPeer,
} from '@/utils/im/conversation'

const PAGE_SIZE = 20

export const useConversationListStore = defineStore('conversationList', () => {
  const conversations = ref<Conversation[]>([])
  const searchResults = ref<Conversation[]>([])
  const loading = ref(false)
  const loadingMore = ref(false)
  const searching = ref(false)
  const searchingMore = ref(false)
  const error = ref<string | null>(null)
  const searchError = ref<string | null>(null)
  const hasMore = ref(true)
  const nextCursor = ref<string | undefined>()
  const searchHasMore = ref(false)
  const searchNextCursor = ref<string | undefined>()
  const activeSearchKeyword = ref('')
  const pendingScrollRequest = ref<{ chatId: string; nonce: number } | null>(null)
  const currentUserId = computed(() => useUserStore().userInfo?.id ?? '')

  let loadPromise: Promise<void> | null = null
  let loadMorePromise: Promise<void> | null = null
  let searchPromise: Promise<void> | null = null
  let searchMorePromise: Promise<void> | null = null
  let searchRequestId = 0
  let scrollRequestNonce = 0
  const ensurePromises = new Map<string, Promise<Conversation | null>>()

  function ensureImSDK() {
    const imStore = useIMStore()
    const userStore = useUserStore()
    if (imStore.imSDK) {
      return imStore.imSDK
    }
    if (!userStore.token) {
      return null
    }
    return imStore.initSDK()
  }

  function upsertConversation(conversation: Conversation) {
    if (!conversation.id) return

    const byId = new Map(conversations.value.map((item) => [item.id, item]))
    byId.set(conversation.id, conversation)
    conversations.value = sortConversationsByActivity([...byId.values()])
  }

  function updateConversationMemberState(conversationId: string, patch: Partial<Conversation>) {
    if (!conversationId || !patch) return
    const applyMemberState = (conversation: Conversation) => (
      conversation.id === conversationId
        ? { ...conversation, ...patch }
        : conversation
    )
    conversations.value = sortConversationsByActivity(conversations.value.map(applyMemberState))
    searchResults.value = sortConversationsByActivity(searchResults.value.map(applyMemberState))
  }

  function updateConversationGroupInfo(conversationId: string, group: NonNullable<Conversation['group_info']>) {
    if (!conversationId || !group?.id) return

    const applyGroupInfo = (conversation: Conversation) => {
      if (conversation.id !== conversationId) return conversation
      return {
        ...conversation,
        display_name: group.name || conversation.display_name,
        display_avatar: group.avatar_url || conversation.display_avatar,
        group_info: {
          ...conversation.group_info,
          ...group,
        },
      }
    }

    conversations.value = sortConversationsByActivity(conversations.value.map(applyGroupInfo))
    searchResults.value = sortConversationsByActivity(searchResults.value.map(applyGroupInfo))
  }

  function removeConversation(conversationId: string) {
    if (!conversationId) return
    conversations.value = conversations.value.filter((conversation) => conversation.id !== conversationId)
    searchResults.value = searchResults.value.filter((conversation) => conversation.id !== conversationId)
  }

  async function loadConversations(options: { reset?: boolean; activeChatId?: string } = {}) {
    if (loadPromise) {
      return loadPromise
    }

    if (options.reset !== false) {
      nextCursor.value = undefined
      hasMore.value = true
    }

    loading.value = true
    error.value = null

    loadPromise = (async () => {
      try {
        const sdk = ensureImSDK()
        if (!sdk) {
          if (!useIMTabStore().isSuspended) {
            error.value = '未登录'
          }
          return
        }

        const page = await sdk.getConversationPage({
          limit: PAGE_SIZE,
          active_chat_id: options.activeChatId,
        })
        conversations.value = sortConversationsByActivity(page.items ?? [])
        nextCursor.value = page.next_cursor
        hasMore.value = page.has_more
      } catch (err) {
        console.error('获取会话列表失败:', err)
        error.value = '获取会话列表失败'
      } finally {
        loading.value = false
        loadPromise = null
      }
    })()

    return loadPromise
  }

  async function loadMoreConversations() {
    if (loadPromise) {
      await loadPromise
    }
    if (loadMorePromise) {
      return loadMorePromise
    }
    if (!hasMore.value || !nextCursor.value || loading.value) {
      return
    }

    loadingMore.value = true
    error.value = null

    loadMorePromise = (async () => {
      try {
        const sdk = ensureImSDK()
        if (!sdk) {
          if (!useIMTabStore().isSuspended) {
            error.value = '未登录'
          }
          return
        }

        const page = await sdk.getConversationPage({
          limit: PAGE_SIZE,
          cursor: nextCursor.value,
        })
        const byId = new Map(conversations.value.map((conversation) => [conversation.id, conversation]))
        for (const conversation of page.items ?? []) {
          byId.set(conversation.id, conversation)
        }
        conversations.value = sortConversationsByActivity([...byId.values()])
        nextCursor.value = page.next_cursor
        hasMore.value = page.has_more
      } catch (err) {
        console.error('加载更多会话失败:', err)
        error.value = '加载更多会话失败'
      } finally {
        loadingMore.value = false
        loadMorePromise = null
      }
    })()

    return loadMorePromise
  }

  async function searchConversations(keyword: string) {
    const normalizedKeyword = keyword.trim()
    activeSearchKeyword.value = normalizedKeyword
    searchNextCursor.value = undefined
    searchHasMore.value = false
    searchError.value = null
    searchingMore.value = false
    searchMorePromise = null

    const requestId = ++searchRequestId
    if (!normalizedKeyword) {
      searchResults.value = []
      searching.value = false
      searchPromise = null
      return
    }

    searching.value = true

    searchPromise = (async () => {
      try {
        const sdk = ensureImSDK()
        if (!sdk) {
          if (!useIMTabStore().isSuspended && requestId === searchRequestId) {
            searchError.value = '未登录'
          }
          return
        }

        const page = await sdk.getConversationPage({
          limit: PAGE_SIZE,
          q: normalizedKeyword,
        })
        if (requestId !== searchRequestId) return

        searchResults.value = sortConversationsByActivity(page.items ?? [])
        searchNextCursor.value = page.next_cursor
        searchHasMore.value = page.has_more
      } catch (err) {
        if (requestId !== searchRequestId) return
        console.error('搜索会话失败:', err)
        searchError.value = '搜索会话失败'
      } finally {
        if (requestId === searchRequestId) {
          searching.value = false
          searchPromise = null
        }
      }
    })()

    return searchPromise
  }

  async function loadMoreSearchConversations() {
    if (searchPromise) {
      await searchPromise
    }
    if (searchMorePromise) {
      return searchMorePromise
    }
    if (!activeSearchKeyword.value || !searchHasMore.value || !searchNextCursor.value || searching.value) {
      return
    }

    const requestId = searchRequestId
    searchingMore.value = true
    searchError.value = null

    searchMorePromise = (async () => {
      try {
        const sdk = ensureImSDK()
        if (!sdk) {
          if (!useIMTabStore().isSuspended && requestId === searchRequestId) {
            searchError.value = '未登录'
          }
          return
        }

        const page = await sdk.getConversationPage({
          limit: PAGE_SIZE,
          cursor: searchNextCursor.value,
          q: activeSearchKeyword.value,
        })
        if (requestId !== searchRequestId) return

        const byId = new Map(searchResults.value.map((conversation) => [conversation.id, conversation]))
        for (const conversation of page.items ?? []) {
          byId.set(conversation.id, conversation)
        }
        searchResults.value = sortConversationsByActivity([...byId.values()])
        searchNextCursor.value = page.next_cursor
        searchHasMore.value = page.has_more
      } catch (err) {
        if (requestId !== searchRequestId) return
        console.error('加载更多搜索会话失败:', err)
        searchError.value = '加载更多搜索会话失败'
      } finally {
        if (requestId === searchRequestId) {
          searchingMore.value = false
          searchMorePromise = null
        }
      }
    })()

    return searchMorePromise
  }

  async function handleIncomingMessage(message: Message, activeChatId?: string) {
    const userId = currentUserId.value
    if (!userId) return
    let knownConversation = message.conversation_id || message.chat_id
      ? conversations.value.some((conversation) => (
        (message.conversation_id && conversation.id === message.conversation_id)
        || (message.chat_id && conversation.chat_id === message.chat_id)
      ))
      : false

    if (!knownConversation && message.chat_id) {
      const conversation = await ensureConversationByChatId(message.chat_id)
      knownConversation = Boolean(conversation)
    } else if (!knownConversation && message.conversation_id) {
      const conversation = await ensureConversationInList(message.conversation_id)
      knownConversation = Boolean(conversation)
    }

    if (!knownConversation) return
    conversations.value = applyIncomingMessage(
      conversations.value,
      message,
      userId,
      activeChatId,
    )
  }

  function clearUnreadForPeer(peerId: string) {
    const userId = currentUserId.value
    if (!userId || !peerId) return

    conversations.value = withClearedUnreadForPeer(conversations.value, peerId, userId)
  }

  function clearUnreadForConversation(conversationId: string) {
    conversations.value = withClearedUnreadForConversation(conversations.value, conversationId)
  }

  async function ensureConversationInList(conversationId: string) {
    if (!conversationId) return null

    const existing = conversations.value.find((conversation) => conversation.id === conversationId)
    if (existing) return existing

    if (loadPromise) {
      await loadPromise
      const loaded = conversations.value.find((conversation) => conversation.id === conversationId)
      if (loaded) return loaded
    }

    const cacheKey = conversationId
    const pending = ensurePromises.get(cacheKey)
    if (pending) return pending

    const promise = (async () => {
      const sdk = ensureImSDK()
      if (!sdk) return null

      const conversation = await sdk.getConversation(conversationId)
      upsertConversation(conversation)
      return conversation
    })()

    ensurePromises.set(cacheKey, promise)
    try {
      return await promise
    } finally {
      ensurePromises.delete(cacheKey)
    }
  }

  async function ensureConversationByChatId(chatId: string) {
    if (!chatId) return null

    const existing = conversations.value.find((conversation) => conversation.chat_id === chatId)
    if (existing) return existing

    if (loadPromise) {
      await loadPromise
      const loaded = conversations.value.find((conversation) => conversation.chat_id === chatId)
      if (loaded) return loaded
    }

    const cacheKey = `chat:${chatId}`
    const pending = ensurePromises.get(cacheKey)
    if (pending) return pending

    const promise = (async () => {
      const sdk = ensureImSDK()
      if (!sdk) return null

      const conversation = await sdk.getConversationByChatId(chatId)
      upsertConversation(conversation)
      return conversation
    })()

    ensurePromises.set(cacheKey, promise)
    try {
      return await promise
    } finally {
      ensurePromises.delete(cacheKey)
    }
  }

  function requestScrollToConversation(chatId: string) {
    if (!chatId) return
    pendingScrollRequest.value = {
      chatId,
      nonce: ++scrollRequestNonce,
    }
  }

  function getPeerUserIds() {
    return collectPeerUserIds(conversations.value, currentUserId.value)
  }

  function resetConversations() {
    conversations.value = []
    searchResults.value = []
    error.value = null
    searchError.value = null
    loading.value = false
    loadingMore.value = false
    searching.value = false
    searchingMore.value = false
    hasMore.value = true
    nextCursor.value = undefined
    searchHasMore.value = false
    searchNextCursor.value = undefined
    activeSearchKeyword.value = ''
    pendingScrollRequest.value = null
    loadPromise = null
    loadMorePromise = null
    searchPromise = null
    searchMorePromise = null
    ensurePromises.clear()
    searchRequestId += 1
  }

  return {
    conversations,
    searchResults,
    loading,
    loadingMore,
    searching,
    searchingMore,
    error,
    searchError,
    hasMore,
    searchHasMore,
    pendingScrollRequest,
    currentUserId,
    loadConversations,
    loadMoreConversations,
    searchConversations,
    loadMoreSearchConversations,
    handleIncomingMessage,
    clearUnreadForPeer,
    clearUnreadForConversation,
    upsertConversation,
    updateConversationMemberState,
    updateConversationGroupInfo,
    removeConversation,
    ensureConversationInList,
    ensureConversationByChatId,
    requestScrollToConversation,
    getPeerUserIds,
    resetConversations,
  }
})
