<template>
  <div class="page">
    <div class="nav-bar">
      <div class="nav-bar-left">
        <div class="nav-bar-title">消息</div>
      </div>
      <div class="nav-bar-right">
        <div class="nav-bar-right-item">
          <button class="btn btn-primary" type="button">
            <i class="ri-more-line"></i>
          </button>
        </div>
      </div>
    </div>

    <ConversationList navigate-mode="push" />
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useIMStore } from '@/stores/im'
import ConversationList from '@/components/im/ConversationList.vue'

const router = useRouter()
const userStore = useUserStore()
const imStore = useIMStore()

onMounted(() => {
  if (!userStore.token) {
    router.replace({ name: 'im-login', query: { redirect: '/im/home' } })
    return
  }

  imStore.initSDK()
})
</script>

<style scoped>
.page {
  height: 100dvh;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: white;
  overflow: hidden;
}

.nav-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-base);
  background: white;
  flex-shrink: 0;
}

.nav-bar-left {
  flex: 1;
}

.nav-bar-right {
  flex: 0;
}

.nav-bar-right-item {
  display: flex;
  gap: var(--spacing-mini);
}

.nav-bar-title {
  font-weight: 600;
}
</style>
