import { getImSdkOptions } from '@/config'

export interface TokenPair {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
}

export interface LoginParams {
  uid: string
  device_id: string
}

export class AuthActionError extends Error {
  reason: 'auth' | 'network' | 'server'
  status?: number

  constructor(reason: 'auth' | 'network' | 'server', message: string, status?: number) {
    super(message)
    this.name = 'AuthActionError'
    this.reason = reason
    this.status = status
  }
}

async function request<T>(path: string, body: unknown, token?: string): Promise<T> {
  const { baseURL } = getImSdkOptions()
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const resp = await fetch(`${baseURL}${path}`, {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  })

  const data = await resp.json()
  if (!resp.ok) {
    const msg = data?.error || `HTTP ${resp.status}`
    if (resp.status === 401 || resp.status === 403) {
      throw new AuthActionError('auth', msg, resp.status)
    }
    throw new AuthActionError('server', msg, resp.status)
  }
  return data as T
}

/** 开发模式登录 */
export async function devLogin(params: LoginParams): Promise<TokenPair> {
  return request<TokenPair>('/api/v1/auth/login', params)
}

/** 刷新 access_token */
export async function refreshAccessToken(refreshToken: string): Promise<TokenPair> {
  return request<TokenPair>('/api/v1/auth/refresh', {}, refreshToken)
}

/** 登出（非必需，前端清除即可） */
export async function logoutSession(refreshToken?: string): Promise<void> {
  try {
    await request('/api/v1/auth/logout', {}, refreshToken)
  } catch {
    // 忽略错误
  }
}

/** ticket 换取 token（SSO 流程，稍后接入） */
export async function exchangeAccessToken(ticket: string, deviceID: string): Promise<TokenPair> {
  return request<TokenPair>('/api/v1/auth/token', { ticket, device_id: deviceID })
}