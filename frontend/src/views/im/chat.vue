<template>
  <div class="chat-page">
    <div class="chat-layout">
      <!-- 侧边栏：会话列表 -->
      <aside class="chat-sidebar">
        <div class="sidebar-header">
          <span class="sidebar-title">会话</span>
        </div>
        <div class="sidebar-list">
          <button
            v-for="conv in conversationList"
            :key="conv.chat_id"
            class="conv-item"
            :class="{ active: selectedChatId === conv.chat_id }"
            @click="selectConversation(conv.chat_id)"
          >
            <span class="conv-name">{{ conv.chat_id }}</span>
            <span v-if="conv.unread_count" class="conv-badge">{{ conv.unread_count }}</span>
          </button>
          <p v-if="!conversationList.length" class="empty-hint">暂无会话</p>
        </div>
        <div class="sidebar-footer">
          <button class="logout-btn" @click="handleLogout">退出</button>
        </div>
      </aside>

      <!-- 聊天区域 -->
      <main class="chat-main" v-if="selectedChatId">
        <div class="chat-header">
          <span class="chat-title">{{ selectedChatId }}</span>
          <span class="connection-status" :class="{ connected: wsConnected }">
            {{ wsConnected ? '已连接' : '未连接' }}
          </span>
        </div>

        <div class="message-list" ref="messageListRef">
          <div v-for="msg in messages" :key="msg.msg_id" class="msg-item" :class="{ mine: msg.from_uid === myUid }">
            <div class="msg-bubble" :class="{ mine: msg.from_uid === myUid }">
              <template v-if="msg.msg_type === 'text'">
                <p class="msg-text">{{ (msg.content as any).text }}</p>
              </template>
              <template v-else-if="msg.msg_type === 'image'">
                <img :src="(msg.content as any).url" class="msg-image" alt="" />
              </template>
              <template v-else>
                <p class="msg-text">[{{ msg.msg_type }}]</p>
              </template>
              <span class="msg-time">{{ formatTime(msg.server_time) }}</span>
            </div>
          </div>
        </div>

        <div class="chat-input">
          <input
            v-model="inputText"
            class="input"
            placeholder="输入消息…"
            @keyup.enter="sendTextMessage"
            :disabled="!wsConnected"
          />
          <button class="send-btn" @click="sendTextMessage" :disabled="!wsConnected">发送</button>
        </div>
      </main>

      <!-- 未选中会话 -->
      <div class="chat-empty" v-else>
        <p>选择一个会话开始聊天</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useIMStore } from '@/stores/im'
import { useUserStore } from '@/stores/user'
import type { Message, Conversation } from '@/sdk/im'
import { ChatType } from '@/sdk/im'

const router = useRouter()
const userStore = useUserStore()
const imStore = useIMStore()

const inputText = ref('')
const messages = ref<Message[]>([])
const selectedChatId = ref('')
const conversationList = ref<Conversation[]>([])
const wsConnected = ref(false)
const messageListRef = ref<HTMLElement | null>(null)

const myUid = computed(() => {
  // 从 JWT sub 解析 uid（简化：直接读 token payload）
  try {
    const t = userStore.token
    if (!t) return ''
    const payload = JSON.parse(atob(t.split('.')[1] || ''))
    return payload.sub || ''
  } catch {
    return ''
  }
})

// ---- 会话列表 ----
async function loadConversations() {
  try {
    const sdk = imStore.imSDK
    if (!sdk) return
    conversationList.value = await sdk.getConversations()
  } catch (e) {
    console.warn('加载会话列表失败', e)
  }
}

function selectConversation(chatId: string) {
  selectedChatId.value = chatId
  messages.value = []
}

// ---- 发送消息 ----
async function sendTextMessage() {
  const text = inputText.value.trim()
  if (!text || !selectedChatId.value) return

  const sdk = imStore.imSDK
  if (!sdk) return

  const parts = (selectedChatId.value || '').split('_')
  const targetUIDs: string[] = parts.filter(p => p !== 'single' && p !== myUid.value) as string[]
  if (!targetUIDs.length) targetUIDs.push('demo_target')

  try {
    await sdk.sendTextMessage(
      selectedChatId.value,
      ChatType.Single,
      text,
      targetUIDs,
      myUid.value || undefined,
    )
    inputText.value = ''
  } catch (e: any) {
    alert('发送失败: ' + (e?.message || ''))
  }
}

