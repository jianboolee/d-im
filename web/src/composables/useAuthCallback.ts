import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useIMStore } from '@/stores/im'
export function useAuthCallback() {
  const route = useRoute()
  const router = useRouter()
  const userStore = useUserStore()
  const imStore = useIMStore()

  const loading = ref(true)
  const error = ref<string | null>(null)

  async function completeEnter() {
    const ticket = typeof route.query.ticket === 'string' ? route.query.ticket : ''
    const conversationId =
      typeof route.query.conversation_id === 'string' ? route.query.conversation_id : ''

    if (!ticket) {
      error.value = '缺少 ticket 参数'
      loading.value = false
      return
    }

    try {
      await userStore.establishSession(ticket)
      await userStore.fetchUser().catch((fetchError) => {
        console.warn('同步用户信息失败:', fetchError)
      })

      // 无 conversation_id：进入会话列表
      if (!conversationId) {
        await router.replace({ name: 'im-chat-index' })
        imStore.initSDK()
        return
      }

      await router.replace({ name: 'im-chat', params: { conversationId } })
      imStore.initSDK()
    } catch (err) {
      console.error('SSO 登录失败:', err)
      error.value = '登录失败，请从业务系统重新进入'
      userStore.logout()
    } finally {
      loading.value = false
    }
  }

  return {
    loading,
    error,
    completeEnter,
  }
}
