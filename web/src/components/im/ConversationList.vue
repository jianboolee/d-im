<template>
  <div ref="rootRef" class="conversation-list-root" :class="{ 'is-embedded': embedded }">
    <div v-if="displayLoading" class="state-block">
      <div class="spinner"></div>
    </div>

    <div v-else class="conversation-list" @scroll="handleScroll">
      <div v-if="displayError" class="inline-error">
        <div class="inline-error-main">
          <i class="ri-error-warning-line"></i>
          <span>{{ errorText }}</span>
        </div>
        <button type="button" @click.stop="refresh">重试</button>
      </div>

      <div
        v-for="item in conversationItems"
        :key="item.id"
        :data-conversation-id="item.id"
        class="conversation-item"
        :class="{ 'is-active': activeConversationId === item.id }"
        @click="selectConversation(item)"
      >
        <div class="avatar">
          <img v-if="item.avatar" :src="item.avatar" alt="">
          <PlaceholderImage
            v-else
            bgColor="#EFF1F8"
            text=""
            aspect="1:1"
          />
          <span v-if="item.unreadCount > 0 && !item.muted" class="unread-badge">
            {{ item.unreadBadge }}
          </span>
          <span v-else-if="item.unreadCount > 0" class="unread-dot"></span>
        </div>

        <div class="conversation-info">
          <div class="conversation-header">
            <span class="nickname">{{ item.displayName }}</span>
            <span class="time">{{ item.time }}</span>
          </div>
          <div class="last-message-row">
            <div class="last-message">{{ item.lastMessagePreview }}</div>
            <i v-if="item.muted" class="ri-notification-off-line muted-icon" aria-label="消息免打扰"></i>
          </div>
        </div>

        <div v-if="item.previewImage" class="conversation-image">
          <img :src="item.previewImage" alt="">
        </div>
      </div>

      <div v-if="conversationItems.length === 0 && !displayError" class="empty-state">
        <i :class="isSearching ? 'ri-search-line' : 'ri-chat-3-line'"></i>
        <p>{{ emptyText }}</p>
      </div>

      <div v-else-if="displayLoadingMore" class="list-footer">
        <div class="spinner"></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useIMStore } from '@/stores/im'
import { useConversationList } from '@/composables/useConversationList'
import { useUserProfiles } from '@/composables/useUserProfiles'
import {
  formatConversationTime,
  formatLastMessagePreview,
  formatUnreadBadge,
} from '@/utils/im/format'
import {
  getConversationDisplayAvatar,
  getConversationDisplayName,
  getPeerUserId,
  getUnreadCount,
} from '@/utils/im/conversation'
import PlaceholderImage from '@/components/common/PlaceholderImage.vue'
import type { Conversation } from '@/sdk/im'

const props = withDefaults(
  defineProps<{
    /** 当前会话，用于高亮 */
    activeConversationId?: string
    /** 嵌入聊天页侧栏时使用（仅影响样式） */
    embedded?: boolean
    /** 点击会话后的路由行为：home 用 push，侧栏用 replace */
    navigateMode?: 'push' | 'replace' | 'none'
    /** 会话搜索关键词 */
    searchKeyword?: string
    /** 仅用于搜索场景：空关键词时不展示普通会话 */
    searchMode?: boolean
  }>(),
  {
    activeConversationId: '',
    embedded: false,
    navigateMode: 'push',
    searchKeyword: '',
    searchMode: false,
  },
)

const emit = defineEmits<{
  select: [conversationId: string]
}>()

const router = useRouter()
const imStore = useIMStore()

const {
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
  currentUserId,
  loadConversations,
  loadMoreConversations,
  searchConversations,
  loadMoreSearchConversations,
  handleIncomingMessage,
  ensureConversationInList,
  requestScrollToConversation,
  pendingScrollRequest,
} = useConversationList()

const { userMap, mergeUsers } = useUserProfiles()
const rootRef = ref<HTMLElement | null>(null)
let searchTimer: number | undefined

const normalizedSearchKeyword = computed(() => props.searchKeyword.trim())
const isSearching = computed(() => props.searchMode || normalizedSearchKeyword.value.length > 0)
const hasSearchKeyword = computed(() => normalizedSearchKeyword.value.length > 0)
const displayConversations = computed(() => (
  isSearching.value ? searchResults.value : conversations.value
))
const displayLoading = computed(() => (
  isSearching.value ? hasSearchKeyword.value && searching.value : loading.value
))
const displayLoadingMore = computed(() => (
  isSearching.value ? searchingMore.value : loadingMore.value
))
const displayError = computed(() => (isSearching.value ? searchError.value : error.value))
const displayHasMore = computed(() => (isSearching.value ? searchHasMore.value : hasMore.value))

