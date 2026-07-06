<!-- 链接消息组件 -->
<template>
    <div class="message-content message-card">
      <div class="message-arrow"></div>
      <a :href="message.content.url" target="_blank" class="card-link">
        <div class="card-info">
          <div class="card-title">{{ message.content.title || '链接' }}</div>
          <div class="card-desc">{{ message.content.description || message.content_preview }}</div>
        </div>
        <div class="card-thumb">
          <template v-if="!imageError">
          <img 
            v-if="message.content.thumb_url || message.content.image_url"
            :src="message.content.thumb_url || message.content.image_url" 
            alt="链接"
            @error="imageError = true"
          >
          <PlaceholderImage 
            v-else
            text="链接"
            bgColor="#EFF1F8"
            width="48px"
            height="48px"
          />
        </template>
        <PlaceholderImage 
          v-else
          text="链接"
          bgColor="#EFF1F8"
          width="48px"
          height="48px"
        />
  
        </div>
      </a>
      <MessageStatus 
        v-if="isMine" 
        :status="message.status" 
        @retry="$emit('retry')" 
      />
    </div>
  </template>
  
  <script setup lang="ts">
  import { ref } from 'vue'
  import type { Message } from '@/types/im'
  import MessageStatus from './MessageStatus.vue'
  import PlaceholderImage from '../common/PlaceholderImage.vue'
  
  defineProps<{
    message: Message
    isMine: boolean
  }>()
  
  defineEmits<{
    (e: 'retry'): void
  }>()
  
  const imageError = ref(false)
  
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
    max-width: 280px;
    width: 80%;
    padding: 0;
  }
  
  .card-link {
    display: flex;
    gap: 8px;
    padding: 8px;
    border-radius: 8px;
    text-decoration: none;
    border: 1px solid var(--border-color-light);
    max-width: 100%;
  }
  
  .card-info {
    flex: 1;
    min-width: 0;
  }
  
  .card-title {
    font-size: 14px;
    font-weight: 500;
    color: var(--text-color-dark);
    margin-bottom: 4px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  
  .card-desc {
    font-size: 12px;
    color: var(--text-color-light);
    overflow: hidden;
    text-overflow: ellipsis;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }
  
  .card-thumb {
    width: 80px;
    height: 80px;
    flex-shrink: 0;
    border-radius: 4px;
    overflow: hidden;
    background: #EFF1F8;
    color: var(--text-color-secondary);
  }
  
  .card-thumb img {
    width: 100%;
    height: 100%;
    object-fit: cover;
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
