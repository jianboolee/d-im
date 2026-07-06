type PreviewOptions = {
  images: string[]
  startPosition?: number
  showIndex?: boolean
  onClose?: () => void
  onChange?: (index: number) => void
}

let root: HTMLDivElement | null = null
let imageEl: HTMLImageElement | null = null
let indexEl: HTMLDivElement | null = null
let options: PreviewOptions | null = null
let currentIndex = 0

function ensureStyle() {
  if (document.getElementById('app-image-preview-style')) return

  const style = document.createElement('style')
  style.id = 'app-image-preview-style'
  style.textContent = `
    .app-image-preview {
      position: fixed;
      inset: 0;
      z-index: 9998;
      display: grid;
      place-items: center;
      background: rgba(0, 0, 0, 0.92);
      opacity: 0;
      transition: opacity 0.18s ease;
    }

    .app-image-preview.is-visible {
      opacity: 1;
    }

    .app-image-preview__image {
      max-width: 100vw;
      max-height: 100vh;
      object-fit: contain;
      user-select: none;
      -webkit-user-drag: none;
    }

    .app-image-preview__index {
      position: fixed;
      top: max(14px, env(safe-area-inset-top));
      left: 50%;
      transform: translateX(-50%);
      padding: 4px 10px;
      border-radius: 12px;
      background: rgba(0, 0, 0, 0.35);
      color: #fff;
      font-size: 14px;
      line-height: 1.2;
    }

    .app-image-preview__button {
      position: fixed;
      display: grid;
      place-items: center;
      width: 42px;
      height: 42px;
      border: 0;
      border-radius: 50%;
      background: rgba(255, 255, 255, 0.12);
      color: #fff;
      cursor: pointer;
    }

    .app-image-preview__button:active {
      background: rgba(255, 255, 255, 0.2);
    }

    .app-image-preview__close {
      top: max(12px, env(safe-area-inset-top));
      right: 14px;
      font-size: 24px;
    }

    .app-image-preview__prev,
    .app-image-preview__next {
      top: 50%;
      transform: translateY(-50%);
      font-size: 28px;
    }

    .app-image-preview__prev {
      left: 14px;
    }

    .app-image-preview__next {
      right: 14px;
    }

    @media (max-width: 767px) {
      .app-image-preview__prev,
      .app-image-preview__next {
        display: none;
      }
    }
  `
  document.head.appendChild(style)
}

function render() {
  if (!options || !imageEl || !indexEl) return

  const total = options.images.length
  const src = options.images[currentIndex]
  imageEl.src = src ?? ''
  indexEl.textContent = `${currentIndex + 1} / ${total}`
  indexEl.style.display = options.showIndex === false || total <= 1 ? 'none' : 'block'
  options.onChange?.(currentIndex)
}

function move(delta: number) {
  if (!options?.images.length) return
  currentIndex = (currentIndex + delta + options.images.length) % options.images.length
  render()
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') imagePreview.close()
  if (event.key === 'ArrowLeft') move(-1)
  if (event.key === 'ArrowRight') move(1)
}

function createRoot() {
  ensureStyle()

  root = document.createElement('div')
  root.className = 'app-image-preview'

  imageEl = document.createElement('img')
  imageEl.className = 'app-image-preview__image'
  imageEl.alt = ''

  indexEl = document.createElement('div')
  indexEl.className = 'app-image-preview__index'

  const closeButton = document.createElement('button')
  closeButton.className = 'app-image-preview__button app-image-preview__close'
  closeButton.type = 'button'
  closeButton.setAttribute('aria-label', '关闭')
  closeButton.textContent = '×'
  closeButton.addEventListener('click', () => imagePreview.close())

  const prevButton = document.createElement('button')
  prevButton.className = 'app-image-preview__button app-image-preview__prev'
  prevButton.type = 'button'
  prevButton.setAttribute('aria-label', '上一张')
  prevButton.textContent = '‹'
  prevButton.addEventListener('click', (event) => {
    event.stopPropagation()
    move(-1)
  })

  const nextButton = document.createElement('button')
  nextButton.className = 'app-image-preview__button app-image-preview__next'
  nextButton.type = 'button'
  nextButton.setAttribute('aria-label', '下一张')
  nextButton.textContent = '›'
  nextButton.addEventListener('click', (event) => {
    event.stopPropagation()
    move(1)
  })

  root.addEventListener('click', (event) => {
    if (event.target === root) imagePreview.close()
  })

  root.append(imageEl, indexEl, closeButton, prevButton, nextButton)
  document.body.appendChild(root)
  document.addEventListener('keydown', handleKeydown)

  window.requestAnimationFrame(() => {
    root?.classList.add('is-visible')
  })
}

export const imagePreview = {
  preview(nextOptions: PreviewOptions) {
    if (typeof document === 'undefined' || nextOptions.images.length === 0) return null

    imagePreview.close()
    options = nextOptions
    currentIndex = Math.min(
      Math.max(nextOptions.startPosition ?? 0, 0),
      nextOptions.images.length - 1,
    )
    createRoot()
    render()

    return imagePreview
  },

  close() {
    if (!root) return

    options?.onClose?.()
    document.removeEventListener('keydown', handleKeydown)
    root.remove()
    root = null
    imageEl = null
    indexEl = null
    options = null
    currentIndex = 0
  },

  swipeTo(index: number) {
    if (!options?.images.length) return

    currentIndex = Math.min(Math.max(index, 0), options.images.length - 1)
    render()
  },
}

export default {
  install(app: { config: { globalProperties: Record<string, unknown> } }) {
    app.config.globalProperties.$imagePreview = imagePreview
  },
}