const errorText = computed(() => {
  if (displayError.value === '未登录') return '登录状态已失效'
  if (displayError.value === '加载更多会话失败' || displayError.value === '加载更多搜索会话失败') {
    return '加载更多失败'
  }
  if (displayError.value === '搜索会话失败') return '搜索失败'
  return '会话列表加载失败'
})

const emptyText = computed(() => {
  if (!isSearching.value) return '暂无消息'
  return hasSearchKeyword.value ? '没有找到相关会话' : '搜索联系人或用户 ID'
})

const conversationItems = computed(() => {
  const uid = currentUserId.value

  return displayConversations.value.map((conversation) => {
    const peerId = getPeerUserId(conversation, uid)
    const profile = conversation.peer_user_info ?? userMap.value[peerId]
    const unreadCount = getUnreadCount(conversation, uid)

    return {
      id: conversation.id,
      peerId,
      conversation,
      avatar: getConversationDisplayAvatar(conversation) || profile?.avatar,
      displayName: getConversationDisplayName(conversation, uid),
      lastMessagePreview: formatLastMessagePreview(conversation.last_message, conversation, uid),
      time: formatConversationTime(
        conversation.last_message?.client_time ?? conversation.updated_at,
      ),
      unreadCount,
      unreadBadge: formatUnreadBadge(unreadCount),
      muted: Boolean(conversation.muted),
      previewImage: conversation.preview_image_url,
    }
  })
})

const mergeConversationUsers = (items: Conversation[]) => {
  const users = items.flatMap((conversation) => [
    conversation.peer_user_info,
  ])
  mergeUsers(users)
}

const scrollToConversation = (conversationId: string) => {
  window.requestAnimationFrame(() => {
    const items = rootRef.value?.querySelectorAll<HTMLElement>('.conversation-item') ?? []
    const target = Array.from(items).find((item) => item.dataset.conversationId === conversationId)
    target?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  })
}

const selectConversation = async (item: { id: string; peerId: string; conversation: Conversation }) => {
  if (!item.id) return

  if (props.searchMode && item.id === props.activeConversationId) {
    requestScrollToConversation(item.id)
    emit('select', item.id)
    return
  }

  if (item.id === props.activeConversationId) {
    emit('select', '')
    if (props.navigateMode === 'none') return
    const emptyLocation = { name: 'im-chat-index' as const }
    if (props.navigateMode === 'replace') {
      router.replace(emptyLocation)
      return
    }
    router.push(emptyLocation)
    return
  }

  emit('select', item.id)

  if (props.navigateMode === 'none') return

  const location = { name: 'im-chat' as const, params: { conversationId: item.id } }
  if (props.navigateMode === 'replace') {
    router.replace(location)
    return
  }

  router.push(location)
}

const onIncomingMessage = async (message: Parameters<typeof handleIncomingMessage>[0]) => {
  mergeUsers([message.sender_profile])
  await handleIncomingMessage(message, props.activeConversationId || undefined)
  mergeConversationUsers(conversations.value)
}

const refresh = async () => {
  if (isSearching.value) {
    await searchConversations(normalizedSearchKeyword.value)
    mergeConversationUsers(searchResults.value)
    return
  }
  await loadConversations()
  mergeConversationUsers(conversations.value)
}

const loadMore = async () => {
  if (!displayHasMore.value || displayLoadingMore.value) return
  if (isSearching.value) {
    await loadMoreSearchConversations()
    mergeConversationUsers(searchResults.value)
    return
  }
  await loadMoreConversations()
  mergeConversationUsers(conversations.value)
}

const handleScroll = (event: Event) => {
  const target = event.target as HTMLElement
  if (!target || displayLoading.value || displayLoadingMore.value || !displayHasMore.value) return

  const distanceToBottom = target.scrollHeight - target.scrollTop - target.clientHeight
  if (distanceToBottom < 80) {
    void loadMore()
  }
}

onMounted(async () => {
  imStore.initSDK()
  imStore.addMessageHandler(onIncomingMessage)
  if (props.searchMode && !hasSearchKeyword.value) return
  await refresh()
})

