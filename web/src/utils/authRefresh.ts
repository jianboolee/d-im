import { config } from '@/config'
import { SUCCESS_CODE, type ApiResponse } from '@/types/api'
import axios from '@/plugins/axios'
import { isAxiosError } from 'axios'
import { getDeviceMeta } from '@/utils/device'

export interface AccessTokenData {
  access_token: string
  expires_in: number
  refresh_token: string
  token_type?: string
}

export type AuthActionErrorReason = 'auth' | 'network' | 'server'

export class AuthActionError extends Error {
  reason: AuthActionErrorReason
  status?: number

  constructor(reason: AuthActionErrorReason, message: string, status?: number) {
    super(message)
    this.name = 'AuthActionError'
    this.reason = reason
    this.status = status
  }
}

function normalizeAuthError(error: unknown, fallbackMessage: string): never {
  const status = isAxiosError(error) ? error.response?.status : undefined
  if (status === 401 || status === 403) {
    throw new AuthActionError('auth', '登录态已失效', status)
  }
  if (status != null) {
    throw new AuthActionError('server', fallbackMessage, status)
  }
  throw new AuthActionError('network', fallbackMessage)
}

function getAPIErrorMessage(response: ApiResponse<unknown> & { error?: string }, fallback: string) {
  return response.error || response.message || fallback
}

export async function loginWithPassword(id: string, password: string): Promise<AccessTokenData> {
  try {
    const device = getDeviceMeta()
    const response = await axios.post<ApiResponse<AccessTokenData>>(
      '/api/v1/auth/login',
      {
        id,
        password,
        device_id: device.device_id,
      },
      {
        baseURL: config.baseURL,
      },
    )

    if (response.data.code === SUCCESS_CODE && response.data.data?.access_token) {
      return response.data.data
    }

    throw new AuthActionError('server', getAPIErrorMessage(response.data, '登录失败'))
  } catch (error) {
    if (error instanceof AuthActionError) {
      throw error
    }
    normalizeAuthError(error, '登录失败')
  }
}

export async function exchangeAccessToken(ticket: string): Promise<AccessTokenData> {
  try {
    const device = getDeviceMeta()
    const response = await axios.post<ApiResponse<AccessTokenData>>(
      '/api/v1/auth/ticket',
      {
        ticket,
        device_id: device.device_id,
      },
      {
        baseURL: config.baseURL,
      },
    )

    if (response.data.code === SUCCESS_CODE && response.data.data?.access_token) {
      return response.data.data
    }

    throw new AuthActionError('server', getAPIErrorMessage(response.data, '建立登录会话失败'))
  } catch (error) {
    if (error instanceof AuthActionError) {
      throw error
    }
    normalizeAuthError(error, '建立登录会话失败')
  }
}

export async function refreshAccessToken(refreshToken: string): Promise<AccessTokenData> {
  try {
    const response = await axios.post<ApiResponse<AccessTokenData>>(
      '/api/v1/auth/refresh',
      {},
      {
        baseURL: config.baseURL,
        headers: {
          Authorization: `Bearer ${refreshToken}`,
        },
      },
    )

    if (response.data.code === SUCCESS_CODE && response.data.data?.access_token) {
      return response.data.data
    }

    throw new AuthActionError('server', getAPIErrorMessage(response.data, '刷新登录态失败'))
  } catch (error) {
    if (error instanceof AuthActionError) {
      throw error
    }
    normalizeAuthError(error, '刷新登录态失败')
  }
}

export async function logoutSession(refreshToken?: string | null): Promise<void> {
  try {
    await axios.post(
      '/api/v1/auth/logout',
      {},
      {
        baseURL: config.baseURL,
        headers: refreshToken
          ? {
              Authorization: `Bearer ${refreshToken}`,
            }
          : undefined,
      },
    )
  } catch (error) {
    if (isAxiosError(error) && error.response?.status && error.response.status < 500) {
      return
    }
    throw error
  }
}
