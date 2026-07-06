type ToastOptions = {
  duration?: number
}

let toastElement: HTMLDivElement | null = null
let toastTimer: ReturnType<typeof window.setTimeout> | null = null

function ensureToastElement() {
  if (toastElement) return toastElement

  toastElement = document.createElement('div')
  toastElement.className = 'app-toast'
  toastElement.setAttribute('role', 'status')
  toastElement.setAttribute('aria-live', 'polite')
  document.body.appendChild(toastElement)

  if (!document.getElementById('app-toast-style')) {
    const style = document.createElement('style')
    style.id = 'app-toast-style'
    style.textContent = `
      .app-toast {
        position: fixed;
        left: 50%;
        top: 50%;
        z-index: 9999;
        max-width: min(280px, calc(100vw - 72px));
        min-width: 112px;
        padding: 13px 16px;
        border-radius: 10px;
        background: rgba(0, 0, 0, 0.72);
        color: #fff;
        font-size: 14px;
        line-height: 1.5;
        text-align: center;
        pointer-events: none;
        opacity: 0;
        transform: translate(-50%, -50%) scale(0.96);
        transition: opacity 0.18s ease, transform 0.18s ease;
        word-break: break-word;
        box-shadow: 0 8px 28px rgba(0, 0, 0, 0.16);
      }

      .app-toast.is-visible {
        opacity: 1;
        transform: translate(-50%, -50%) scale(1);
      }
    `
    document.head.appendChild(style)
  }

  return toastElement
}

export function showToast(message: string, options: ToastOptions = {}) {
  if (typeof document === 'undefined') return

  const element = ensureToastElement()
  element.textContent = message

  if (toastTimer) {
    window.clearTimeout(toastTimer)
    toastTimer = null
  }

  window.requestAnimationFrame(() => {
    element.classList.add('is-visible')
  })

  toastTimer = window.setTimeout(() => {
    element.classList.remove('is-visible')
  }, options.duration ?? 2000)
}
