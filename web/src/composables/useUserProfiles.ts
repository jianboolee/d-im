import { ref, computed } from 'vue'
import { request } from '@/utils/request'
import { SUCCESS_CODE, type ApiResponse } from '@/types/api'
import type { UserInfo } from '@/types/user'

const cache = ref<Record<string, UserInfo>>({})
const pending = new Map<string, Promise<UserInfo | null>>()

export function useUserProfiles() {
  async function fetchUser(userId: string): Promise<UserInfo | null> {
    if (!userId) return null

    const cached = cache.value[userId]
    if (cached) return cached

    const existing = pending.get(userId)
    if (existing) return existing

    const promise = (async () => {
      try {
        const response = await request<ApiResponse<UserInfo>>(`/api/v1/users/${userId}`)
        if (response.code === SUCCESS_CODE) {
          cache.value = { ...cache.value, [userId]: response.data }
          return response.data
        }
      } catch (error) {
        console.error('获取用户信息失败:', error)
      } finally {
        pending.delete(userId)
      }

      return null
    })()

    pending.set(userId, promise)
    return promise
  }

  async function fetchUsers(userIds: Iterable<string>) {
    const uniqueIds = [...new Set(userIds)].filter(Boolean)
    await Promise.all(uniqueIds.map((userId) => fetchUser(userId)))
  }

  function mergeUsers(users: Iterable<UserInfo | null | undefined>) {
    const next = { ...cache.value }
    let changed = false

    for (const user of users) {
      if (!user?.id) continue
      next[user.id] = user
      changed = true
    }

    if (changed) {
      cache.value = next
    }
  }

  const userMap = computed(() => cache.value)

  return {
    userMap,
    fetchUser,
    fetchUsers,
    mergeUsers,
  }
}
