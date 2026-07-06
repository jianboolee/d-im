import { config } from '@/config'
import type { ApiResponse } from '@/types/api'
import { request } from './request'

export interface UploadedFile {
  url: string
  filename: string
  size: number
  width?: number
  height?: number
  format?: string
}

export const UPLOAD_TIMEOUT_MS = 60000

/** 上传单个文件到 IM 服务 */
export async function uploadIMFile(file: File): Promise<UploadedFile> {
  const formData = new FormData()
  formData.append('file', file)

  const response = await request<unknown>(config.api.uploads, {
    method: 'POST',
    body: formData,
    headers: {},
    timeout: UPLOAD_TIMEOUT_MS,
  })

  const data = (
    response && typeof response === 'object' && 'code' in response
      ? (response as ApiResponse<UploadedFile>).data
      : response
  ) as UploadedFile

  if (!data?.url) {
    throw new Error('上传失败')
  }

  return data
}