// ---- 接收消息 ----
function onNewMessage(msg: Message) {
  if (msg.chat_id === selectedChatId.value) {
    messages.value = [...messages.value, msg]
    nextTick(() => scrollToBottom())
  }
}

function onConnection(status: import('@/sdk/im').ConnectionStatus) {
  wsConnected.value = status.status === 'connected'
}

// ---- 初始化 ----
onMounted(async () => {
  imStore.addMessageHandler(onNewMessage)
  imStore.initSDK()
  await loadConversations()
})

onUnmounted(() => {
  imStore.removeMessageHandler(onNewMessage)
})

function scrollToBottom() {
  const el = messageListRef.value
  if (el) el.scrollTop = el.scrollHeight
}

function formatTime(iso: string) {
  if (!iso) return ''
  return new Date(iso).toLocaleTimeString()
}

async function handleLogout() {
  await userStore.logout()
  router.replace({ name: 'im-login' })
}
</script>

<style scoped>
.chat-page { height: 100dvh; display: flex; }
.chat-layout { display: flex; width: 100%; height: 100%; }

.chat-sidebar {
  width: 280px; background: #f0f2f5; border-right: 1px solid #e0e0e0;
  display: flex; flex-direction: column;
}
.sidebar-header { padding: 16px; border-bottom: 1px solid #e0e0e0; }
.sidebar-title { font-size: 18px; font-weight: 600; }
.sidebar-list { flex: 1; overflow-y: auto; padding: 8px; }
.conv-item {
  display: flex; justify-content: space-between; align-items: center;
  width: 100%; padding: 12px; border-radius: 8px; border: none;
  background: transparent; cursor: pointer; text-align: left; margin-bottom: 4px;
}
.conv-item:hover { background: #e4e6eb; }
.conv-item.active { background: #d0e2ff; }
.conv-name { font-size: 14px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.conv-badge {
  background: #e84040; color: #fff; border-radius: 10px;
  font-size: 12px; padding: 2px 8px; min-width: 20px; text-align: center;
}
.empty-hint { color: #999; text-align: center; margin-top: 40px; font-size: 14px; }
.sidebar-footer { padding: 12px; border-top: 1px solid #e0e0e0; }
.logout-btn { width: 100%; padding: 10px; border: none; background: #f5f5f5; border-radius: 6px; cursor: pointer; }
.logout-btn:hover { background: #e8e8e8; }

.chat-main { flex: 1; display: flex; flex-direction: column; min-width: 0; }
.chat-header {
  padding: 12px 16px; border-bottom: 1px solid #e0e0e0;
  display: flex; justify-content: space-between; align-items: center;
}
.chat-title { font-size: 16px; font-weight: 600; }
.connection-status { font-size: 12px; color: #999; }
.connection-status.connected { color: #22c55e; }

.message-list { flex: 1; overflow-y: auto; padding: 16px; }
.msg-item { display: flex; margin-bottom: 12px; }
.msg-item.mine { justify-content: flex-end; }
.msg-bubble { max-width: 70%; padding: 8px 14px; border-radius: 12px; background: #f0f0f0; position: relative; }
.msg-bubble.mine { background: #4b86f8; color: #fff; }
.msg-text { margin: 0; font-size: 14px; line-height: 1.5; }
.msg-image { max-width: 200px; border-radius: 8px; }
.msg-time { font-size: 11px; color: #999; margin-top: 4px; display: block; }
.msg-bubble.mine .msg-time { color: rgba(255,255,255,0.7); }

.chat-input {
  padding: 12px 16px; border-top: 1px solid #e0e0e0; display: flex; gap: 8px;
}
.input { flex: 1; padding: 10px 14px; border: 1px solid #ddd; border-radius: 8px; outline: none; font-size: 14px; }
.input:focus { border-color: #4b86f8; }
.send-btn { padding: 10px 20px; background: #4b86f8; color: #fff; border: none; border-radius: 8px; cursor: pointer; font-size: 14px; }
.send-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.chat-empty { flex: 1; display: flex; align-items: center; justify-content: center; color: #999; font-size: 16px; }
</style>