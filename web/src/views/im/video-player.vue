<template>
  <main class="video-player-page">
    <video
      v-if="videoUrl"
      class="video-player"
      :src="videoUrl"
      :poster="poster"
      :type="mimeType"
      controls
      autoplay
      playsinline
    >
      您的设备不支持视频播放
    </video>
    <div v-else class="video-empty">
      <i class="ri-error-warning-line"></i>
      <span>视频地址无效</span>
    </div>
  </main>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()

const firstQueryValue = (value: unknown) => {
  if (Array.isArray(value)) return value[0] ?? ''
  return typeof value === 'string' ? value : ''
}

const videoUrl = computed(() => firstQueryValue(route.query.url))
const poster = computed(() => firstQueryValue(route.query.poster))
const format = computed(() => firstQueryValue(route.query.type).toLowerCase())
const mimeType = computed(() => {
  if (!format.value) return 'video/mp4'
  if (format.value === 'mov') return 'video/quicktime'
  return `video/${format.value}`
})
</script>

<style scoped>
.video-player-page {
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: #020617;
}

.video-player {
  width: min(100%, 1080px);
  max-height: calc(100dvh - 32px);
  border-radius: 12px;
  background: black;
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.4);
}

.video-empty {
  display: flex;
  flex-direction: column;
  gap: 12px;
  align-items: center;
  color: rgba(255, 255, 255, 0.72);
}

.video-empty i {
  font-size: 42px;
}
</style>
