import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import { request } from '@/utils/request'
import {
  AuthActionError,
  exchangeAccessToken,
  loginWithPassword,
  logoutSession,
  refreshAccessToken,
} from '@/utils/authRefresh'
import { useConversationList } from '@/composables/useConversationList'
import { useIMStore } from '@/stores/im'
import { useIMTabStore } from '@/stores/imTab'
import { useUnreadMessageStore } from '@/stores/unreadMessage'
import { getRefreshScheduleDelayMs, getTokenExpiryMs, isTokenExpiringSoon } from '@/utils/token'
import type { UserInfo } from '@/types/user'
import { SUCCESS_CODE, type ApiResponse } from '@/types/api'

let refreshPromise: Promise<string | null> | null = null
let authChannel: BroadcastChannel | null = null
let refreshTimer: ReturnType<typeof setTimeout> | null = null
let visibilityBound = false
let authChannelBound = false
const AUTH_REFRESH_LOCK_KEY = 'd-im-auth-refresh-lock'
const REFRESH_TOKEN_KEY = 'd-im-refresh-token'
const AUTH_REFRESH_LOCK_TTL_MS = 15_000

type AuthMessage =
  | { type: 'token-updated'; token: string; refreshToken: string }
  | { type: 'logout' }

interface EnsureValidTokenOptions {
  force?: boolean
  logoutOnAuthError?: boolean
}

interface LogoutOptions {
  broadcast?: boolean
  revokeSession?: boolean
}

interface SetTokenOptions {
  broadcast?: boolean
}

function readStoredRefreshToken() {
  if (typeof localStorage === 'undefined') {
    return null
  }
  return localStorage.getItem(REFRESH_TOKEN_KEY)
}

function writeStoredRefreshToken(nextToken: string | null) {
  if (typeof localStorage === 'undefined') {
    return
  }
  if (nextToken) {
    localStorage.setItem(REFRESH_TOKEN_KEY, nextToken)
  } else {
    localStorage.removeItem(REFRESH_TOKEN_KEY)
  }
}

function createAuthChannel() {
  if (typeof window === 'undefined' || authChannel) {
    return authChannel
  }

  authChannel = new BroadcastChannel('d-im-auth')
  return authChannel
}

function clearRefreshTimer() {
  if (refreshTimer) {
    clearTimeout(refreshTimer)
    refreshTimer = null
  }
}

function getAuthTabId() {
  if (typeof sessionStorage === 'undefined') {
    return 'server'
  }

  const existing = sessionStorage.getItem('d-im-auth-tab-id')
  if (existing) {
    return existing
  }

  const nextId =
    typeof crypto !== 'undefined' && 'randomUUID' in crypto
      ? crypto.randomUUID()
      : `${Date.now()}-${Math.random().toString(36).slice(2)}`
  sessionStorage.setItem('d-im-auth-tab-id', nextId)
  return nextId
}

function readRefreshLock() {
  if (typeof localStorage === 'undefined') {
    return null
  }

  const raw = localStorage.getItem(AUTH_REFRESH_LOCK_KEY)
  if (!raw) {
    return null
  }

  try {
    return JSON.parse(raw) as { owner: string; expiresAt: number }
  } catch {
    return null
  }
}

function acquireRefreshLock(owner: string) {
  if (typeof localStorage === 'undefined') {
    return true
  }

  const now = Date.now()
  const current = readRefreshLock()
  if (current && current.expiresAt > now && current.owner !== owner) {
    return false
  }

  const next = JSON.stringify({
    owner,
    expiresAt: now + AUTH_REFRESH_LOCK_TTL_MS,
  })
  localStorage.setItem(AUTH_REFRESH_LOCK_KEY, next)

  const confirmed = readRefreshLock()
  return confirmed?.owner === owner
}

function releaseRefreshLock(owner: string) {
  if (typeof localStorage === 'undefined') {
    return
  }

  const current = readRefreshLock()
  if (current?.owner === owner) {
    localStorage.removeItem(AUTH_REFRESH_LOCK_KEY)
  }
}

