import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

const OWNER_KEY = 'd-im-active-tab'
const HEARTBEAT_INTERVAL = 2000
const OWNER_TIMEOUT = 6000

interface TabOwner {
  tabId: string
  userId: string
  updatedAt: number
}

type TabMessage =
  | { type: 'takeover'; owner: TabOwner }
  | { type: 'release'; tabId: string; userId: string }

const createTabId = () => {
  const existing = sessionStorage.getItem('d-im-tab-id')
  if (existing) return existing

  const id =
    typeof crypto !== 'undefined' && 'randomUUID' in crypto
      ? crypto.randomUUID()
      : `${Date.now()}-${Math.random().toString(36).slice(2)}`
  sessionStorage.setItem('d-im-tab-id', id)
  return id
}

const readOwner = (): TabOwner | null => {
  const raw = localStorage.getItem(OWNER_KEY)
  if (!raw) return null

  try {
    return JSON.parse(raw) as TabOwner
  } catch {
    return null
  }
}

const isOwnerAlive = (owner: TabOwner | null) =>
  Boolean(owner && Date.now() - owner.updatedAt < OWNER_TIMEOUT)

export const useIMTabStore = defineStore('imTab', () => {
  const tabId = createTabId()
  const userId = ref('')
  const owner = ref<TabOwner | null>(readOwner())
  const initialized = ref(false)
  const channel = ref<BroadcastChannel | null>(null)
  const heartbeatTimer = ref<ReturnType<typeof setInterval> | null>(null)
  const staleCheckTimer = ref<ReturnType<typeof setInterval> | null>(null)

  const isPrimaryTab = computed(
    () => Boolean(userId.value && owner.value?.userId === userId.value && owner.value?.tabId === tabId),
  )
  const isSuspended = computed(() => initialized.value && userId.value !== '' && !isPrimaryTab.value)

  const writeOwner = (nextOwner: TabOwner) => {
    owner.value = nextOwner
    localStorage.setItem(OWNER_KEY, JSON.stringify(nextOwner))
  }

  const stopHeartbeat = () => {
    if (heartbeatTimer.value) {
      clearInterval(heartbeatTimer.value)
      heartbeatTimer.value = null
    }
  }

  const startHeartbeat = () => {
    stopHeartbeat()
    heartbeatTimer.value = setInterval(() => {
      if (!isPrimaryTab.value || !userId.value) return
      writeOwner({
        tabId,
        userId: userId.value,
        updatedAt: Date.now(),
      })
    }, HEARTBEAT_INTERVAL)
  }

  const becomeSecondary = (nextOwner: TabOwner) => {
    owner.value = nextOwner
    stopHeartbeat()
  }

  const claimActive = () => {
    if (!userId.value) return

    const nextOwner: TabOwner = {
      tabId,
      userId: userId.value,
      updatedAt: Date.now(),
    }
    writeOwner(nextOwner)
    startHeartbeat()
    channel.value?.postMessage({ type: 'takeover', owner: nextOwner } satisfies TabMessage)
  }

  const handleChannelMessage = (event: MessageEvent<TabMessage>) => {
    const message = event.data
    if (!message || !userId.value) return

    if (message.type === 'takeover' && message.owner.userId === userId.value) {
      if (message.owner.tabId !== tabId) {
        becomeSecondary(message.owner)
      }
      return
    }

    if (message.type === 'release' && message.userId === userId.value && message.tabId === owner.value?.tabId) {
      owner.value = null
    }
  }

  const handleStorage = (event: StorageEvent) => {
    if (event.key !== OWNER_KEY) return
    owner.value = readOwner()
    if (owner.value?.userId === userId.value && owner.value.tabId !== tabId) {
      stopHeartbeat()
    }
  }

  const startStaleCheck = () => {
    if (staleCheckTimer.value) return
    staleCheckTimer.value = setInterval(() => {
      if (!initialized.value || !userId.value || isPrimaryTab.value) return
      const currentOwner = readOwner()
      owner.value = currentOwner
      if (!isOwnerAlive(currentOwner) || currentOwner?.userId !== userId.value) {
        claimActive()
      }
    }, HEARTBEAT_INTERVAL)
  }

  const init = (nextUserId: string) => {
    if (!nextUserId) return

    if (!channel.value) {
      channel.value = new BroadcastChannel('d-im-tab-coordinator')
      channel.value.addEventListener('message', handleChannelMessage)
      window.addEventListener('storage', handleStorage)
      window.addEventListener('beforeunload', release)
    }

    userId.value = nextUserId
    initialized.value = true
    startStaleCheck()
    claimActive()
  }

  function release() {
    if (isPrimaryTab.value && userId.value) {
      localStorage.removeItem(OWNER_KEY)
      channel.value?.postMessage({ type: 'release', tabId, userId: userId.value } satisfies TabMessage)
    }
    stopHeartbeat()
  }

  const reset = () => {
    release()
    initialized.value = false
    userId.value = ''
    owner.value = null
  }

  return {
    tabId,
    owner,
    initialized,
    isPrimaryTab,
    isSuspended,
    init,
    claimActive,
    reset,
  }
})
