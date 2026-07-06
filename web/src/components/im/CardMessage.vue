<!-- 卡片消息组件 -->
<template>
  <div class="message-content message-card">
    <div class="message-arrow"></div>
    <div @click="handleCardClick" class="card-link">
      <div v-if="message.content.image_url" class="card-thumb">
        <img 
          v-if="!imageError"
          :src="message.content.image_url" 
          alt=""
          @error="imageError = true"
        >
        <PlaceholderImage 
          v-else
          text=""
          bgColor="#EFF1F8"
          aspect="4:3"
        />
      </div>
      <div class="card-info">
        <div class="card-title">{{ message.content.title || '' }}</div>
        <div class="card-desc">{{ message.content.description }}</div>
        <div class="card-price" v-if="message.content.price_text">
          <span class="price-amount">{{ message.content.price_text }}</span>
        </div>
      </div>
    </div>
    <MessageStatus 
      v-if="isMine" 
      :status="message.status" 
      @retry="$emit('retry')" 
    />
    <ConfirmDialog
      :visible="showConfirm"
      title="打开链接"
      message="即将在浏览器中打开以下链接"
      :detail="linkToOpen"
      confirm-text="打开"
      cancel-text="取消"
      @confirm="openLink"
      @cancel="showConfirm = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { Message } from '@/types/im'
import MessageStatus from './MessageStatus.vue'
import PlaceholderImage from '../common/PlaceholderImage.vue'
import ConfirmDialog from '../common/ConfirmDialog.vue'

const props = defineProps<{
  message: Message
  isMine: boolean
}>()

defineEmits<{
  (e: 'retry'): void
}>()

const imageError = ref(false)
const showConfirm = ref(false)
const linkToOpen = ref('')

const handleCardClick = () => {
  const url = props.message.content.action_url
  if (!url) return
  linkToOpen.value = url
  showConfirm.value = true
}

const openLink = () => {
  showConfirm.value = false
  if (linkToOpen.value) {
    window.open(linkToOpen.value, '_blank', 'noopener,noreferrer')
  }
}
</script>

<style scoped>
.message-content {
  position: relative;
  border-radius: 12px;
  background: var(--bg-color);
  max-width: 320px;
  width: 80%;
  padding: 0;
}

.message-content.message-card {
  max-width: 320px;
  width: 80%;
  padding: 0;
  overflow: hidden;
  background: var(--bg-color);
}

.card-link {
  display: flex;
  flex-direction: column;
  text-decoration: none;
  border: 1px solid var(--border-color-light);
  border-radius: 12px;
  overflow: hidden;
  background: var(--bg-color);
  cursor: pointer;
}

.card-thumb {
  width: 100%;
  aspect-ratio: 4/3;
  background: #EFF1F8;
  overflow: hidden;
}

.card-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.card-info {
  padding: 10px 12px;
  flex: 1;
  min-width: 0;
}

.card-title {
  font-size: 15px;
  font-weight: 500;
  color: var(--text-color-dark);
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  line-height: 1.4;
}

.card-desc {
  font-size: 13px;
  color: var(--text-color-muted);
  line-height: 1.5;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.card-price {
  margin-top: 6px;
}

.price-amount {
  font-size: 14px;
  color: var(--price-color);
  font-weight: 600;
  font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

.message-arrow {
  position: absolute;
  top: 12px;
  left: -6px;
  width: 0;
  height: 0;
  border-style: solid;
  border-width: 6px 6px 6px 0;
  border-color: transparent var(--bg-color) transparent transparent;
}

.message-mine .message-content {
  background: var(--bg-color);
}

.message-mine .message-arrow {
  left: auto;
  right: -6px;
  border-width: 6px 0 6px 6px;
  border-color: transparent transparent transparent var(--primary-color);
}

</style> 
