<template>
  <div class="chat-page">
    <div class="chat-layout">
      <aside class="chat-sidebar">
        <div class="sidebar-header">
          <span class="sidebar-title">消息</span>
          <div class="sidebar-actions">
            <button class="sidebar-icon-btn" type="button" aria-label="新建会话" @click="openNewConversationModal">
              <i class="ri-add-line"></i>
            </button>
            <button class="sidebar-icon-btn" type="button" aria-label="搜索会话" @click="openConversationSearch">
              <i class="ri-search-line"></i>
            </button>
          </div>
        </div>
        <ConversationList
          embedded
          navigate-mode="replace"
          :active-chat-id="routeChatId"
        />
        <div ref="sidebarMenuRef" class="sidebar-footer">
          <button
            class="sidebar-footer-trigger"
            type="button"
            :class="{ 'is-active': showSidebarMenu }"
            aria-label="更多"
            @click="showSidebarMenu = !showSidebarMenu"
          >
            <div class="sidebar-footer-user" :title="userStore.userInfo?.nickname || '当前用户'">
              <img
                v-if="userStore.userInfo?.avatar"
                class="sidebar-footer-avatar"
                :src="userStore.userInfo.avatar"
                alt=""
              >
              <div v-else class="sidebar-footer-avatar sidebar-footer-avatar-fallback">
                <i class="ri-user-3-line"></i>
              </div>
              <span class="sidebar-footer-name">
                {{ userStore.userInfo?.nickname || '当前用户' }}
              </span>
            </div>
            <span class="sidebar-menu-btn" aria-hidden="true">
              <i class="ri-menu-line"></i>
            </span>
          </button>
          <div v-if="showSidebarMenu" class="sidebar-menu">
            <div class="sidebar-user">
              <img
                class="sidebar-user-avatar"
                :src="userStore.userInfo?.avatar || ''"
                alt=""
              >
              <span class="sidebar-user-name">
                {{ userStore.userInfo?.nickname || '当前用户' }}
              </span>
            </div>
            <button type="button" class="sidebar-menu-item" @click="handleLogout">
              <i class="ri-logout-box-r-line"></i>
              <span>退出登录</span>
            </button>
          </div>
        </div>
      </aside>

      <div v-if="imTabStore.isSuspended" class="chat-main chat-suspended-main">
        <div class="chat-suspended-state">
          <i class="ri-computer-line"></i>
          <p>已在其他标签页打开</p>
          <button type="button" @click="handleTakeoverTab">在此标签页使用</button>
        </div>
      </div>
      <div v-else-if="hasSelectedConversation" class="chat-main">
    <div class="nav-bar">
      <div class="nav-bar-content">
        <button
          v-if="isMobileViewport"
          class="nav-side-btn back-btn"
          type="button"
          aria-label="返回"
          @click="handleBack"
        >
          <i class="ri-arrow-left-s-line"></i>
        </button>
        <div class="nav-bar-center">
          <div class="user-info">
            <h1 class="title">{{ conversationTitle }}</h1>
          </div>
          <i v-if="!isConnected"  class="ri-loader-4-line nav-reconnect-icon" aria-label="连接中"></i>
        </div>
        <button class="nav-side-btn" type="button" aria-label="会话信息" @click="openConversationInfo">
          <i class="ri-more-line"></i>
        </button>
      </div>
    </div>

    <div class="message-container" @click="showSidebarMenu = false">
      <div ref="messageListRef" class="message-list" @scroll="handleScroll">
        <div v-if="loading" class="loading-spinner">
          <div class="spinner"></div>
        </div>
        <div
          v-if="!hasMore && !firstLoad && messages.length > pageSize"
          class="no-more-messages"
        >
          没有更多消息了
        </div>
        <template v-for="item in timelineItems" :key="item.id">
          <div v-if="item.type === 'time'" class="message-time-divider">
            {{ item.text }}
          </div>
          <SystemEventMessage
            v-else-if="isSystemEventMessage(item.message)"
            :message="item.message"
          />
          <div
            v-else
            class="message-item"
            :class="{ 'message-mine': item.message.sender_id === currentUserId }"
          >
            <div v-if="getMessageAvatar(item.message)" class="message-avatar">
              <img
                :src="getMessageAvatar(item.message)"
                alt=""
              >
            </div>
            <div class="message-wrapper">
              <div v-if="shouldShowSenderName(item.message)" class="message-sender-name">
                {{ getMessageSenderName(item.message) }}
              </div>
              <component
                :is="MessageComponents[item.message.type || MessageType.Text] ?? MessageComponents[MessageType.Text]"
                :message="item.message"
                :isMine="item.message.sender_id === currentUserId"
                @retry="retryMessage(item.message)"
              />
            </div>
          </div>
        </template>
      </div>
    </div>

    <div v-if="peerUserType === 'system'" class="system-notice-bar">
      <i class="ri-information-line"></i>
      <span>系统消息，暂不支持回复</span>
    </div>
    <div v-else class="message-input-container">
      <div class="message-input">
        <div class="message-input-content">
          <MultilineInput
            v-model="messageText"
            placeholder="发消息..."
            :minRows="inputMinRows"
            :maxRows="inputMaxRows"
            @enter="sendMessage"
            @focus="handleMessageInputFocus"
          />
          <div class="message-input-actions">
            <MessageMoreOptions
              @select-file="handleSelectFile"
              @upload-success="handleUploadSuccess"
              @upload-error="handleUploadError"
            />
            <button
              type="button"
              class="send-btn"
              :disabled="!messageText.trim()"
              @click="sendMessage"
            >
              <i class="ri-arrow-up-line"></i>
            </button>
          </div>
        </div>
      </div>
    </div>
      </div>
      <div v-else class="chat-main chat-empty-main">
        <div class="chat-empty-state">
          <i class="ri-chat-3-line"></i>
        </div>
      </div>
    </div>
    <ConversationSearchModal
      v-model="showConversationSearch"
      navigate-mode="replace"
      :active-chat-id="routeChatId"
    />
    <Teleport to="body">
      <div v-if="showNewConversationModal" class="new-conversation-modal" @click.self="closeNewConversationModal">
        <form class="new-conversation-panel" role="dialog" aria-modal="true" aria-label="新建会话" @submit.prevent="submitNewConversation">
          <div class="new-conversation-header">
            <h2>新建会话</h2>
            <button class="new-conversation-close" type="button" aria-label="关闭" @click="closeNewConversationModal">
              <i class="ri-close-line"></i>
            </button>
          </div>
          <div class="new-conversation-tabs" role="tablist" aria-label="新建会话类型">
            <button
              type="button"
              :class="{ 'is-active': newConversationMode === 'single' }"
              @click="newConversationMode = 'single'"
            >
              单聊
            </button>
            <button
              type="button"
              :class="{ 'is-active': newConversationMode === 'group' }"
              @click="newConversationMode = 'group'"
            >
              群聊
            </button>
          </div>
          <label v-if="newConversationMode === 'single'" class="new-conversation-field">
            <span>用户 ID</span>
            <input
              ref="newConversationInputRef"
              v-model="newConversationUserId"
              type="text"
              autocomplete="off"
              placeholder="输入用户 ID"
              @keydown.esc="closeNewConversationModal"
            >
          </label>
          <template v-else>
            <label class="new-conversation-field">
              <span>群名称</span>
              <input
                ref="newConversationInputRef"
                v-model="newGroupName"
                type="text"
                autocomplete="off"
                placeholder="输入群名称"
                @keydown.esc="closeNewConversationModal"
              >
            </label>
            <label class="new-conversation-field">
              <span>成员用户 ID</span>
              <textarea
                v-model="newGroupMemberIdsText"
                rows="4"
                placeholder="输入用户 ID，多个 ID 可用逗号、空格或换行分隔"
                @keydown.esc="closeNewConversationModal"
              ></textarea>
            </label>
          </template>
          <p v-if="newConversationError" class="new-conversation-error">{{ newConversationError }}</p>
          <div class="new-conversation-actions">
            <button class="new-conversation-cancel" type="button" @click="closeNewConversationModal">
              取消
            </button>
            <button class="new-conversation-submit" type="submit" :disabled="creatingConversation || !canSubmitNewConversation">
              {{ creatingConversation ? '创建中' : '创建' }}
            </button>
          </div>
        </form>
      </div>
    </Teleport>
    <GroupInviteDrawer
      v-model="showInviteMembersDrawer"
      :loading="invitingMembers"
      :error="inviteMembersError"
      :current-user-id="currentUserId"
      @submit="inviteMembers"
    />
    <ConversationInfoPanel
      v-model="showConversationInfoDrawer"
      :conversation="conversation"
      :participants="conversationInfoParticipants"
      @invite="handleInviteMembers"
    />
  </div>
