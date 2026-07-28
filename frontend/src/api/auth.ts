import { apiRequest } from '@/api/client'

export interface SessionUser {
  id: string
  email: string
  full_name: string
  role: 'admin' | 'warehouse_manager' | 'picker' | 'viewer'
  warehouse_id?: string
}

export interface TokenPair {
  access_token: string
  refresh_token: string
  token_type: 'Bearer'
  access_token_expires_at: string
  refresh_token_expires_at: string
  user: SessionUser
}

export function login(email: string, password: string) {
  return apiRequest<TokenPair>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })
}

export function refresh(refreshToken: string) {
  return apiRequest<TokenPair>('/auth/refresh', {
    method: 'POST',
    body: JSON.stringify({ refresh_token: refreshToken }),
  })
}

export function logout(refreshToken: string) {
  return apiRequest<void>('/auth/logout', {
    method: 'POST',
    body: JSON.stringify({ refresh_token: refreshToken }),
  })
}

export function getCurrentUser() {
  return apiRequest<SessionUser>('/auth/me')
}
