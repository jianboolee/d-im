import { defineStore } from 'pinia'
import { ref } from 'vue'
import { normalizeUnreadCount } from '@/utils/im/format'

export const useUnreadMessageStore = defineStore('unreadMessage', () => {
  const unreadCount = ref(0)

  const setUnreadCount = (count: number) => {
    unreadCount.value = normalizeUnreadCount(count)
  }

  // 总未读暂不启用：保留 API 形状，避免调用方需要同步改动。
  const fetchUnreadCount = async (): Promise<number> => unreadCount.value
  const requestRefresh = () => {}

  const increment = (count = 1) => {
    setUnreadCount(unreadCount.value + count)
  }

  const decrement = (count = 1) => {
    setUnreadCount(unreadCount.value - count)
  }

  const reset = () => {
    setUnreadCount(0)
  }

  const startHeartbeat = () => {}
  const stopHeartbeat = () => {}

  return {
    unreadCount,
    fetchUnreadCount,
    setUnreadCount,
    increment,
    decrement,
    reset,
    startHeartbeat,
    stopHeartbeat,
    requestRefresh,
  }
})
