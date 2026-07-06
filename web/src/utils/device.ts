const DEVICE_ID_KEY = 'd-im-device-id'

export interface DeviceMeta {
  platform: string
  device_id: string
  device_name?: string
  app_version?: string
  push_token?: string
}

function createDeviceId() {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`
}

export function getDeviceId() {
  if (typeof localStorage === 'undefined') {
    return createDeviceId()
  }

  const existing = localStorage.getItem(DEVICE_ID_KEY)
  if (existing) {
    return existing
  }

  const nextID = createDeviceId()
  localStorage.setItem(DEVICE_ID_KEY, nextID)
  return nextID
}

export function getDeviceMeta(): DeviceMeta {
  const userAgent = typeof navigator !== 'undefined' ? navigator.userAgent : ''
  const platform = typeof navigator !== 'undefined' ? navigator.platform : ''

  return {
    platform: 'web',
    device_id: getDeviceId(),
    device_name: platform || userAgent || 'Web',
    app_version: import.meta.env.VITE_APP_VERSION || '',
  }
}