</template>

<script setup lang="ts">
import { defineAsyncComponent, ref, computed, nextTick, watch, provide, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { showToast } from '@/plugins/toast'
import { useUserStore } from '@/stores/user'
import { useIMStore } from '@/stores/im'
import { useIMTabStore } from '@/stores/imTab'
import { useConversationList } from '@/composables/useConversationList'
import { useUserProfiles } from '@/composables/useUserProfiles'
import { MessageType } from '@/sdk/im'
import type { ChatMessage, Conversation, MessageContent, UploadState } from '@/types/im'
import type { UserInfo } from '@/types/user'
import { MessageComponents } from '@/components/im'
import MultilineInput from '@/components/im/MultilineInput.vue'
import ConversationList from '@/components/im/ConversationList.vue'
import SystemEventMessage from '@/components/im/SystemEventMessage.vue'
import { usePageTitleNotification } from '@/composables/usePageTitleNotification'
import { buildMessageTimeline } from '@/utils/im/timeline'
import {
  getConversationDisplayName,
  getPeerUserId,
  isGroupConversation as isGroupConversationModel,
} from '@/utils/im/conversation'
import { ReadReporter } from '@/utils/im/readReporter'
import { readImageDimensions, getFileFormat } from '@/utils/file'
import { uploadIMFile } from '@/utils/upload'

const props = defineProps<{
  chatId: string
}>()

const MessageMoreOptions = defineAsyncComponent(() => import('@/components/im/MessageMoreOptions.vue'))
const ConversationSearchModal = defineAsyncComponent(() => import('@/components/im/ConversationSearchModal.vue'))
const ConversationInfoPanel = defineAsyncComponent(() => import('@/components/im/ConversationInfoPanel.vue'))
const GroupInviteDrawer = defineAsyncComponent(() => import('@/components/im/GroupInviteDrawer.vue'))

const router = useRouter()
const userStore = useUserStore()
const imStore = useIMStore()
const imTabStore = useIMTabStore()
const {
  conversations,
  clearUnreadForPeer,
  clearUnreadForConversation,
  handleIncomingMessage: updateConversationByMessage,
  ensureConversationByChatId,
  upsertConversation,
  updateConversationMemberState,
  updateConversationGroupInfo,
  requestScrollToConversation,
} = useConversationList()
const { userMap, fetchUser, mergeUsers } = useUserProfiles()

const messageText = ref('')
const messages = ref<ChatMessage[]>([])
const conversation = ref<Conversation | null>(null)
const targetUser = ref<UserInfo | null>(null)
const messageListRef = ref<HTMLElement | null>(null)
const sidebarMenuRef = ref<HTMLElement | null>(null)
const newConversationInputRef = ref<HTMLInputElement | null>(null)
const showSidebarMenu = ref(false)
const showConversationSearch = ref(false)
const showNewConversationModal = ref(false)
const showInviteMembersDrawer = ref(false)
const showConversationInfoDrawer = ref(false)
const newConversationMode = ref<'single' | 'group'>('single')
const newConversationUserId = ref('')
const newGroupName = ref('')
const newGroupMemberIdsText = ref('')
const newConversationError = ref('')
const creatingConversation = ref(false)
const inviteMembersError = ref('')
const invitingMembers = ref(false)
const isMobileViewport = ref(false)
let cleanupViewportListener: (() => void) | null = null
const pendingUploadMessageIds = new WeakMap<File, string>()
const messageDrafts = new Map<string, string>()
const readReporter = new ReadReporter(
  () => imStore.imSDK,
  (readState) => {
    if (conversation.value?.id === readState.conversation_id) {
      conversation.value = {
        ...conversation.value,
        last_read_sequence: readState.last_read_sequence,
        last_read_at: readState.read_at,
        unread_count: 0,
      }
    }
    updateConversationMemberState(readState.conversation_id, {
      last_read_sequence: readState.last_read_sequence,
      last_read_at: readState.read_at,
      unread_count: 0,
    })
  },
)

const isConnected = computed(() => imStore.isConnected)
const currentUserId = computed(() => userStore.userInfo?.id)
const routeChatId = computed(() => props.chatId)
const currentConversationId = computed(() => conversation.value?.id || '')
const hasSelectedConversation = computed(() => Boolean(routeChatId.value))
const inputMinRows = computed(() => (isMobileViewport.value ? 1 : 2))
const inputMaxRows = computed(() => (isMobileViewport.value ? 10 : 15))
const timelineItems = computed(() => buildMessageTimeline(messages.value))
const activeChatId = computed(() => conversation.value?.chat_id || routeChatId.value)
const isGroupConversation = computed(() => isGroupConversationModel(conversation.value))
const peerUserId = computed(() => {
  if (!conversation.value || isGroupConversation.value) return ''
  return conversation.value.peer_user_info?.id
    || getPeerUserId(conversation.value, currentUserId.value || '')
})
const peerUserType = computed(
  () => isGroupConversation.value
    ? ''
    : conversation.value?.peer_user_info?.type || userMap.value[peerUserId.value]?.type || '',
)
const conversationTitle = computed(() => (
  conversation.value
    ? getConversationDisplayName(conversation.value, currentUserId.value || '')
    : '-'
))
const conversationInfoParticipants = computed<UserInfo[]>(() => {
  const currentId = currentUserId.value
  if (isGroupConversation.value) {
    return []
  }

  const participantIds = conversation.value?.participants.filter((id) => id && id !== currentId) ?? []
  const usersById = new Map<string, UserInfo>()

  const peerInfo = conversation.value?.peer_user_info
  if (peerInfo?.id && peerInfo.id !== currentId) {
    usersById.set(peerInfo.id, peerInfo)
  }

  if (targetUser.value?.id && targetUser.value.id !== currentId) {
    usersById.set(targetUser.value.id, targetUser.value)
  }

  participantIds.forEach((id) => {
    usersById.set(id, userMap.value[id] ?? usersById.get(id) ?? { id })
  })

  return [...usersById.values()]
})
const canSubmitNewConversation = computed(() => {
  if (newConversationMode.value === 'single') {
    return Boolean(newConversationUserId.value.trim())
  }
  return Boolean(parseUserIds(newGroupMemberIdsText.value).length > 0)
})

const pageTitle = computed(() => conversationTitle.value || '消息')
const { setBaseTitle } = usePageTitleNotification('消息')

const mergeConversationUsers = (item: Conversation) => {
  mergeUsers([
    item.peer_user_info,
  ])
}

const getMessageAvatar = (message: ChatMessage) => {
  if (message.sender_id === currentUserId.value) {
    return userStore.userInfo?.avatar || ''
  }
  if (isGroupConversation.value) {
    return message.sender_profile?.avatar || userMap.value[message.sender_id || '']?.avatar || ''
  }
  return message.sender_profile?.avatar || targetUser.value?.avatar || ''
}

const isSystemEventMessage = (message: ChatMessage) => message.type === MessageType.SystemEvent

const shouldShowSenderName = (message: ChatMessage) => (
  isGroupConversation.value
  && message.sender_id !== currentUserId.value
  && !isSystemEventMessage(message)
)

const getMessageSenderName = (message: ChatMessage) => (
  message.sender_profile?.nickname
  || userMap.value[message.sender_id || '']?.nickname
  || message.sender_id
  || ''
)

watch(pageTitle, (title) => {
  setBaseTitle(title)
}, { immediate: true })

watch(messageText, (value) => {
  const id = routeChatId.value
  if (!id) return

  if (value) {
    messageDrafts.set(id, value)
  } else {
    messageDrafts.delete(id)
  }
})

watch(
  () => conversations.value.find(
    (item) => item.chat_id === routeChatId.value,
  ),
  (user) => {
    if (!user) return
    conversation.value = user
    mergeConversationUsers(user)
    const peerInfo = user.peer_user_info
    if (!isGroupConversationModel(user) && peerInfo) {
      mergeUsers([peerInfo])
      targetUser.value = peerInfo
    }
  },
)

const handleBack = () => {
  if (window.history.state?.back) {
    router.back()
    return
  }

  router.push({ name: 'im-home' })
}

const handleLogout = () => {
  showSidebarMenu.value = false
  imTabStore.reset()
  userStore.logout()
  router.replace({ name: 'im-login' })
}

const handleTakeoverTab = () => {
  imTabStore.claimActive()
  initChat()
}

const openConversationSearch = () => {
  showConversationSearch.value = true
  showSidebarMenu.value = false
}

const openNewConversationModal = () => {
  showNewConversationModal.value = true
  showSidebarMenu.value = false
  newConversationError.value = ''

  nextTick(() => {
    newConversationInputRef.value?.focus()
  })
}

const closeNewConversationModal = () => {
  if (creatingConversation.value) return
  showNewConversationModal.value = false
  newConversationMode.value = 'single'
  newConversationUserId.value = ''
  newGroupName.value = ''
  newGroupMemberIdsText.value = ''
  newConversationError.value = ''
}

const parseUserIds = (value: string) => Array.from(
  new Set(
    value
      .split(/[\s,，;；]+/)
      .map((item) => item.trim())
      .filter(Boolean),
  ),
)

const submitNewConversation = async () => {
  if (newConversationMode.value === 'group') {
    await createGroupConversation()
    return
  }
  await createSingleConversation()
}

const createSingleConversation = async () => {
  const peerUserId = newConversationUserId.value.trim()
  if (!peerUserId || creatingConversation.value) return

  if (peerUserId === currentUserId.value) {
    newConversationError.value = '不能和自己创建单聊'
    return
  }

  const sdk = imStore.imSDK ?? imStore.initSDK()
  if (!sdk) {
    newConversationError.value = '请先登录'
    return
  }

  creatingConversation.value = true
  newConversationError.value = ''

  try {
    const nextConversation = await sdk.activateConversation(peerUserId)
    upsertConversation(nextConversation)
    showNewConversationModal.value = false
    newConversationUserId.value = ''
    await router.replace({ name: 'im-chat', params: { chatId: nextConversation.chat_id } })
    requestScrollToConversation(nextConversation.chat_id)
  } catch (error) {
    console.error('创建会话失败:', error)
    newConversationError.value = error instanceof Error ? error.message : '创建会话失败'
  } finally {
    creatingConversation.value = false
  }
}

const createGroupConversation = async () => {
  const name = newGroupName.value.trim()
  const memberIds = parseUserIds(newGroupMemberIdsText.value)
    .filter((id) => id !== currentUserId.value)
  if (creatingConversation.value) return

  if (memberIds.length === 0) {
    newConversationError.value = '至少输入一个成员用户 ID'
    return
  }

  const sdk = imStore.imSDK ?? imStore.initSDK()
  if (!sdk) {
    newConversationError.value = '请先登录'
    return
  }

  creatingConversation.value = true
  newConversationError.value = ''

  try {
    const detail = await sdk.createGroup({
      name,
      member_user_ids: memberIds,
    })
    const nextConversation = detail.conversation
    if (!nextConversation?.id) {
      throw new Error('创建群聊失败')
    }
    const fullConversation = await sdk.getConversation(nextConversation.id)
    upsertConversation(fullConversation)
    showNewConversationModal.value = false
    newConversationMode.value = 'single'
    newGroupName.value = ''
    newGroupMemberIdsText.value = ''
    await router.replace({ name: 'im-chat', params: { chatId: fullConversation.chat_id } })
    requestScrollToConversation(fullConversation.chat_id)
    showToast('群聊已创建')
  } catch (error) {
    console.error('创建群聊失败:', error)
    newConversationError.value = error instanceof Error ? error.message : '创建群聊失败'
  } finally {
    creatingConversation.value = false
  }
}

const openConversationInfo = () => {
  showConversationInfoDrawer.value = true
  showSidebarMenu.value = false
}

const handleInviteMembers = () => {
  if (!isGroupConversation.value || !conversation.value?.group_id) {
    showToast('当前会话不支持邀请')
    return
  }
  showInviteMembersDrawer.value = true
  inviteMembersError.value = ''
}

const inviteMembers = async (memberIds: string[]) => {
  const groupId = conversation.value?.group_id
  const conversationIdValue = conversation.value?.id
  if (!groupId || !conversationIdValue || invitingMembers.value) return

  if (memberIds.length === 0) {
    inviteMembersError.value = '至少输入一个成员用户 ID'
    return
  }

  const sdk = imStore.imSDK ?? imStore.initSDK()
  if (!sdk) {
    inviteMembersError.value = '请先登录'
    return
  }

  invitingMembers.value = true
  inviteMembersError.value = ''

  try {
    const detail = await sdk.inviteGroupMembers(groupId, memberIds)
    if (detail.group) {
      updateConversationGroupInfo(conversationIdValue, {
        id: detail.group.id,
        name: detail.group.name,
        avatar_url: detail.group.avatar_url,
        member_count: detail.group.member_count,
      })
    }
    showInviteMembersDrawer.value = false
    showToast('邀请已发送')
  } catch (error) {
    console.error('邀请成员失败:', error)
    inviteMembersError.value = error instanceof Error ? error.message : '邀请成员失败'
  } finally {
    invitingMembers.value = false
  }
}

const handleDocumentPointerDown = (event: PointerEvent) => {
  if (!showSidebarMenu.value) return
  const target = event.target
  if (target instanceof Node && sidebarMenuRef.value?.contains(target)) {
    return
  }
  showSidebarMenu.value = false
}

const pageSize = 20
const loading = ref(false)
const hasMore = ref(true)
const firstLoad = ref(true)
const initialized = ref(false)

const chatImages = computed(() =>
  messages.value
    .filter((msg) => msg.type === MessageType.Image && msg.content.url)
    .map((msg) => msg.content.url!),
)

provide('chatImages', chatImages)

const isCurrentChatMessage = (message: ChatMessage) => {
  return Boolean(
    (message.chat_id && message.chat_id === activeChatId.value)
    || (message.conversation_id && message.conversation_id === currentConversationId.value),
  )
}

const isNearBottom = () => {
  if (!messageListRef.value) return false
  const { scrollTop, scrollHeight, clientHeight } = messageListRef.value
  return scrollHeight - scrollTop - clientHeight < 120
}

const scrollToBottom = (smooth = true, force = false) => {
  if (!force && !isNearBottom()) return

  nextTick(() => {
    if (!messageListRef.value) return
    const maxScrollTop = messageListRef.value.scrollHeight - messageListRef.value.clientHeight
    messageListRef.value.scrollTo({
      top: maxScrollTop,
      behavior: smooth ? 'smooth' : 'auto',
    })
  })
}

const handleMessageInputFocus = () => {
  scrollToBottom(true, true)
}

const sortMessages = (items: ChatMessage[]) =>
  items.sort((a, b) => {
    const timeDiff =
      new Date(a.created_at ?? 0).getTime() - new Date(b.created_at ?? 0).getTime()
    if (timeDiff !== 0) return timeDiff
    return String(a.id ?? '').localeCompare(String(b.id ?? ''))
  })

const createClientMessageId = () => {
  const random =
    typeof crypto !== 'undefined' && 'randomUUID' in crypto
      ? crypto.randomUUID()
      : `${Date.now()}-${Math.random().toString(36).slice(2)}`
  return `cmid_${currentUserId.value ?? 'unknown'}_${random}`
}

const mergeMessages = (incoming: ChatMessage[]) => {
  if (incoming.length === 0) return
  mergeUsers(incoming.map((message) => message.sender_profile))

  const next = [...messages.value]

  for (const msg of incoming) {
    const existingIndex = next.findIndex((item) => {
      if (msg.client_message_id && item.client_message_id === msg.client_message_id) return true
      if (msg.id && item.id === msg.id) return true
      return false
    })

    const existing = existingIndex === -1 ? undefined : next[existingIndex]
    const merged = {
      ...existing,
      ...msg,
      status: msg.sender_id === currentUserId.value ? 'sent' : msg.status,
      uploadState: existing?.uploadState && msg.id && !msg.id.startsWith('temp-')
        ? undefined
        : existing?.uploadState,
    }

    if (existingIndex === -1) {
      next.push(merged)
    } else {
      next[existingIndex] = merged
    }
  }

  const seen = new Set<string>()
  const deduped = next.filter((msg) => {
    const key = msg.client_message_id
      ? `client:${msg.client_message_id}`
      : msg.id
        ? `id:${msg.id}`
        : ''
    if (!key) return true
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })

  messages.value = sortMessages(deduped)
}

const syncConversationByMessage = (message: ChatMessage, shouldScroll = false) => {
  updateConversationByMessage(
    message as Parameters<typeof updateConversationByMessage>[0],
    activeChatId.value,
  )
  if (shouldScroll) {
    requestScrollToConversation(activeChatId.value)
  }
}

const handleNewMessage = async (message: ChatMessage) => {
  if (!isCurrentChatMessage(message)) return

  mergeMessages([message])
  syncConversationByMessage(message, message.sender_id === currentUserId.value)
  scrollToBottom(true, message.sender_id === currentUserId.value)

  if (!isGroupConversation.value && message.sender_id === peerUserId.value) {
    await syncUnreadState()
  }
}

const sendMessage = async () => {
  if (!messageText.value.trim() || !currentUserId.value) return
  if (!isGroupConversation.value && !peerUserId.value) {
    showToast('无效的会话')
    return
  }

  const content = messageText.value.trim()
  const clientMessageId = createClientMessageId()
  messageText.value = ''
  messageDrafts.delete(routeChatId.value)

  const tempMessage: ChatMessage = {
    id: `temp-${clientMessageId}`,
    client_message_id: clientMessageId,
    conversation_id: currentConversationId.value,
    chat_id: activeChatId.value,
    sender_id: currentUserId.value,
    type: MessageType.Text,
    content: { text: content },
    content_preview: content,
    status: 'sending',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  }

  messages.value = [...messages.value, tempMessage]
  scrollToBottom(true, true)

  try {
    await imStore.imSDK?.sendMessage(
      activeChatId.value,
      MessageType.Text,
      { text: content },
      clientMessageId,
    )
  } catch (error) {
    console.error('发送消息失败:', error)
    const messageIndex = messages.value.findIndex((msg) => msg.id === tempMessage.id)
    if (messageIndex !== -1 && messages.value[messageIndex]) {
      messages.value[messageIndex].status = 'failed'
      messages.value = [...messages.value]
    }
    showToast('发送失败')
  }
}

const handleScroll = (e: Event) => {
  const target = e.target as HTMLElement
  if (!target || !initialized.value || firstLoad.value || loading.value || !hasMore.value) return
  if (target.scrollTop <= 50) {
    fetchHistoryMessages(true)
  }
}

const fetchHistoryMessages = async (loadMore = false) => {
  if (loading.value) return
  if (!loadMore && !firstLoad.value && !hasMore.value) return

  try {
    loading.value = true
    if (!imStore.imSDK) return

    const oldScrollTop = messageListRef.value?.scrollTop || 0
    const oldScrollHeight = messageListRef.value?.scrollHeight || 0

    const oldestMessage = loadMore && messages.value.length > 0 ? messages.value[0] : undefined
    const beforeId = oldestMessage?.id && !oldestMessage.id.startsWith('temp-')
      ? oldestMessage.id
      : undefined

    const response = await imStore.imSDK.getConversationMessages(activeChatId.value, {
      limit: pageSize,
      before_id: beforeId,
    })

    const newMessages = response.sort(
      (a, b) => new Date(a.created_at ?? 0).getTime() - new Date(b.created_at ?? 0).getTime(),
    )
    mergeUsers(newMessages.map((message) => message.sender_profile))

    if (loadMore) {
      mergeMessages(newMessages)
      nextTick(() => {
        if (!messageListRef.value) return
        const scrollDiff = messageListRef.value.scrollHeight - oldScrollHeight
        messageListRef.value.scrollTop = oldScrollTop + scrollDiff
      })
    } else {
      messages.value = sortMessages(newMessages)
      scrollToBottom(false)
    }

    hasMore.value = newMessages.length === pageSize

    if (!loadMore) {
      await syncUnreadState()
    }
  } catch (error) {
    console.error('获取历史消息失败:', error)
    showToast('加载消息失败')
  } finally {
    loading.value = false
    if (!loadMore) {
      firstLoad.value = false
      initialized.value = true
    }
  }
}

const syncLatestMessages = async () => {
  if (!initialized.value || !imStore.imSDK || loading.value) return

  const wasNearBottom = isNearBottom()

  try {
    const knownMessageIds = new Set<string>()
    for (const msg of messages.value) {
      if (msg.id && !msg.id.startsWith('temp-')) {
        knownMessageIds.add(msg.id)
      }
    }
    const response = await imStore.imSDK.getConversationMessages(activeChatId.value, {
      limit: pageSize,
    })
    const incoming = response
      .filter(isCurrentChatMessage)
      .filter((msg) => !msg.id || !knownMessageIds.has(msg.id))

    if (incoming.length === 0) {
      await syncUnreadState()
      return
    }

    mergeMessages(incoming)

    await syncUnreadState()
    scrollToBottom(true, wasNearBottom)
  } catch (error) {
    console.error('同步最新消息失败:', error)
  }
}

const syncUnreadState = async () => {
  const maxSequence = getVisibleMaxSequence()
  const localReadSeq = conversation.value?.last_read_sequence ?? 0
  if (currentConversationId.value && maxSequence > 0) {
    readReporter.ack(currentConversationId.value, localReadSeq)
  }
  if (currentConversationId.value && maxSequence > localReadSeq) {
    readReporter.schedule(currentConversationId.value, maxSequence)
  }
  if (!isGroupConversation.value && peerUserId.value) {
    clearUnreadForPeer(peerUserId.value)
  }
  clearUnreadForConversation(currentConversationId.value)
}

const getVisibleMaxSequence = () => messages.value.reduce((max, message) => {
  const sequence = message.seq ?? 0
  return Number.isFinite(sequence) && sequence > max ? sequence : max
}, 0)

const flushReadState = async (targetConversationId = currentConversationId.value) => {
  if (!targetConversationId) return
  try {
    await readReporter.flush(targetConversationId)
  } catch (error) {
    console.error('标记会话已读失败:', error)
  }
}

const fetchTargetUser = async () => {
  if (conversation.value && isGroupConversation.value) {
    mergeConversationUsers(conversation.value)
    targetUser.value = null
    return
  }

  const peerInfo = conversation.value?.peer_user_info
  if (peerInfo) {
    mergeUsers([peerInfo])
    targetUser.value = peerInfo
    return
  }

  if (!peerUserId.value) {
    targetUser.value = null
    return
  }

  const fromConversation = conversations.value.find(
    (item) => item.chat_id === routeChatId.value,
  )
  const fromConversationPeer = fromConversation?.peer_user_info
  if (fromConversationPeer) {
    mergeUsers([fromConversationPeer])
    targetUser.value = fromConversationPeer
    return
  }

  const cached = userMap.value[peerUserId.value]
  if (cached) {
    targetUser.value = cached
    return
  }

  targetUser.value = await fetchUser(peerUserId.value)
}

const fetchConversation = async () => {
  const existing = conversations.value.find((item) => item.chat_id === routeChatId.value)
  if (existing) {
    conversation.value = existing
    mergeConversationUsers(existing)
    const peerInfo = existing.peer_user_info
    if (!isGroupConversationModel(existing) && peerInfo) {
      mergeUsers([peerInfo])
      targetUser.value = peerInfo
    }
  }

  try {
    const ensuredConversation = await ensureConversationByChatId(routeChatId.value)
    if (ensuredConversation) {
      conversation.value = ensuredConversation
    }
  } catch {
    // 路由里的 chat_id 只打开已有会话，不隐式创建新会话。
    await router.replace({ name: 'im-chat-index' })
    return
  }
  requestScrollToConversation(routeChatId.value)
  if (!conversation.value) return

  mergeConversationUsers(conversation.value)
  const peerInfo = conversation.value.peer_user_info
  if (!isGroupConversation.value && peerInfo) {
    mergeUsers([peerInfo])
    targetUser.value = peerInfo
  }
}

const retryMessage = async (message: ChatMessage) => {
  const messageIndex = messages.value.findIndex((msg) => msg.id === message.id)
  if (messageIndex === -1) return

  const current = messages.value[messageIndex]
  if (!current) return

  if (current.type === MessageType.Image && current.uploadState?.localFile) {
    await retryUploadImageMessage(current, messageIndex)
    return
  }

  const clientMessageId = current.client_message_id ?? createClientMessageId()
  current.status = 'sending'
  current.client_message_id = clientMessageId
  messages.value = [...messages.value]

  try {
    await imStore.imSDK?.sendMessage(
      activeChatId.value,
      message.type ?? MessageType.Text,
      message.content,
      clientMessageId,
    )
  } catch (error) {
    console.error('重新发送消息失败:', error)
    if (messages.value[messageIndex]) {
      messages.value[messageIndex].status = 'failed'
      messages.value = [...messages.value]
    }
    showToast('发送失败')
  }
}

const retryUploadImageMessage = async (message: ChatMessage, messageIndex: number) => {
  const file = message.uploadState?.localFile
  if (!file) {
    showToast('图片文件已失效，请重新选择')
    return
  }

  const current = messages.value[messageIndex]
  if (!current) return

  const clientMessageId = current.client_message_id ?? createClientMessageId()
  current.status = 'sending'
  current.client_message_id = clientMessageId
  current.uploadState = { localFile: file, uploading: true }
  messages.value = [...messages.value]

  try {
    const uploaded = await uploadIMFile(file)
    const dimensions = await readImageDimensions(file)
    const format = uploaded.format ?? getFileFormat(file)
    const w = uploaded.width ?? dimensions.width
    const h = uploaded.height ?? dimensions.height

    await imStore.imSDK?.sendMessage(
      activeChatId.value,
      MessageType.Image,
      {
        url: uploaded.url,
        width: w,
        height: h,
        size: uploaded.size,
        format,
        file_name: uploaded.filename || file.name,
      },
      clientMessageId,
    )
    const previewUrl = current.content.url
    if (previewUrl?.startsWith('blob:')) {
      URL.revokeObjectURL(previewUrl)
    }
  } catch (error) {
    console.error('重新上传图片失败:', error)
    const latest = messages.value.find((msg) => msg.id === current.id)
    if (latest) {
      latest.status = 'failed'
      latest.uploadState = { localFile: file, uploading: false, uploadFailed: true }
      messages.value = [...messages.value]
    }
    showToast('上传失败，点击重试')
  }
}

// 文件上传预览信息（纯前端本地结构）
interface FilePreview {
  url: string        // 预览 blob URL
  filename?: string
  size: number
  width?: number
  height?: number
  format?: string
  uploading?: boolean
}

const handleSelectFile = (file: File, type: string, preview: FilePreview) => {
  if (!currentUserId.value || (!isGroupConversation.value && !peerUserId.value)) return

  const messageType = type as MessageType
  const clientMessageId = createClientMessageId()
  const tempMessageId = `temp-${clientMessageId}`
  const tempMessage: ChatMessage = {
    id: tempMessageId,
    client_message_id: clientMessageId,
    conversation_id: currentConversationId.value,
    chat_id: activeChatId.value,
    sender_id: currentUserId.value,
    type: messageType,
    content_preview: messageType === MessageType.Image ? '[图片]' : '[视频]',
    status: 'sending',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    content: {
      url: preview.url,
      width: preview.width,
      height: preview.height,
      size: preview.size,
      format: preview.format,
      file_name: preview.filename || file.name,
    },
    uploadState: { uploading: true, localFile: file },
  }

  pendingUploadMessageIds.set(file, tempMessageId)
  messages.value = [...messages.value, tempMessage]
  scrollToBottom(true, true)
}

const handleUploadSuccess = async (file: File, type: string, uploaded: { url: string; filename?: string; size: number; width: number; height: number; format?: string }) => {
  const messageType = type as MessageType
  const pendingMessageId = pendingUploadMessageIds.get(file)
  const messageIndex = messages.value.findIndex(
    (msg) =>
      msg.status === 'sending' &&
      msg.type === messageType &&
      msg.uploadState?.uploading &&
      (!pendingMessageId || msg.id === pendingMessageId),
  )
  if (messageIndex === -1) return

  const current = messages.value[messageIndex]
  if (!current) return

  const previewUrl = current.content.url

  try {
    await imStore.imSDK?.sendMessage(
      activeChatId.value,
      messageType,
      {
        url: uploaded.url,
        width: uploaded.width,
        height: uploaded.height,
        size: uploaded.size,
        format: uploaded.format,
        file_name: uploaded.filename || file.name,
      } as MessageContent,
      current.client_message_id,
    )
    pendingUploadMessageIds.delete(file)
    if (previewUrl?.startsWith('blob:')) {
      URL.revokeObjectURL(previewUrl)
    }
  } catch (error) {
    console.error('发送媒体消息失败:', error)
    if (messages.value[messageIndex]) {
      messages.value[messageIndex].status = 'failed'
      messages.value[messageIndex].uploadState = { localFile: file, uploading: false, uploadFailed: true }
      messages.value = [...messages.value]
    }
    pendingUploadMessageIds.delete(file)
    showToast('发送失败')
  }
}

const handleUploadError = (file: File, type: string) => {
  const messageType = type as MessageType
  const pendingMessageId = pendingUploadMessageIds.get(file)
  const messageIndex = messages.value.findIndex(
    (msg) =>
      msg.status === 'sending' &&
      msg.type === messageType &&
      msg.uploadState?.uploading &&
      (!pendingMessageId || msg.id === pendingMessageId),
  )
  if (messageIndex === -1) return

  const current = messages.value[messageIndex]
  if (!current) return

  current.status = 'failed'
  current.uploadState = { localFile: file, uploading: false, uploadFailed: true }
  messages.value = [...messages.value]
  pendingUploadMessageIds.delete(file)
}

const waitForConnection = (timeoutMs = 10000) =>
  new Promise<void>((resolve, reject) => {
    if (imStore.isConnected) {
      resolve()
      return
    }

    const timer = window.setTimeout(() => {
      stop()
      reject(new Error('WebSocket connection timeout'))
    }, timeoutMs)

    const stop = watch(
      () => imStore.isConnected,
      (connected) => {
        if (connected) {
          window.clearTimeout(timer)
          stop()
          resolve()
        }
      },
      { immediate: true },
    )
  })

const resetChatState = () => {
  initialized.value = false
  firstLoad.value = true
  hasMore.value = true
  messages.value = []
  targetUser.value = null
  conversation.value = null
}

const restoreMessageDraft = (id: string) => {
  messageText.value = id ? messageDrafts.get(id) ?? '' : ''
}

const initChat = async () => {
  if (!userStore.token) {
    router.replace({ name: 'im-login', query: { redirect: router.currentRoute.value.fullPath } })
    return
  }

  if (imTabStore.isSuspended) {
    showConversationInfoDrawer.value = false
    resetChatState()
    imStore.closeConnection()
    return
  }

  if (!routeChatId.value) {
    showConversationInfoDrawer.value = false
    resetChatState()
    restoreMessageDraft('')
    imStore.initSDK()
    imStore.addMessageHandler(handleNewMessage)
    return
  }

  resetChatState()
  restoreMessageDraft(routeChatId.value)
  imStore.initSDK()
  imStore.addMessageHandler(handleNewMessage)

  try {
    await fetchConversation()
    if (!isGroupConversation.value && !peerUserId.value) {
      throw new Error('invalid conversation participants')
    }
    await Promise.all([fetchTargetUser(), waitForConnection()])
    await fetchHistoryMessages()
  } catch (error) {
    console.error('初始化聊天失败:', error)
    showToast('连接失败，请稍后重试')
  }
}

onMounted(() => {
  const mediaQuery = window.matchMedia('(max-width: 767px)')
  const handleViewportChange = (event: MediaQueryListEvent) => {
    isMobileViewport.value = event.matches
  }
  isMobileViewport.value = mediaQuery.matches
  mediaQuery.addEventListener('change', handleViewportChange)
  document.addEventListener('pointerdown', handleDocumentPointerDown)
  document.addEventListener('visibilitychange', handleVisibilityChange)
  cleanupViewportListener = () => {
    mediaQuery.removeEventListener('change', handleViewportChange)
    document.removeEventListener('pointerdown', handleDocumentPointerDown)
    document.removeEventListener('visibilitychange', handleVisibilityChange)
  }

  initChat()
})

watch(
  () => imTabStore.isPrimaryTab,
  (isPrimary, wasPrimary) => {
    if (!imTabStore.initialized || isPrimary === wasPrimary) return

    if (!isPrimary) {
      resetChatState()
      imStore.closeConnection()
      return
    }

    initChat()
  },
)

watch(
  () => props.chatId,
  (nextChatId, prevChatId) => {
    if (nextChatId !== prevChatId) {
      if (currentConversationId.value) {
        void flushReadState(currentConversationId.value)
      }
      if (prevChatId && messageText.value) {
        messageDrafts.set(prevChatId, messageText.value)
      }
      showConversationInfoDrawer.value = false
      imStore.removeMessageHandler(handleNewMessage)
      initChat()
    }
  },
)

const handleVisibilityChange = () => {
  if (document.visibilityState === 'hidden') {
    void flushReadState()
    return
  }
  void syncUnreadState()
}

watch(
  () => imStore.isConnected,
  (connected, wasConnected) => {
    if (hasSelectedConversation.value && connected && wasConnected === false) {
      syncLatestMessages()
    }
  },
)

onUnmounted(() => {
  void readReporter.flushAll()
  cleanupViewportListener?.()
  imStore.removeMessageHandler(handleNewMessage)
})
</script>

<style scoped>
.chat-page {
  height: 100dvh;
  background: white;
}

.chat-layout {
  display: flex;
  height: 100%;
  min-height: 0;
}

.chat-sidebar {
  width: 300px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg-color);
  border-right: 1px solid var(--border-color-light);
  min-height: 0;
}

