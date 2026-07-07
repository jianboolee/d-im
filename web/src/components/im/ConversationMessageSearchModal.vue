<template>
  <Teleport to="body">
    <div v-if="modelValue" class="message-search-modal" @click.self="close">
      <div class="search-panel" role="dialog" aria-modal="true" aria-label="查找聊天内容">
        <div class="search-header">
          <div class="search-input-wrap">
            <i class="ri-search-line"></i>
            <input
              ref="searchInputRef"
              v-model="keyword"
              type="search"
              placeholder="搜索聊天内容"
              @keydown.enter="search"
              @keydown.esc="close"
            >
            <button
              v-if="keyword"
              class="search-clear-btn"
              type="button"
              aria-label="清空搜索"
              @click="clear"
            >
              <i class="ri-close-circle-fill"></i>
            </button>
          </div>
          <button class="search-cancel-btn" type="button" @click="close">
            取消
          </button>
        </div>

        <div class="search-body">
          <div v-if="searching && results.length === 0" class="search-state">
            <div class="spinner"></div>
          </div>
          <div v-else-if="results.length > 0" class="search-results">
            <div
              v-for="message in results"
              :key="message.id"
              class="search-result-item"
            >
              <div class="result-main">
                <span class="result-sender">{{ getSenderName(message) }}</span>
                <span class="result-text">{{ message.content_preview || message.content.text || message.content.title || '[消息]' }}</span>
              </div>
              <span class="result-time">{{ formatTime(message.created_at) }}</span>
            </div>
            <button
              v-if="hasMore"
              type="button"
              class="load-more-btn"
              :disabled="searching"
              @click="loadMore"
            >
              {{ searching ? '加载中...' : '加载更多' }}
            </button>
          </div>
          <div v-else class="empty-state">
            <i :class="hasSearched ? 'ri-search-eye-line' : 'ri-search-line'"></i>
            <p>{{ hasSearched ? '没有找到相关内容' : '输入关键词搜索当前会话' }}</p>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import dayjs from 'dayjs'
import { nextTick, ref, watch } from 'vue'
import { showToast } from '@/plugins/toast'
import { useIMStore } from '@/stores/im'
import { useUserProfiles } from '@/composables/useUserProfiles'
import type { Message } from '@/sdk/im'

