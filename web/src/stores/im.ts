import { defineStore } from 'pinia'
import { ref } from 'vue'
import IMSDK, { type Message, type ConnectionStatus, type MessageHandler, type ConnectionHandler, MessageType } from '@/sdk/im'
import { getImSdkOptions } from '@/config'
import { useUserStore } from './user'
import { usePageNotificationStore } from './pageNotification'
import { useIMTabStore } from './imTab'

export const useIMStore = defineStore('im', () => {
    const imSDK = ref<IMSDK | null>(null)
    const isConnected = ref(false)
    const messageHandlers = ref<MessageHandler[]>([])
    const connectionHandlers = ref<ConnectionHandler[]>([])
    const reconnectTimer = ref<ReturnType<typeof setTimeout> | null>(null)
    const heartbeatTimer = ref<ReturnType<typeof setInterval> | null>(null)
    const reconnectCount = ref(0)
    const connectingPromise = ref<Promise<void> | null>(null)
    const sdkToken = ref<string | null>(null)
    const maxReconnectAttempts = 5
    const baseReconnectDelay = 3000
    const userStore = useUserStore()
    const pageNotificationStore = usePageNotificationStore()
    const manualDisconnect = ref(false)
    const imTabStore = useIMTabStore()

    const canUseConnection = () => !imTabStore.initialized || imTabStore.isPrimaryTab

    const clearReconnectTimer = () => {
        if (reconnectTimer.value) {
            clearTimeout(reconnectTimer.value)
            reconnectTimer.value = null
        }
    }

    // 获取未读消息总数
    const fetchUnreadCount = async () => {}

    // 重连方法
    const reconnect = async () => {
        if (!canUseConnection()) return
        if (manualDisconnect.value) return
        if (isConnected.value) return

        try {
            await connectWithCurrentToken()
        } catch (error) {
            console.error('重连失败:', error)
            handleReconnectError()
        }
    }

    // 处理重连错误
    const handleReconnectError = () => {
        if (!canUseConnection()) return
        if (manualDisconnect.value) return
        
        reconnectCount.value++

        if (reconnectCount.value >= maxReconnectAttempts) {
            console.error('重连失败次数过多')
            return
        }

        const delay = baseReconnectDelay * Math.min(Math.pow(2, reconnectCount.value - 1), 10)
        console.log(`将在 ${delay/1000} 秒后进行第 ${reconnectCount.value} 次重连`)

        clearReconnectTimer()
        reconnectTimer.value = setTimeout(reconnect, delay)
    }

    const createSDK = (token: string) => {
        const { baseURL, wsURL } = getImSdkOptions()
        console.log('初始化 IM SDK', { baseURL, wsURL })

        imSDK.value = new IMSDK({
            baseURL,
            wsURL,
            token,
        })
        sdkToken.value = token

        // 添加全局连接状态处理器
        imSDK.value.onConnection((status: ConnectionStatus) => {
            console.log('WebSocket连接状态变化:', status)
            isConnected.value = status.status === 'connected'

            if (status.status === 'connected') {
                imSDK.value?.startHeartbeat()
            } else if (status.status === 'disconnected' && !manualDisconnect.value && !connectingPromise.value) {
                imSDK.value?.stopHeartbeat()
                handleReconnectError()
            }
        })

        // 添加全局消息处理器，用于更新未读消息计数
        imSDK.value.onMessage((message: Message) => {
            if (
                message.type !== MessageType.Pong &&
                message.type !== MessageType.Ping &&
                message.sender_id !== userStore.userInfo?.id &&
                !message._ws_muted
            ) {
                pageNotificationStore.notifyNewMessage()
            }
        })
    }

    const getFreshToken = async () => {
        return await userStore.ensureValidToken({ logoutOnAuthError: true })
    }

    const syncAccessToken = (token: string) => {
        if (!token) return

        if (!imSDK.value) {
            return
        }

        imSDK.value.updateToken(token)
        sdkToken.value = token

        if (!isConnected.value && !connectingPromise.value && !manualDisconnect.value) {
            void connectWithCurrentToken()
        }
    }

    const connectWithCurrentToken = async () => {
        if (!canUseConnection()) return
        if (manualDisconnect.value) return
        if (isConnected.value) return
        if (connectingPromise.value) return connectingPromise.value

        connectingPromise.value = (async () => {
            const token = await getFreshToken()
            if (!token || manualDisconnect.value || !canUseConnection()) return

            if (!imSDK.value) {
                createSDK(token)
            } else if (sdkToken.value !== token) {
                imSDK.value.updateToken(token)
                sdkToken.value = token
            }

            clearReconnectTimer()
            await imSDK.value?.connect()

            if (!manualDisconnect.value && canUseConnection()) {
                console.log('WebSocket连接成功')
                isConnected.value = true
                reconnectCount.value = 0
                imSDK.value?.startHeartbeat()
            }
        })()

        try {
            await connectingPromise.value
        } catch (error) {
            if (!manualDisconnect.value) {
                console.error('WebSocket连接失败:', error)
                isConnected.value = false
                handleReconnectError()
            }
        } finally {
            connectingPromise.value = null
        }
    }

    // 初始化SDK
    const initSDK = () => {
        if (!canUseConnection()) return imSDK.value
        manualDisconnect.value = false

        if (!userStore.token) return null

        if (!imSDK.value) {
            createSDK(userStore.token)
        }

        void connectWithCurrentToken()

        return imSDK.value
    }

    // 添加消息处理器
    const addMessageHandler = (handler: MessageHandler) => {
        if (!imSDK.value) return
        messageHandlers.value.push(handler)
        imSDK.value.onMessage(handler)
    }

    // 移除消息处理器
    const removeMessageHandler = (handler: MessageHandler) => {
        if (!imSDK.value) return
        const index = messageHandlers.value.indexOf(handler)
        if (index > -1) {
            messageHandlers.value.splice(index, 1)
            imSDK.value.offMessage(handler)
        }
    }

    // 添加连接状态处理器
    const addConnectionHandler = (handler: ConnectionHandler) => {
        if (!imSDK.value) return
        connectionHandlers.value.push(handler)
        imSDK.value.onConnection(handler)
    }

    // 移除连接状态处理器
    const removeConnectionHandler = (handler: ConnectionHandler) => {
        if (!imSDK.value) return
        const index = connectionHandlers.value.indexOf(handler)
        if (index > -1) {
            connectionHandlers.value.splice(index, 1)
            imSDK.value.offConnection(handler)
        }
    }

    // 清理所有处理器
    const clearHandlers = () => {
        if (!imSDK.value) return
        messageHandlers.value.forEach(handler => imSDK.value?.offMessage(handler))
        connectionHandlers.value.forEach(handler => imSDK.value?.offConnection(handler))
        messageHandlers.value = []
        connectionHandlers.value = []
    }

    // 手动关闭连接
    const closeConnection = () => {
        console.log('手动关闭连接')
        manualDisconnect.value = true
        
        clearReconnectTimer()
        connectingPromise.value = null

        if (imSDK.value) {
            imSDK.value.stopHeartbeat()
            clearHandlers()
            imSDK.value.disconnect()
            imSDK.value = null
        }
        
        isConnected.value = false
        reconnectCount.value = 0
        sdkToken.value = null
    }

    return {
        imSDK,
        isConnected,
        initSDK,
        addMessageHandler,
        removeMessageHandler,
        addConnectionHandler,
        removeConnectionHandler,
        closeConnection,
        clearHandlers,
        reconnect,
        fetchUnreadCount,
        syncAccessToken
    }
})