.sidebar-header {
  flex-shrink: 0;
  padding: 14px var(--spacing-base);
  border-bottom: 1px solid var(--border-color-light);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-small);
}

.sidebar-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color-dark);
}

.sidebar-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.sidebar-icon-btn,
.sidebar-menu-btn {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text-color-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.sidebar-icon-btn i,
.sidebar-menu-btn i {
  font-size: 20px;
  line-height: 1;
}

.sidebar-icon-btn:hover,
.sidebar-menu-btn:hover,
.sidebar-menu-btn.is-active {
  background: var(--bg-color-gray);
  color: var(--text-color-dark);
}

.new-conversation-modal {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 14vh 16px 24px;
  background: rgba(15, 23, 42, 0.32);
}

.new-conversation-panel {
  width: min(420px, 100%);
  overflow: hidden;
  border-radius: 12px;
  background: var(--bg-color);
  box-shadow: 0 22px 60px rgba(15, 23, 42, 0.18);
}

.new-conversation-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 18px;
  border-bottom: 1px solid var(--border-color-light);
}

.new-conversation-header h2 {
  margin: 0;
  color: var(--text-color-dark);
  font-size: 16px;
  font-weight: 600;
  line-height: 1.4;
}

.new-conversation-tabs {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
  margin: 14px 18px 0;
  padding: 4px;
  border-radius: 8px;
  background: #f3f5f9;
}

