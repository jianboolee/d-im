import { defineStore } from 'pinia'
import { ref } from 'vue'
import IMSDK, { type Message, type ConnectionStatus, type MessageHandler, type ConnectionHandler } from '@/sdk/im'
import { getImSdkOptions } from '@/config'
import { useUserStore } from './user'

export const useIMStore = defineStore('im', () => {
  const imSDK = ref<IMSDK | null>(null)
  const isConnected = ref(false)
  const messageHandlers = ref<MessageHandler[]>([])
  const connectionHandlers = ref<ConnectionHandler[]>([])
  const reconnectTimer = ref<ReturnType<typeof setTimeout> | null>(null)
  const reconnectCount = ref(0)
  const connectingPromise = ref<Promise<void> | null>(null)
  const sdkToken = ref<string | null>(null)
  const maxReconnectAttempts = 5
  const baseReconnectDelay = 3000
  const userStore = useUserStore()
  const manualDisconnect = ref(false)

  const canUseConnection = () => true

  const clearReconnectTimer = () => {
    if (reconnectTimer.value) { clearTimeout(reconnectTimer.value); reconnectTimer.value = null }
  }

  const handleReconnectError = () => {
    if (!canUseConnection() || manualDisconnect.value) return
    reconnectCount.value++
    if (reconnectCount.value >= maxReconnectAttempts) return
    const delay = baseReconnectDelay * Math.min(Math.pow(2, reconnectCount.value - 1), 10)
    clearReconnectTimer()
    reconnectTimer.value = setTimeout(reconnect, delay)
  }

  const reconnect = async () => {
    if (!canUseConnection() || manualDisconnect.value || isConnected.value) return
    try { await connectWithCurrentToken() }
    catch (e) { console.error('重连失败:', e); handleReconnectError() }
  }

  const createSDK = (token: string) => {
    const { baseURL, wsURL } = getImSdkOptions()
    const deviceId = userStore.deviceId || 'web_unknown'
    console.log('[im-store] init SDK', { baseURL, wsURL, deviceId })

    imSDK.value = new IMSDK({ baseURL, wsURL, token, deviceId })
    sdkToken.value = token

    imSDK.value.onConnection((status: ConnectionStatus) => {
      console.log('[im-store] WS status:', status.status)
      isConnected.value = status.status === 'connected'
      if (status.status === 'disconnected' && !manualDisconnect.value && !connectingPromise.value) {
        handleReconnectError()
      }
    })

    imSDK.value.onMessage((_msg: Message) => {
      // 全局消息处理（未读计数等业务逻辑）
    })
  }

  const getFreshToken = async (): Promise<string | null> => {
    return userStore.ensureValidToken()
  }

  const syncAccessToken = (token: string) => {
    if (!token || !imSDK.value) return
    imSDK.value.updateToken(token)
    sdkToken.value = token
    if (!isConnected.value && !connectingPromise.value && !manualDisconnect.value) {
      void connectWithCurrentToken()
    }
  }

  const connectWithCurrentToken = async () => {
    if (!canUseConnection() || manualDisconnect.value || isConnected.value) return
    if (connectingPromise.value) return connectingPromise.value

    connectingPromise.value = (async () => {
      const token = await getFreshToken()
      if (!token || manualDisconnect.value || !canUseConnection()) return
      if (!imSDK.value) createSDK(token)
      else if (sdkToken.value !== token) { imSDK.value.updateToken(token); sdkToken.value = token }
      clearReconnectTimer()
      await imSDK.value?.connect()
      if (!manualDisconnect.value && canUseConnection()) {
        isConnected.value = true
        reconnectCount.value = 0
      }
    })()

    try { await connectingPromise.value }
    catch (e) {
      if (!manualDisconnect.value) { isConnected.value = false; handleReconnectError() }
    } finally { connectingPromise.value = null }
  }

  const initSDK = () => {
    if (!canUseConnection()) return imSDK.value
    manualDisconnect.value = false
    if (!userStore.token) return null
    if (!imSDK.value) createSDK(userStore.token)
    void connectWithCurrentToken()
    return imSDK.value
  }

  const closeConnection = () => {
    manualDisconnect.value = true
    clearReconnectTimer()
    connectingPromise.value = null
    if (imSDK.value) {
      imSDK.value.disconnect()
      imSDK.value = null
    }
    isConnected.value = false
    reconnectCount.value = 0
    sdkToken.value = null
  }

  const addMessageHandler = (h: MessageHandler) => { messageHandlers.value.push(h); imSDK.value?.onMessage(h) }
  const removeMessageHandler = (h: MessageHandler) => {
    const i = messageHandlers.value.indexOf(h)
    if (i > -1) { messageHandlers.value.splice(i, 1); imSDK.value?.offMessage(h) }
  }

  return {
    imSDK, isConnected,
    initSDK, closeConnection,
    addMessageHandler, removeMessageHandler,
    reconnect, syncAccessToken,
  }
})