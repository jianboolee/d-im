/**
 * IM API / WebSocket 基址配置。
 *
 * 开发环境直接指向后端：API → localhost:8080，WebSocket → localhost:8081
 * 生产环境通过 VITE_IM_API_BASE 环境变量覆盖
 */
function apiOrigin(): string {
  if (import.meta.env.VITE_IM_API_BASE) {
    return import.meta.env.VITE_IM_API_BASE.replace(/\/$/, '')
  }
  return 'http://localhost:8080'
}

function wsOrigin(): string {
  if (import.meta.env.VITE_IM_WS_BASE) {
    return import.meta.env.VITE_IM_WS_BASE.replace(/\/$/, '')
  }
  return 'ws://localhost:8081'
}

export function getImSdkOptions() {
  return {
    baseURL: apiOrigin(),
    wsURL: `${wsOrigin()}/ws`,
  }
}

export const config = {
  apiOrigin,
  wsOrigin,
}