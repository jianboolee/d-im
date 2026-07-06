export const SUCCESS_CODE = 0

export interface ApiResponse<T> {
  code: number
  data: T
  message?: string
}

export function isSuccessResponse<T>(response: ApiResponse<T>): boolean {
  return response.code === SUCCESS_CODE
}