.new-conversation-tabs button {
  height: 32px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--text-color-secondary);
  font-size: 14px;
  cursor: pointer;
}

.new-conversation-tabs button.is-active {
  background: white;
  color: var(--text-color-dark);
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08);
}

.new-conversation-close {
  width: 32px;
  height: 32px;
  padding: 0;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text-color-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.new-conversation-close i {
  font-size: 20px;
  line-height: 1;
}

.new-conversation-close:hover {
  background: var(--bg-color-gray);
  color: var(--text-color-dark);
}

.new-conversation-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 18px 18px 0;
}

.new-conversation-field span {
  color: var(--text-color-secondary);
  font-size: 13px;
  line-height: 1.4;
}

.new-conversation-field input,
.new-conversation-field textarea {
  width: 100%;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  outline: none;
  color: var(--text-color-dark);
  background: var(--bg-color);
  font-size: 14px;
  line-height: 1.4;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.new-conversation-field input {
  height: 40px;
  padding: 0 12px;
}

.new-conversation-field textarea {
  min-height: 92px;
  resize: vertical;
  padding: 10px 12px;
}

.new-conversation-field input:focus,
.new-conversation-field textarea:focus {
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.12);
}

.new-conversation-error {
  margin: 10px 18px 0;
  color: var(--danger-color);
  font-size: 13px;
  line-height: 1.4;
}

