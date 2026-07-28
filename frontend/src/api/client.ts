import type { ApiFailure, ApiResponse } from '@/types/api'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'
let accessToken = ''

export function setApiAccessToken(token: string) {
  accessToken = token
}

export class ApiClientError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiClientError'
    this.status = status
    this.code = code
  }
}

export async function apiRequest<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers: {
      Accept: 'application/json',
      ...(options.body ? { 'Content-Type': 'application/json' } : {}),
      ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
      ...options.headers,
    },
  })

  if (response.status === 204) {
    return undefined as T
  }

  const payload = (await response.json()) as ApiResponse<T>

  if (!response.ok || !payload.success) {
    const failure = payload as ApiFailure
    throw new ApiClientError(
      response.status,
      failure.error?.code ?? 'REQUEST_FAILED',
      failure.error?.message ?? 'The request could not be completed.',
    )
  }

  return payload.data
}
