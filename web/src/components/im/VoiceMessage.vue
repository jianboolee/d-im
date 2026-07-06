<!-- 语音消息组件 -->
<template>
  <div class="message-content message-voice">
    <div class="message-arrow"></div>
    <div class="voice-container" @click="togglePlay">
      <i :class="isPlaying ? 'ri-pause-fill' : 'ri-play-fill'"></i>
      <div class="voice-wave" :class="{ 'wave-animate': isPlaying }">
        <i v-for="n in 4" :key="n" class="wave-bar"></i>
      </div>
      <span class="duration" v-if="message.content.duration">
        {{ formatDuration(Number(message.content.duration)) }}
      </span>
    </div>
    <MessageStatus 
      v-if="isMine" 
      :status="message.status" 
      @retry="$emit('retry')" 
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onUnmounted } from 'vue'
import type { Message } from '@/types/im'
import MessageStatus from './MessageStatus.vue'

const props = defineProps<{
  message: Message
  isMine: boolean
}>()

defineEmits<{
  (e: 'retry'): void
}>()

const isPlaying = ref(false)
const audio = new Audio()

// 格式化时长
const formatDuration = (seconds: number) => {
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = Math.floor(seconds % 60)
  return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`
}

// 播放/暂停
const togglePlay = () => {
  if (isPlaying.value) {
    audio.pause()
    isPlaying.value = false
  } else {
    if (props.message.content.url) {
      audio.src = props.message.content.url
      audio.play()
      isPlaying.value = true
    }
  }
}

// 监听音频播放结束
audio.addEventListener('ended', () => {
  isPlaying.value = false
})

// 组件卸载时清理
onUnmounted(() => {
  audio.pause()
  audio.src = ''
})
</script>

<style scoped>
.message-content {
  position: relative;
  padding: 8px 12px;
  border-radius: 12px;
  background: var(--bg-color);
  min-width: 120px;
}
.message-content.message-voice {
  max-width: 80%;
  padding: 4px 12px;
}

.voice-container {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.voice-container i {
  font-size: 20px;
  color: var(--text-color-dark);
}

:deep(.message-mine) .voice-container i {
  color: white;
}

.voice-wave {
  display: flex;
  align-items: center;
  gap: 2px;
  height: 20px;
}

.wave-bar {
  width: 3px;
  height: 100%;
  background: var(--text-color-dark);
  opacity: 0.3;
  transform-origin: center;
}

:deep(.message-mine) .wave-bar {
  background: white;
}

.wave-animate .wave-bar {
  animation: wave 1s ease-in-out infinite;
}

.wave-animate .wave-bar:nth-child(2) {
  animation-delay: 0.2s;
}

.wave-animate .wave-bar:nth-child(3) {
  animation-delay: 0.4s;
}

.wave-animate .wave-bar:nth-child(4) {
  animation-delay: 0.6s;
}

@keyframes wave {
  0%, 100% {
    transform: scaleY(0.5);
  }
  50% {
    transform: scaleY(1);
  }
}

.duration {
  font-size: 12px;
  color: var(--text-color-dark);
}

:deep(.message-mine) .duration {
  color: white;
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

</style> 