.new-conversation-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 18px;
}

.new-conversation-cancel,
.new-conversation-submit {
  min-width: 72px;
  height: 36px;
  padding: 0 14px;
  border-radius: 8px;
  font-size: 14px;
  line-height: 1.4;
  cursor: pointer;
}

.new-conversation-cancel {
  border: 1px solid var(--border-color);
  background: var(--bg-color);
  color: var(--text-color-secondary);
}

.new-conversation-submit {
  border: none;
  background: var(--primary-color);
  color: #fff;
}

.new-conversation-submit:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.sidebar-footer {
  position: relative;
  flex-shrink: 0;
  padding: 10px var(--spacing-base);
  border-top: 1px solid var(--border-color-light);
}

.sidebar-footer-trigger {
  width: 100%;
  border: none;
  padding: 0;
  background: transparent;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  cursor: pointer;
  border-radius: 10px;
  transition: background 0.15s ease;
}

.sidebar-footer-trigger:hover,
.sidebar-footer-trigger.is-active {
  background: transparent;
}

.sidebar-footer-user {
  min-width: 0;
  flex: 1;
  display: flex;
  align-items: center;
  gap: 10px;
}

.sidebar-footer-avatar {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  border-radius: 50%;
  object-fit: cover;
  background: #eef1f6;
}

