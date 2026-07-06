import { useUserStore } from '@/stores/user'
import { config } from '@/config'
import axios from '@/plugins/axios'

interface ApiError {
  response?: {
    status: number
  }
}

interface RequestOptions {
  method?: string
  body?: unknown
  headers?: Record<string, string>
  params?: Record<string, string>
  timeout?: number
  skipAuthRetry?: boolean
}

export const request = async <T>(url: string, options: RequestOptions = {}): Promise<T> => {
  const userStore = useUserStore()
  const token = options.skipAuthRetry
    ? userStore.token
    : await userStore.ensureValidToken({ logoutOnAuthError: true })

  const headers: Record<string, string> = {
    ...(!(options.body instanceof FormData) && { 'Content-Type': 'application/json' }),
    ...options.headers,
  }

  if (token) {
    headers.Authorization = `Bearer ${token}`
  }

  try {
    const response = await axios.request<T>({
      url,
      baseURL: config.baseURL,
      method: options.method || 'GET',
      headers,
      data: options.body,
      params: options.params,
      timeout: options.timeout,
    })
    return response.data
  } catch (error) {
    const status = (error as ApiError).response?.status
    if (status === 401 && !options.skipAuthRetry) {
      const newToken = await userStore.ensureValidToken({
        force: true,
        logoutOnAuthError: true,
      })
      if (newToken) {
        return request<T>(url, { ...options, skipAuthRetry: true })
      }
    }
    throw error
  }
}
