<!-- 消息状态组件 -->
<template>
  <button
    v-if="status === 'failed'"
    class="message-retry-btn"
    type="button"
    aria-label="重发"
    @click.stop="$emit('retry')"
  >
    <i class="ri-refresh-line"></i>
  </button>
</template>

<script setup lang="ts">
defineProps<{
  status?: 'sending' | 'failed' | 'sent' | 'delivered' | 'read' | 'recalled'
}>()

defineEmits<{
  (e: 'retry'): void
}>()
</script>

<style scoped>
.message-retry-btn {
  position: absolute;
  left: -34px;
  top: 2px;
  width: 26px;
  height: 26px;
  border: none;
  border-radius: 50%;
  background: #fff1f0;
  color: var(--error-color);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(15, 23, 42, 0.12);
}

.message-retry-btn i {
  font-size: 15px;
  line-height: 1;
}

.message-retry-btn:active {
  opacity: 0.75;
}
</style> 