.sidebar-footer-avatar-fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-color-secondary);
}

.sidebar-footer-avatar-fallback i {
  font-size: 16px;
  line-height: 1;
}

.sidebar-footer-name {
  min-width: 0;
  color: var(--text-color-dark);
  font-size: 14px;
  font-weight: 500;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-menu {
  position: absolute;
  left: var(--spacing-base);
  bottom: calc(100% + 6px);
  width: 270px;
  padding: 6px;
  border: 1px solid var(--border-color-light);
  border-radius: 8px;
  background: white;
  box-shadow: 0 10px 28px rgba(15, 23, 42, 0.12);
  z-index: 20;
}

.sidebar-user {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  padding: 7px 8px 9px;
  margin-bottom: 4px;
  border-bottom: 1px solid var(--border-color-light);
}

.sidebar-user-avatar {
  width: 28px;
  height: 28px;
  flex-shrink: 0;
  border-radius: 50%;
  background: #eef1f6;
  object-fit: cover;
}

.sidebar-user-name {
  min-width: 0;
  color: var(--text-color-dark);
  font-size: 13px;
  font-weight: 500;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-menu-item {
  width: 100%;
  border: none;
  background: transparent;
  border-radius: 6px;
  padding: 8px 10px;
  color: var(--text-color-dark);
  font-size: 14px;
  line-height: 1.4;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  text-align: left;
}

.sidebar-menu-item i {
  flex-shrink: 0;
  color: var(--text-color-secondary);
  font-size: 16px;
  line-height: 1;
}

.sidebar-menu-item:hover {
  background: var(--bg-color-gray);
}

.chat-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: white;
}

.chat-empty-main {
  align-items: center;
  justify-content: center;
  background: #fafbfe;
}

.chat-empty-state {
  width: 88px;
  height: 88px;
  border-radius: 50%;
  color: #8a96aa;
  display: flex;
  align-items: center;
  justify-content: center;
}

.chat-empty-state i {
  font-size: 38px;
  line-height: 1;
}

.chat-suspended-main {
  align-items: center;
  justify-content: center;
  background: #fafbfe;
}

.chat-suspended-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  color: #687386;
}

