import { defineStore } from 'pinia'
import { ref } from 'vue'
import { devLogin, refreshAccessToken, type TokenPair, type LoginParams, AuthActionError } from '@/utils/authRefresh'
import { useIMStore } from '@/stores/im'

const REFRESH_TOKEN_KEY = 'd-im-refresh-token'
const ACCESS_TOKEN_KEY = 'd-im-access-token'

let refreshTimer: ReturnType<typeof setTimeout> | null = null

export const useUserStore = defineStore('user', () => {
  const token = ref<string | null>(localStorage.getItem(ACCESS_TOKEN_KEY))
  const refreshTokenValue = ref<string | null>(localStorage.getItem(REFRESH_TOKEN_KEY))
  const deviceId = ref<string>('')
  const userInfo = ref<{ id: string; nickname?: string; avatar?: string } | null>(null)
  const sessionExpired = ref(false)

  function clearRefreshTimer() {
    if (refreshTimer) {
      clearTimeout(refreshTimer)
      refreshTimer = null
    }
  }

  function scheduleRefresh(ttl: number) {
    clearRefreshTimer()
    const delay = Math.max(ttl * 1000 * 0.7, 10_000) // 70% 过期前刷新
    refreshTimer = setTimeout(() => {
      void refresh().catch(() => {})
    }, delay)
  }

  /** 开发模式登录 */
  async function login(uid: string, did: string) {
    deviceId.value = did
    const pair = await devLogin({ uid, device_id: did })
    saveTokens(pair)
    return pair
  }

  /** 保存 token 到 localStorage */
  function saveTokens(pair: TokenPair) {
    token.value = pair.access_token
    refreshTokenValue.value = pair.refresh_token
    localStorage.setItem(ACCESS_TOKEN_KEY, pair.access_token)
    localStorage.setItem(REFRESH_TOKEN_KEY, pair.refresh_token)
    scheduleRefresh(pair.expires_in)

    const imStore = useIMStore()
    imStore.syncAccessToken(pair.access_token)
  }

  /** 刷新 token */
  async function refresh(): Promise<string | null> {
    if (!refreshTokenValue.value) return null
    try {
      const pair = await refreshAccessToken(refreshTokenValue.value)
      saveTokens(pair)
      return pair.access_token
    } catch (e) {
      if (e instanceof AuthActionError && e.reason === 'auth') {
        sessionExpired.value = true
      }
      return null
    }
  }

  /** 确保 token 有效 */
  async function ensureValidToken(): Promise<string | null> {
    if (token.value) return token.value
    return refresh()
  }

  /** 登出 */
  async function logout() {
    clearRefreshTimer()
    token.value = null
    refreshTokenValue.value = null
    localStorage.removeItem(ACCESS_TOKEN_KEY)
    localStorage.removeItem(REFRESH_TOKEN_KEY)

    const imStore = useIMStore()
    imStore.closeConnection()
  }

  return {
    token,
    deviceId,
    userInfo,
    refreshToken: refreshTokenValue,
    sessionExpired,
    login,
    logout,
    refresh,
    ensureValidToken,
    saveTokens,
  }
})