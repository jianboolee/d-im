import { onMounted, onUnmounted } from 'vue'
import { usePageNotificationStore } from '@/stores/pageNotification'

export function usePageTitleNotification(defaultTitle = '消息') {
  const notificationStore = usePageNotificationStore()

  const setBaseTitle = (title: string) => {
    notificationStore.setBaseTitle(title)
  }

  onMounted(() => {
    setBaseTitle(defaultTitle)
  })

  onUnmounted(() => {
    notificationStore.setBaseTitle('消息')
  })

  return {
    setBaseTitle,
    restoreTitle: notificationStore.clearNotice,
  }
}