.chat-suspended-state i {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  background: #eef3fb;
  color: #8a96aa;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 32px;
  line-height: 1;
}

.chat-suspended-state p {
  margin: 0;
  font-size: 14px;
}

.chat-suspended-state button {
  border: none;
  border-radius: 8px;
  background: #4b86f8;
  color: white;
  padding: 8px 14px;
  font-size: 14px;
  cursor: pointer;
}

@media (max-width: 767px) {
  .chat-sidebar {
    display: none;
  }
}

.nav-bar {
  position: sticky;
  top: 0;
  background: white;
  z-index: 100;
  border-bottom: 1px solid var(--border-color-light);
}

.nav-bar-content {
  height: 50px;
  margin: 0 auto;
  width: 100%;
  display: flex;
  align-items: center;
  padding: 0 var(--spacing-base);
}

.nav-side-btn {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  border: none;
  background: none;
  padding: 0;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-color-dark);
}

.nav-side-btn i {
  font-size: 22px;
  line-height: 1;
}

.back-btn {
  justify-content: flex-start;
  width: auto;
  padding-right: 4px;
}

.nav-bar-center {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: flex-start;
  min-width: 0;
}

.nav-reconnect-icon {
  font-size: 16px;
  color: var(--text-color-dark);
  animation: nav-reconnect-spin 0.8s linear infinite;
}