onUnmounted(() => {
  if (searchTimer) {
    window.clearTimeout(searchTimer)
  }
  imStore.removeMessageHandler(onIncomingMessage)
})

watch(
  () => imStore.imSDK,
  (sdk, prev) => {
    if (props.searchMode && !hasSearchKeyword.value) return
    if (sdk && !prev && conversations.value.length === 0 && !loading.value) {
      void refresh()
    }
  },
)

watch(
  normalizedSearchKeyword,
  (keyword) => {
    if (searchTimer) {
      window.clearTimeout(searchTimer)
    }
    searchTimer = window.setTimeout(async () => {
      await searchConversations(keyword)
      mergeConversationUsers(searchResults.value)
    }, keyword ? 300 : 0)
  },
)

watch(
  pendingScrollRequest,
  async (request) => {
    if (!request?.conversationId) return
    await ensureConversationInList(request.conversationId)
    await nextTick()
    scrollToConversation(request.conversationId)
  },
)

defineExpose({ refresh })
</script>

<style scoped>
.conversation-list-root {
  display: flex;
  flex-direction: column;
  min-height: 0;
  flex: 1;
}

.conversation-list-root.is-embedded {
  background: var(--bg-color);
}

.state-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--spacing-base);
  padding: 48px var(--spacing-base);
  color: var(--text-color-light);
}

.spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--border-color);
  border-top-color: var(--primary-color);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.conversation-list {
  flex: 1;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
}

.inline-error {
  margin: 10px var(--spacing-base);
  padding: 9px 10px;
  border: 1px solid #ffe1df;
  border-radius: 8px;
  background: #fff7f6;
  color: #b94a44;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.inline-error-main {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-size: 13px;
  line-height: 1.4;
}

.inline-error-main i {
  flex-shrink: 0;
  font-size: 16px;
  line-height: 1;
}

.inline-error-main span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.inline-error button {
  flex-shrink: 0;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #4b86f8;
  padding: 4px 6px;
  font-size: 13px;
  cursor: pointer;
}

.inline-error button:hover {
  background: rgba(75, 134, 248, 0.08);
}

.conversation-item {
  display: flex;
  align-items: flex-start;
  padding: 10px var(--spacing-base);
  background: transparent;
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.is-embedded .conversation-item:hover {
  background: white;
}

.is-embedded .conversation-item.is-active {
  background: var(--bg-color-gray);
}

.conversation-list-root:not(.is-embedded) .conversation-item {
  background: white;
}

.avatar {
  position: relative;
  width: 48px;
  height: 48px;
  margin-right: var(--spacing-base);
  background: var(--bg-color);
  border-radius: 50%;
  flex-shrink: 0;
}

.avatar img {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  object-fit: cover;
}

.unread-badge {
  position: absolute;
  top: -4px;
  right: -4px;
  min-width: 18px;
  height: 18px;
  padding: 0 4px;
  background: var(--error-color);
  color: white;
  border-radius: 9px;
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.unread-dot {
  position: absolute;
  top: 0;
  right: 0;
  width: 12px;
  height: 12px;
  border: 2px solid #fff;
  border-radius: 50%;
  background: var(--error-color);
}

.conversation-info {
  flex: 1;
  min-width: 0;
  padding-bottom: 2px;
}

.conversation-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.nickname {
  font-weight: 700;
  font-size: var(--font-size-base);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.time {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--text-color-light);
}

.conversation-image {
  margin-left: var(--spacing-small);
  background-color: var(--bg-color);
  overflow: hidden;
  width: 48px;
  max-height: 48px;
  border-radius: 8px;
  aspect-ratio: 1 / 1;
  flex-shrink: 0;
}

.conversation-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.last-message-row {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.last-message {
  min-width: 0;
  flex: 1;
  color: var(--text-color-secondary);
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.muted-icon {
  flex-shrink: 0;
  color: var(--text-color-light);
  font-size: 12px;
  line-height: 1;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px var(--spacing-base);
  color: var(--text-color-light);
}

.empty-state i {
  font-size: 48px;
  margin-bottom: var(--spacing-base);
}

.empty-state p {
  font-size: var(--font-size-base);
  margin: 0;
}

.list-footer {
  display: flex;
  justify-content: center;
  padding: 16px;
}
</style>
