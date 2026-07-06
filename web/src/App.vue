<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { RouterView } from 'vue-router'
import { usePageNotificationStore } from '@/stores/pageNotification'

const pageNotificationStore = usePageNotificationStore()

onMounted(() => {
  pageNotificationStore.mount()
})

onUnmounted(() => {
  pageNotificationStore.unmount()
})
</script>

<template>

  <RouterView />
</template>
