<!-- 图片消息组件 -->
<template>
  <div class="message-content message-image">
    <div class="message-arrow"></div>
    <div class="image-container" :style="containerStyle">
      <div class="image-wrapper">
        <ImageView
          :src="message.content.url || ''"
          :alt="message.content_preview || '图片'"
          placeholderText="图片"
          :width="containerStyle.width"
          :height="containerStyle.height"
          maxWidth="100%"
          fit="cover"
          @click="preview"
          @load="handleImageLoad"
        />
        
        <!-- 上传中状态 -->
        <div v-if="message.uploadState?.uploading" class="upload-overlay">
          <div class="upload-indicator">
            <div class="upload-spinner"></div>
            <span>上传中</span>
          </div>
        </div>
      </div>
    </div>
    
    <MessageStatus 
      :status="message.status" 
      :is-mine="isMine"
      @retry="$emit('retry')"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, inject, ref, type Ref } from 'vue'
import type { ChatMessage } from '@/types/im'
import MessageStatus from './MessageStatus.vue'
import ImageView from '@/components/common/ImageView.vue'
import { imagePreview } from '@/plugins/imagePreview'

const props = defineProps<{
  message: ChatMessage
  isMine: boolean
}>()

defineEmits<{
  (e: 'retry'): void
}>()

// 注入所有图片URL列表，确保是 ref 类型
const chatImages = inject<Ref<string[]>>('chatImages', ref([]))

// 实际图片尺寸
const actualWidth = ref(0)
const actualHeight = ref(0)

// 处理图片加载完成
const handleImageLoad = (event: Event) => {
  const img = event.target as HTMLImageElement
  actualWidth.value = img.naturalWidth
  actualHeight.value = img.naturalHeight
}

// 从 content 中获取预设尺寸（可选）
const getMetaDimensions = () => {
  const w = Number(props.message.content.width)
  const h = Number(props.message.content.height)
  if (w > 0 && h > 0) return { width: w, height: h }
  return null
}

// 获取有效的宽高数据
const getValidDimensions = () => {
  // 优先使用加载后的实际尺寸
  if (actualWidth.value && actualHeight.value) {
    return {
      width: actualWidth.value,
      height: actualHeight.value
    }
  }
  
  // 其次使用 meta 中的预设尺寸
  const metaDims = getMetaDimensions()
  if (metaDims) return metaDims
  
  // 最后使用默认尺寸 (4:3)
  return {
    width: 400,
    height: 300
  }
}

// 计算容器样式
const containerStyle = computed(() => {
  const { width, height } = getValidDimensions()
  const safeWidth = Math.max(width, 1)
  const safeHeight = Math.max(height, 1)
  const ratio = safeWidth / safeHeight
  const maxBubbleWidth = 220
  const maxBubbleHeight = 280
  const minBubbleWidth = 96
  const minBubbleHeight = 96
  const rawWidth = ratio >= 1
    ? maxBubbleWidth
    : Math.min(maxBubbleWidth, maxBubbleHeight * ratio)
  const rawHeight = ratio >= 1
    ? Math.min(maxBubbleHeight, maxBubbleWidth / ratio)
    : maxBubbleHeight
  const displayWidth = Math.round(Math.max(minBubbleWidth, rawWidth))
  const displayHeight = Math.round(Math.max(minBubbleHeight, rawHeight))
  
  return {
    width: `${displayWidth}px`,
    height: `${displayHeight}px`,
  }
})

// 预览图片
const preview = () => {
  if (!props.message.content.url) return
  // 找到当前图片在列表中的索引
  const index = chatImages.value.findIndex((url: string) => url === props.message.content.url)
  if (index === -1) return
  
  imagePreview.preview({
      images: chatImages.value,
      startPosition: index >= 0 ? index : 0,
      showIndex: chatImages.value.length > 1
    })
}
</script>

<style scoped>
.message-content.message-image {
  position: relative;
  padding: 0;
  overflow: visible;
  background: transparent;
}

.image-container {
  position: relative;
  overflow: hidden;
  border-radius: 10px;
  background: #eef1f6;
}

.image-wrapper {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.image-wrapper :deep(.image-view) {
  width: 100%;
  height: 100%;
}

/* 上传中的遮罩层 */
.upload-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background:
    linear-gradient(180deg, rgba(15, 23, 42, 0.18), rgba(15, 23, 42, 0.34)),
    rgba(15, 23, 42, 0.18);
  backdrop-filter: blur(1px);
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: inherit;
}

.upload-indicator {
  min-width: 72px;
  padding: 8px 10px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.58);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  font-size: 12px;
  line-height: 1;
}

.upload-spinner {
  width: 15px;
  height: 15px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
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

:deep(.message-mine) .message-content {
  background: var(--primary-color);
}

:deep(.message-mine) .message-arrow {
  left: auto;
  right: -6px;
  border-width: 6px 0 6px 6px;
  border-color: transparent transparent transparent var(--primary-color);
}

.message-mine .message-arrow {
  left: auto;
  right: -6px;
  border-width: 6px 0 6px 6px;
  border-color: transparent transparent transparent var(--bg-color);
}

.message-content.message-image :deep(.message-retry-btn) {
  left: -36px;
  top: 4px;
  z-index: 2;
}

</style> 
