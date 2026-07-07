<template>
  <Teleport to="body">
    <div v-if="modelValue" class="conversation-search-modal" @click.self="close">
      <div class="search-panel" role="dialog" aria-modal="true" aria-label="搜索会话">
        <div class="search-header">
          <div class="search-input-wrap">
            <i class="ri-search-line"></i>
            <input
              ref="searchInputRef"
              v-model="keyword"
              type="search"
              placeholder="搜索联系人或用户 ID"
              @keydown.esc="close"
            >
            <button
              v-if="keyword"
              class="search-clear-btn"
              type="button"
              aria-label="清空搜索"
              @click="keyword = ''"
            >
              <i class="ri-close-circle-fill"></i>
            </button>
          </div>
          <button class="search-cancel-btn" type="button" @click="close">
            取消
          </button>
        </div>

        <ConversationList
          embedded
          search-mode
          :active-chat-id="activeChatId"
          :navigate-mode="navigateMode"
          :search-keyword="keyword"
          @select="handleSelect"
        />
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import ConversationList from '@/components/im/ConversationList.vue'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    activeChatId?: string
    navigateMode?: 'push' | 'replace' | 'none'
  }>(),
  {
    activeChatId: '',
    navigateMode: 'replace',
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  select: [chatId: string]
}>()

const keyword = ref('')
const searchInputRef = ref<HTMLInputElement | null>(null)

const close = () => {
  emit('update:modelValue', false)
}

const handleSelect = (chatId: string) => {
  emit('select', chatId)
  close()
}

watch(
  () => props.modelValue,
  (visible) => {
    if (!visible) {
      keyword.value = ''
      return
    }

    nextTick(() => {
      searchInputRef.value?.focus()
    })
  },
)
</script>

<style scoped>
.conversation-search-modal {
  position: fixed;
  inset: 0;
  z-index: 1000;
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

.search-clear-btn {
  width: 22px;
  height: 22px;
  flex-shrink: 0;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--text-color-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.search-clear-btn i {
  font-size: 16px;
  line-height: 1;
}

.search-cancel-btn {
  flex-shrink: 0;
  border: none;
  background: transparent;
  color: var(--text-color-secondary);
  font-size: 14px;
  line-height: 1.4;
  cursor: pointer;
}

.search-clear-btn:hover,
.search-cancel-btn:hover {
  color: var(--text-color-dark);
}

@media (max-width: 767px) {
  .conversation-search-modal {
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
