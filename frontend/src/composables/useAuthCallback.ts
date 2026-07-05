import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useIMStore } from '@/stores/im'
import { exchangeAccessToken } from '@/utils/authRefresh'

export function useAuthCallback() {
  const route = useRoute()
  const router = useRouter()
  const userStore = useUserStore()
  const imStore = useIMStore()

  const loading = ref(true)
  const error = ref<string | null>(null)

  async function completeEnter() {
    const ticket = typeof route.query.ticket === 'string' ? route.query.ticket : ''

    if (!ticket) {
      error.value = '缺少 ticket 参数'
      loading.value = false
      return
    }

    try {
      const deviceId = 'web_sso_' + Date.now()
      const pair = await exchangeAccessToken(ticket, deviceId)
      userStore.saveTokens(pair)
      userStore.deviceId = deviceId

      imStore.initSDK()
      await router.replace({ name: 'im-chat-index' })
    } catch (err) {
      console.error('SSO 登录失败:', err)
      error.value = '登录失败，请从业务系统重新进入'
    } finally {
      loading.value = false
    }
  }

  return { loading, error, completeEnter }
}