@keyframes nav-reconnect-spin {
  to {
    transform: rotate(360deg);
  }
}

.user-info {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 0;
  max-width: 100%;
}

.title {
  font-size: var(--font-size-small);
  font-weight: 600;
  margin: 0;
  line-height: 1.2;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.message-container {
  flex: 1;
  overflow: hidden;
  position: relative;
}

.message-list {
  height: 100%;
  overflow-y: auto;
  padding: var(--spacing-small);
  -webkit-overflow-scrolling: touch;
}

.message-item {
  display: flex;
  align-items: flex-start;
  margin-bottom: var(--spacing-large);
}

.message-time-divider {
  width: fit-content;
  max-width: calc(100% - 48px);
  margin: 12px auto 16px;
  padding: 3px 8px;
  border-radius: 10px;
  background: transparent;
  color: #8a93a3;
  font-size: 12px;
  line-height: 1.4;
  text-align: center;
}

.message-mine {
  flex-direction: row-reverse;
}

.message-avatar {
  width: 32px;
  height: 32px;
  margin: 0 var(--spacing-small);
  flex-shrink: 0;
}

.message-avatar img {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  object-fit: cover;
}

.message-wrapper {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  flex-grow: 1;
  min-width: 0;
}

.message-sender-name {
  margin: 0 0 4px 2px;
  color: #8a93a3;
  font-size: 12px;
  line-height: 1.3;
}

.message-mine .message-wrapper {
  align-items: flex-end;
}

.message-input-container {
  position: sticky;
  bottom: 0;
  left: 0;
  right: 0;
  background: white;
  border-top: 1px solid var(--border-color-light);
  padding-bottom: env(safe-area-inset-bottom);
  z-index: 10;
}

.message-input {
  padding: 10px var(--spacing-base);
  display: flex;
}

.message-input-content {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  flex: 1;
  min-width: 0;
  border: 1px solid var(--border-color-light);
  background: #f6f8fc;
  border-radius: 18px;
  min-height: 64px;
  overflow: visible;
  padding: 0 8px 0 16px;
}

@media (max-width: 767px) {
  .message-input {
    padding: 8px 10px;
  }

  .message-input-content {
    min-height: 42px;
    border-radius: 21px;
    padding-left: 14px;
  }
}

.message-input-actions {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 0;
}

.message-input .send-btn {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: #4b86f8;
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: opacity 0.2s ease;
  font-size: 18px;
  line-height: 1.4;
  font-weight: 500;
}

.message-input .send-btn:disabled {
  /* opacity: 0.38; */
  cursor: not-allowed;
}

/* .message-input .send-btn:not(:disabled):active {
  opacity: 0.85;
} */

.loading-spinner {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: var(--spacing-small);
  height: 40px;
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
}

.spinner {
  width: 20px;
  height: 20px;
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

.no-more-messages {
  text-align: center;
  color: var(--text-color-light);
  font-size: 12px;
  padding: var(--spacing-small);
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.system-notice-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px var(--spacing-base);
  background: #f0f4ff;
  border-top: 1px solid #dce3f5;
  color: #5b6c94;
  font-size: 13px;
}

.system-notice-bar i {
  font-size: 18px;
  line-height: 1;
  color: #8ba0cb;
}
</style>