const props = defineProps<{
  modelValue: boolean
  chatId: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const imStore = useIMStore()
const { userMap, mergeUsers } = useUserProfiles()
const searchInputRef = ref<HTMLInputElement | null>(null)
const keyword = ref('')
const results = ref<Message[]>([])
const nextCursor = ref<string | undefined>()
const hasMore = ref(false)
const searching = ref(false)
const hasSearched = ref(false)

const close = () => {
  emit('update:modelValue', false)
}

const clear = () => {
  keyword.value = ''
  results.value = []
  nextCursor.value = undefined
  hasMore.value = false
  hasSearched.value = false
}

const search = async () => {
  const q = keyword.value.trim()
  if (!q || !props.chatId || !imStore.imSDK) return
  try {
    searching.value = true
    hasSearched.value = true
    const page = await imStore.imSDK.searchChatMessages(props.chatId, {
      q,
      limit: 20,
    })
    results.value = page.items
    nextCursor.value = page.next_cursor
    hasMore.value = page.has_more
    mergeUsers(page.items.map((message) => message.sender_profile))
  } catch (error) {
    console.error('搜索聊天内容失败:', error)
    showToast('搜索失败')
  } finally {
    searching.value = false
  }
}

const loadMore = async () => {
  const q = keyword.value.trim()
  if (!q || !props.chatId || !imStore.imSDK || !hasMore.value || searching.value) return
  try {
    searching.value = true
    const page = await imStore.imSDK.searchChatMessages(props.chatId, {
      q,
      limit: 20,
      cursor: nextCursor.value,
    })
    results.value = [...results.value, ...page.items]
    nextCursor.value = page.next_cursor
    hasMore.value = page.has_more
    mergeUsers(page.items.map((message) => message.sender_profile))
  } catch (error) {
    console.error('加载更多聊天搜索结果失败:', error)
    showToast('加载失败')
  } finally {
    searching.value = false
  }
}

const getSenderName = (message: Message) => (
  message.sender_profile?.nickname
  || userMap.value[message.sender_id || '']?.nickname
  || message.sender_id
  || '未知用户'
)

const formatTime = (value?: string) => value ? dayjs(value).format('MM-DD HH:mm') : ''

watch(
  () => props.modelValue,
  (visible) => {
    if (!visible) {
      clear()
      return
    }

    nextTick(() => {
      searchInputRef.value?.focus()
    })
  },
)

watch(
  () => props.chatId,
  () => {
    clear()
  },
)
</script>

<style scoped>
.message-search-modal {
  position: fixed;
  inset: 0;
  z-index: 1250;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 8vh 16px 24px;
  background: rgba(15, 23, 42, 0.32);
}

.search-panel {
  width: min(520px, 100%);
  height: min(640px, 78dvh);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: 12px;
  background: var(--bg-color);
  box-shadow: 0 22px 60px rgba(15, 23, 42, 0.18);
}

.search-header {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px;
  border-bottom: 1px solid var(--border-color-light);
  background: white;
}

.search-input-wrap {
  min-width: 0;
  height: 38px;
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  border-radius: 8px;
  background: #f3f5f9;
  color: var(--text-color-secondary);
}

.search-input-wrap > i {
  flex-shrink: 0;
  font-size: 18px;
  line-height: 1;
}

.search-input-wrap input {
  min-width: 0;
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  color: var(--text-color-dark);
  font-size: 14px;
  line-height: 1.4;
}

.search-input-wrap input::-webkit-search-cancel-button {
  display: none;
}

.search-clear-btn,
.search-cancel-btn {
  border: none;
  background: transparent;
  color: var(--text-color-secondary);
  cursor: pointer;
}

.search-clear-btn {
  width: 22px;
  height: 22px;
  flex-shrink: 0;
  padding: 0;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.search-clear-btn i {
  font-size: 16px;
  line-height: 1;
}

.search-cancel-btn {
  flex-shrink: 0;
  font-size: 14px;
  line-height: 1.4;
}

.search-clear-btn:hover,
.search-cancel-btn:hover {
  color: var(--text-color-dark);
}

.search-body {
  min-height: 0;
  flex: 1;
  overflow-y: auto;
}

.search-state,
.empty-state {
  min-height: 260px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--text-color-light);
}

.empty-state i {
  font-size: 42px;
  line-height: 1;
}

.empty-state p {
  margin: 0;
  font-size: 14px;
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

.search-results {
  display: flex;
  flex-direction: column;
  gap: 1px;
  padding: 8px 0;
}

.search-result-item {
  width: 100%;
  min-height: 58px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 16px;
  border: none;
  background: #fff;
  text-align: left;
}

.search-result-item:hover {
  background: #f7f8fb;
}

.result-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.result-sender {
  color: var(--text-color-dark);
  font-size: 13px;
  font-weight: 600;
  line-height: 1.3;
}

.result-text {
  color: var(--text-color-secondary);
  font-size: 13px;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.result-time {
  flex-shrink: 0;
  color: var(--text-color-light);
  font-size: 12px;
}

.load-more-btn {
  width: calc(100% - 32px);
  margin: 10px 16px 14px;
  border: none;
  border-radius: 8px;
  background: #eef3ff;
  color: #4b70c8;
  padding: 10px;
  font-size: 13px;
  cursor: pointer;
}

.load-more-btn:disabled {
  cursor: not-allowed;
  opacity: 0.7;
}

@media (max-width: 767px) {
  .message-search-modal {
    padding: 0;
    background: white;
  }

  .search-panel {
    width: 100%;
    height: 100dvh;
    border-radius: 0;
    box-shadow: none;
  }

  .search-header {
    padding: calc(10px + env(safe-area-inset-top)) 10px 10px;
  }
}
</style>
