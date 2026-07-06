/**
 * IM API / WebSocket 基址配置。
 *
 * 开发环境默认走 Vite 同源代理（/api/v1 → gateway，/ws → connector），
 * 生产可按需设置完整 URL。
 */
function trimTrailingSlash(value: string): string {
  return value.replace(/\/$/, '')
}

export function getImApiOrigin(): string {
  const base = import.meta.env.VITE_IM_API_BASE ?? '/'
  if (base === '/' || base === '') {
    return typeof window !== 'undefined' ? window.location.origin : ''
  }
  return trimTrailingSlash(base)
}

export function getImWsUrl(): string {
  const wsBase = import.meta.env.VITE_IM_WS_BASE
  if (wsBase) {
    return `${trimTrailingSlash(wsBase.replace(/^http/, 'ws'))}/ws`
  }

  const apiBase = import.meta.env.VITE_IM_API_BASE ?? '/'
  if (apiBase === '/' || apiBase === '') {
    if (typeof window === 'undefined') {
      return 'ws://localhost:5173/ws'
    }
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${proto}//${window.location.host}/ws`
  }

  return `${trimTrailingSlash(apiBase.replace(/^http/, 'ws'))}/ws`
}

export function getImSdkOptions() {
  return {
    baseURL: getImApiOrigin(),
    wsURL: getImWsUrl(),
  }
}

export const config = {
  baseURL: import.meta.env.VITE_IM_API_BASE ?? '/',
  api: {
    users: '/api/v1/users',
    uploads: '/api/v1/uploads/image',
  },
}