function waitForRemoteAuthUpdate(timeoutMs = AUTH_REFRESH_LOCK_TTL_MS): Promise<AuthMessage | null> {
  const channel = createAuthChannel()
  if (!channel) {
    return Promise.resolve(null)
  }

  return new Promise((resolve) => {
    const timer = window.setTimeout(() => {
      channel.removeEventListener('message', onMessage)
      resolve(null)
    }, timeoutMs)

    const onMessage = (event: MessageEvent<AuthMessage>) => {
      const message = event.data
      if (!message || (message.type !== 'token-updated' && message.type !== 'logout')) {
        return
      }
      window.clearTimeout(timer)
      channel.removeEventListener('message', onMessage)
      resolve(message)
    }

    channel.addEventListener('message', onMessage)
  })
}

export const useUserStore = defineStore('user', () => {
  const token = ref<string | null>(null)
  const refreshTokenValue = ref<string | null>(readStoredRefreshToken())
  const userInfo = ref<UserInfo | null>(null)
  const sessionExpired = ref(false)
  const unreadMessageStore = useUnreadMessageStore()
  const authTabId = getAuthTabId()

  const scheduleRefresh = (nextToken: string | null) => {
    clearRefreshTimer()
    if (!nextToken) {
      return
    }

    const delay = getRefreshScheduleDelayMs(nextToken)
    if (delay == null) {
      return
    }

    refreshTimer = setTimeout(() => {
      void ensureValidToken({ force: true, logoutOnAuthError: true })
    }, delay)
  }

  const bindVisibilityRefresh = () => {
    if (typeof document === 'undefined' || visibilityBound) {
      return
    }

    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'visible') {
        void ensureValidToken({ logoutOnAuthError: true })
      }
    })
    visibilityBound = true
  }

  const setToken = (newToken: string, newRefreshToken: string, options: SetTokenOptions = {}) => {
    token.value = newToken
    refreshTokenValue.value = newRefreshToken
    writeStoredRefreshToken(newRefreshToken)
    scheduleRefresh(newToken)

    const imStore = useIMStore()
    imStore.syncAccessToken(newToken)

    if (options.broadcast !== false) {
      createAuthChannel()?.postMessage({
        type: 'token-updated',
        token: newToken,
        refreshToken: newRefreshToken,
      } satisfies AuthMessage)
    }
  }

  const clearLocalAuthState = (broadcast = false) => {
    clearRefreshTimer()
    token.value = null
    refreshTokenValue.value = null
    writeStoredRefreshToken(null)
    userInfo.value = null

    const imTabStore = useIMTabStore()
    imTabStore.reset()

    const imStore = useIMStore()
    imStore.closeConnection()

    if (broadcast) {
      createAuthChannel()?.postMessage({ type: 'logout' } satisfies AuthMessage)
    }
  }

  const applyLogoutSideEffects = () => {
    const { resetConversations } = useConversationList()
    resetConversations()
  }

  const logout = async (options: LogoutOptions = {}) => {
    const { broadcast = true, revokeSession = true } = options
    const currentRefreshToken = refreshTokenValue.value

    clearLocalAuthState(broadcast)
    applyLogoutSideEffects()

    if (revokeSession) {
      try {
        await logoutSession(currentRefreshToken)
      } catch (error) {
        console.error('登出会话失败:', error)
      }
    }
  }

  const hasUsableToken = (value: string | null) => {
    if (!value) {
      return false
    }
    const expiryMs = getTokenExpiryMs(value)
    return expiryMs != null && expiryMs > Date.now()
  }

  const refreshToken = async (
    options: Pick<EnsureValidTokenOptions, 'logoutOnAuthError'> & { broadcast?: boolean } = {},
  ): Promise<string | null> => {
    if (refreshPromise) {
      return refreshPromise
    }

    if (!acquireRefreshLock(authTabId)) {
      const remoteUpdate = await waitForRemoteAuthUpdate()
      if (remoteUpdate?.type === 'token-updated' && token.value) {
        return token.value
      }
      if (remoteUpdate?.type === 'logout') {
        return null
      }
      if (!acquireRefreshLock(authTabId)) {
        return token.value
      }
    }

    refreshPromise = (async () => {
      try {
        if (!refreshTokenValue.value) {
          return null
        }
        const result = await refreshAccessToken(refreshTokenValue.value)
        if (result?.access_token) {
          setToken(result.access_token, result.refresh_token, { broadcast: options.broadcast !== false })
          return result.access_token
        }
        return null
      } finally {
        releaseRefreshLock(authTabId)
        refreshPromise = null
      }
    })()

    try {
      return await refreshPromise
    } catch (error) {
      if (hasUsableToken(token.value)) {
        return token.value
      }
      if (
        error instanceof AuthActionError &&
        error.reason === 'auth' &&
        options.logoutOnAuthError !== false
      ) {
        sessionExpired.value = true
        return null
      }
      throw error
    }
  }

  const ensureValidToken = async (
    options: EnsureValidTokenOptions = {},
  ): Promise<string | null> => {
    // 会话已过期（Modal 展示中），不再尝试刷新，直接返回 null
    if (sessionExpired.value) {
      return null
    }

    const currentToken = token.value
    if (!options.force && currentToken && !isTokenExpiringSoon(currentToken)) {
      return currentToken
    }

    try {
      return await refreshToken({ broadcast: true, logoutOnAuthError: options.logoutOnAuthError })
    } catch (error) {
      if (!options.force && hasUsableToken(currentToken)) {
        return currentToken
      }
      if (
        error instanceof AuthActionError &&
        error.reason === 'auth' &&
        options.logoutOnAuthError !== false
      ) {
        sessionExpired.value = true
        return null
      }
      return null
    }
  }

  const initialize = async () => {
    bindVisibilityRefresh()
    const channel = createAuthChannel()
    if (channel && !authChannelBound) {
      channel.addEventListener('message', (event: MessageEvent<AuthMessage>) => {
        const message = event.data
        if (!message) {
          return
        }
        if (message.type === 'token-updated') {
          setToken(message.token, message.refreshToken, { broadcast: false })
          if (!userInfo.value) {
            void fetchUser().catch((error) => {
              console.error('同步用户信息失败:', error)
            })
          }
          return
        }
        if (message.type === 'logout') {
          clearLocalAuthState(false)
          applyLogoutSideEffects()
        }
      })
      authChannelBound = true
    }

    const restoredToken = await ensureValidToken({ force: true, logoutOnAuthError: false })
    if (!restoredToken) {
      clearLocalAuthState(false)
      return
    }

    try {
      await fetchUser()
    } catch (error) {
      console.error('初始化用户信息失败:', error)
      await logout({ revokeSession: false })
    }
  }

  const establishSession = async (ticket: string): Promise<string> => {
    const result = await exchangeAccessToken(ticket)
    setToken(result.access_token, result.refresh_token)
    return result.access_token
  }

  const login = async (id: string, password: string): Promise<string> => {
    const result = await loginWithPassword(id, password)
    setToken(result.access_token, result.refresh_token)
    return result.access_token
  }

  const setUserInfo = (info: UserInfo) => {
    userInfo.value = info
  }

  const fetchUser = async (): Promise<UserInfo> => {
    const response = await request<ApiResponse<UserInfo>>('/api/v1/users/me')
    if (response.code === SUCCESS_CODE && response.data?.id) {
      setUserInfo(response.data)
      return response.data
    }
    throw new Error('无法获取用户信息')
  }

  // 总未读消息数
  watch(token, (newToken, oldToken) => {
    if (newToken && !oldToken) {
      // unreadMessageStore.startHeartbeat()
      // void unreadMessageStore.fetchUnreadCount().catch((error) => {
      //   console.error('同步未读消息总数失败:', error)
      // })
    } else if (!newToken && oldToken) {
      unreadMessageStore.reset()
      unreadMessageStore.stopHeartbeat()
    }
  })

  const confirmSessionExpired = async () => {
    sessionExpired.value = false
    await logout({ revokeSession: false })
  }

  return {
    token,
    userInfo,
    sessionExpired,
    setToken,
    setUserInfo,
    logout,
    confirmSessionExpired,
    refreshToken,
    ensureValidToken,
    fetchUser,
    initialize,
    establishSession,
    login,
  }
})
