import { defineStore } from 'pinia'
import { ref } from 'vue'

const DEFAULT_TITLE = '消息'
const NOTICE_TITLE = '【有新消息】'
const BLINK_INTERVAL = 1000
const MAX_BLINK_FRAMES = 8

export const usePageNotificationStore = defineStore('pageNotification', () => {
  const baseTitle = ref(DEFAULT_TITLE)
  const hasUnreadNotice = ref(false)
  const blinkOn = ref(false)
  const mounted = ref(false)
  let blinkTimer: ReturnType<typeof setInterval> | null = null
  let blinkFrames = 0
  let faviconEl: HTMLLinkElement | null = null
  let originalFaviconHref = ''
  let noticeFaviconHref = ''

  function getFaviconElement() {
    if (typeof document === 'undefined') return null
    const existing = document.querySelector<HTMLLinkElement>('link[rel~="icon"]')
    if (existing) return existing

    const created = document.createElement('link')
    created.rel = 'icon'
    created.href = '/favicon.ico'
    document.head.appendChild(created)
    return created
  }

  function createNoticeFavicon(baseHref: string) {
    if (typeof document === 'undefined') return ''

    const canvas = document.createElement('canvas')
    const size = 32
    canvas.width = size
    canvas.height = size
    const ctx = canvas.getContext('2d')
    if (!ctx) return ''

    return new Promise<string>((resolve) => {
      const finish = () => {
        ctx.beginPath()
        ctx.arc(24, 8, 7, 0, Math.PI * 2)
        ctx.fillStyle = '#f43f5e'
        ctx.fill()
        ctx.lineWidth = 2
        ctx.strokeStyle = '#ffffff'
        ctx.stroke()
        resolve(canvas.toDataURL('image/png'))
      }

      if (!baseHref) {
        ctx.fillStyle = '#4b86f8'
        ctx.fillRect(5, 5, 22, 22)
        finish()
        return
      }

      const image = new Image()
      image.crossOrigin = 'anonymous'
      image.onload = () => {
        ctx.drawImage(image, 0, 0, size, size)
        finish()
      }
      image.onerror = () => {
        ctx.fillStyle = '#4b86f8'
        ctx.fillRect(5, 5, 22, 22)
        finish()
      }
      image.src = baseHref
    })
  }

  async function ensureNoticeFavicon() {
    faviconEl = getFaviconElement()
    if (!faviconEl) return

    if (!originalFaviconHref) {
      originalFaviconHref = faviconEl.href
    }
    if (!noticeFaviconHref) {
      noticeFaviconHref = await createNoticeFavicon(originalFaviconHref)
    }
  }

  function applyTitle() {
    if (typeof document === 'undefined') return
    document.title = hasUnreadNotice.value && blinkOn.value ? NOTICE_TITLE : baseTitle.value
  }

  function applyFavicon() {
    if (!faviconEl || !originalFaviconHref || !noticeFaviconHref) return
    faviconEl.href = hasUnreadNotice.value && blinkOn.value ? noticeFaviconHref : originalFaviconHref
  }

  function applyNotificationFrame() {
    applyTitle()
    applyFavicon()
  }

  function stopBlink() {
    if (blinkTimer) {
      clearInterval(blinkTimer)
      blinkTimer = null
    }
    blinkFrames = 0
    blinkOn.value = false
    applyFavicon()
  }

  function holdNoticeFrame() {
    if (blinkTimer) {
      clearInterval(blinkTimer)
      blinkTimer = null
    }
    blinkFrames = 0
    blinkOn.value = true
    applyNotificationFrame()
  }

  async function startBlink() {
    if (blinkTimer) return
    await ensureNoticeFavicon()
    blinkFrames = 0
    blinkOn.value = true
    applyNotificationFrame()
    blinkTimer = setInterval(() => {
      blinkFrames += 1
      if (blinkFrames >= MAX_BLINK_FRAMES) {
        holdNoticeFrame()
        return
      }
      blinkOn.value = !blinkOn.value
      applyNotificationFrame()
    }, BLINK_INTERVAL)
  }

  function setBaseTitle(title: string) {
    baseTitle.value = title.trim() || DEFAULT_TITLE
    applyTitle()
  }

  function notifyNewMessage() {
    if (typeof document === 'undefined') return
    if (document.visibilityState === 'visible') return

    hasUnreadNotice.value = true
    void startBlink()
  }

  function clearNotice() {
    hasUnreadNotice.value = false
    stopBlink()
    applyNotificationFrame()
  }

  function handleVisibilityChange() {
    if (document.visibilityState === 'visible') {
      clearNotice()
    }
  }

  function mount() {
    if (mounted.value || typeof document === 'undefined') return

    mounted.value = true
    document.addEventListener('visibilitychange', handleVisibilityChange)
    window.addEventListener('focus', clearNotice)
    applyTitle()
  }

  function unmount() {
    if (!mounted.value || typeof document === 'undefined') return

    mounted.value = false
    document.removeEventListener('visibilitychange', handleVisibilityChange)
    window.removeEventListener('focus', clearNotice)
    stopBlink()
    applyNotificationFrame()
  }

  return {
    baseTitle,
    hasUnreadNotice,
    setBaseTitle,
    notifyNewMessage,
    clearNotice,
    mount,
    unmount,
  }
})
