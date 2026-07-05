<template>
  <div class="enter-page">
    <div v-if="loading" class="state">
      <div class="spinner"></div>
      <p>正在进入…</p>
    </div>
    <div v-else-if="error" class="state state-error">
      <p>{{ error }}</p>
      <router-link :to="{ name: 'im-login' }" class="link">返回</router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useAuthCallback } from '@/composables/useAuthCallback'

const { loading, error, completeEnter } = useAuthCallback()

onMounted(() => {
  completeEnter()
})
</script>

<style scoped>
.enter-page {
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f6fa;
}

.state {
  text-align: center;
  color: #666;
}

.state-error {
  color: #c0392b;
}

.spinner {
  width: 32px;
  height: 32px;
  margin: 0 auto 12px;
  border: 3px solid #e0e0e0;
  border-top-color: #4b86f8;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.link {
  display: inline-block;
  margin-top: 12px;
  color: #4b86f8;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
