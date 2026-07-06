<template>
  <div class="message-content message-video">
    <div class="message-arrow"></div>
    <button class="video-card" type="button" @click="openVideo">
      <img
        v-if="poster"
        class="video-poster"
        :src="poster"
        alt=""
      >
      <div v-else class="video-placeholder">
        <i class="ri-movie-2-line"></i>
      </div>
      <div class="video-overlay">
        <span class="play-button">
          <i class="ri-play-fill"></i>
        </span>
        <span v-if="durationText" class="duration">{{ durationText }}</span>
      </div>
    </button>
    <MessageStatus
      v-if="isMine"
      :status="message.status"
      @retry="$emit('retry')"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import type { Message } from '@/types/im'
import MessageStatus from './MessageStatus.vue'

const props = defineProps<{
  message: Message
  isMine: boolean
}>()

defineEmits<{
  (e: 'retry'): void
}>()

const router = useRouter()

const videoUrl = computed(() => props.message.content.url || '')
const poster = computed(() => props.message.content.thumb_url || props.message.content.image_url || '')
const format = computed(() => props.message.content.format || '')

const durationText = computed(() => {
  const raw = props.message.content.duration
  const seconds = raw ? Number(raw) : 0
  if (!Number.isFinite(seconds) || seconds <= 0) return ''
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = Math.floor(seconds % 60)
  return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`
})

const aspectRatio = computed(() => {
  const width = props.message.content.width ? Number(props.message.content.width) : 0
  const height = props.message.content.height ? Number(props.message.content.height) : 0
  if (width && height && width > height) return '16/9'
  return '9/16'
})

const openVideo = () => {
  if (!videoUrl.value) return
  const route = router.resolve({
    name: 'im-video-player',
    query: {
      url: videoUrl.value,
      poster: poster.value || undefined,
      type: format.value || undefined,
    },
  })
  window.open(route.href, '_blank', 'noopener,noreferrer')
}
</script>

<style scoped>
.message-content {
  position: relative;
  padding: 4px;
  border-radius: 12px;
  background: var(--bg-color);
  max-width: 140px;
}

.message-content.message-video {
  width: 80%;
  padding: 0;
}

.video-card {
  position: relative;
  display: block;
  width: 100%;
  padding: 0;
  border: 0;
  border-radius: 8px;
  overflow: hidden;
  cursor: pointer;
  background: #0f172a;
  aspect-ratio: v-bind('aspectRatio');
}

.video-poster,
.video-placeholder {
  width: 100%;
  height: 100%;
}

.video-poster {
  display: block;
  object-fit: cover;
}

.video-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.72);
}

.video-placeholder i {
  font-size: 34px;
}

.video-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(180deg, rgba(15, 23, 42, 0.1), rgba(15, 23, 42, 0.38));
}

.play-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.92);
  color: #111827;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.28);
}

.play-button i {
  font-size: 26px;
  transform: translateX(1px);
}

.duration {
  position: absolute;
  right: 8px;
  bottom: 8px;
  padding: 2px 6px;
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.56);
  color: white;
  font-size: 12px;
  line-height: 18px;
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
  background: var(--bg-color);
}

:deep(.message-mine) .message-arrow,
.message-mine .message-arrow {
  left: auto;
  right: -6px;
  border-width: 6px 0 6px 6px;
  border-color: transparent transparent transparent var(--bg-color);
}
</style>
