<!-- 图片显示组件 -->
<template>
  <div 
    class="image-view"
    :style="{
      width: typeof width === 'number' ? `${width}px` : width,
      height: typeof height === 'number' ? `${height}px` : height,
      maxWidth
    }"
  >
    <template v-if="!error">
      <img
        v-show="!loading" 
        ref="imageRef"
        :src="src"
        :alt="alt"
        @load="handleLoad"
        @error="handleError"
        class="image"
        :style="{
          objectFit: fit,
          height: height ? '100%' : 'auto'
        }"
      >
      <div v-show="loading" class="loading">
        <div class="spinner"></div>
      </div>
    </template>
    
    <div v-else class="placeholder">
      <PlaceholderImage
        :text="placeholderText"
        :bgColor="placeholderBgColor"
        width="100%"
        height="100%"
        aspect="1:1"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import PlaceholderImage from './PlaceholderImage.vue'

const props = withDefaults(defineProps<{
  src: string
  alt?: string
  width?: string | number
  height?: string | number
  maxWidth?: string
  fit?: 'cover' | 'contain'
  placeholderText?: string
  placeholderBgColor?: string
}>(), {
  alt: '',
  width: '100%',
  height: '',
  maxWidth: '180px',
  fit: 'contain',
  placeholderText: '',
  placeholderBgColor: '#EFF1F8'
})

const loading = ref(true)
const error = ref(false)
const imageRef = ref<HTMLImageElement | null>(null)
const isLandscape = ref(false)

const handleLoad = () => {
  loading.value = false
  
  // 获取图片的实际宽高比
  if (imageRef.value) {
    const { naturalWidth, naturalHeight } = imageRef.value
    isLandscape.value = naturalWidth > naturalHeight
  }
}

const handleError = () => {
  loading.value = false
  error.value = true
}
</script>

<style scoped>
.image-view {
  position: relative;
  overflow: hidden;
  background: var(--bg-color-light);
  border-radius: inherit;
  line-height: 0;
}

.image {
  width: 100%;
  display: block;
  border-radius: inherit;
}

.loading {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-color-light);
}

.placeholder {
  width: 100%;
  aspect-ratio: 1/1;
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
</style> 
