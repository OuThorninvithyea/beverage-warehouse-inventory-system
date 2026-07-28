export interface ApiSuccess<T> {
  success: true
  data: T
}

export interface ApiErrorDetail {
  code: string
  message: string
}

export interface ApiFailure {
  success: false
  error: ApiErrorDetail
}

export type ApiResponse<T> = ApiSuccess<T> | ApiFailure
