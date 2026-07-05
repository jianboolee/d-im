const MAX_REFRESH_THRESHOLD_MS = 5 * 60 * 1000
const MIN_SCHEDULE_DELAY_MS = 10_000

interface JwtPayload {
  exp?: number
  iat?: number
  sub?: string
}

function decodeBase64Url(value: string): string {
  const normalized = value.replace(/-/g, '+').replace(/_/g, '/')
  const padding = normalized.length % 4 === 0 ? '' : '='.repeat(4 - (normalized.length % 4))
  return atob(normalized + padding)
}

export function parseJwtPayload(token: string): JwtPayload | null {
  const parts = token.split('.')
  if (parts.length !== 3) {
    return null
  }

  try {
    const payload = JSON.parse(decodeBase64Url(parts[1]!)) as JwtPayload
    return payload
  } catch {
    return null
  }
}

export function getTokenExpiryMs(token: string): number | null {
  const payload = parseJwtPayload(token)
  if (typeof payload?.exp !== 'number') {
    return null
  }

  return payload.exp * 1000
}

/** 在 token 剩余寿命的后半段（最多提前 5 分钟）触发续期 */
export function getRefreshThresholdMs(remainingMs: number): number {
  if (remainingMs <= 0) {
    return 0
  }

  const proportional = Math.floor(remainingMs / 2)
  return Math.min(MAX_REFRESH_THRESHOLD_MS, proportional)
}

export function isTokenExpiringSoon(token: string): boolean {
  const expiryMs = getTokenExpiryMs(token)
  if (expiryMs == null) {
    return true
  }

  const remainingMs = expiryMs - Date.now()
  if (remainingMs <= 0) {
    return true
  }

  return remainingMs <= getRefreshThresholdMs(remainingMs)
}

/** 距离下次应触发静默续期的毫秒数 */
export function getRefreshScheduleDelayMs(token: string): number | null {
  const expiryMs = getTokenExpiryMs(token)
  if (expiryMs == null) {
    return null
  }

  const remainingMs = expiryMs - Date.now()
  if (remainingMs <= 0) {
    return 0
  }

  const threshold = getRefreshThresholdMs(remainingMs)
  if (remainingMs <= threshold) {
    return MIN_SCHEDULE_DELAY_MS
  }

  return Math.max(remainingMs - threshold, MIN_SCHEDULE_DELAY_MS)
}

export { MAX_REFRESH_THRESHOLD_MS as REFRESH_THRESHOLD_MS }
