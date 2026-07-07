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
    const chatId =
      typeof route.query.chat_id === 'string' ? route.query.chat_id : ''
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

      if (chatId) {
        await router.replace({ name: 'im-chat', params: { chatId } })
        imStore.initSDK()
        return
      }

      // 无 chat_id：进入会话列表。旧 conversation_id 入口保留一次转换。
      if (!conversationId) {
        await router.replace({ name: 'im-chat-index' })
        imStore.initSDK()
        return
      }

      const sdk = imStore.initSDK()
      if (!sdk) {
        throw new Error('IM SDK 初始化失败')
      }
      const conversation = await sdk.getConversation(conversationId)
      await router.replace({ name: 'im-chat', params: { chatId: conversation.chat_id } })
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
