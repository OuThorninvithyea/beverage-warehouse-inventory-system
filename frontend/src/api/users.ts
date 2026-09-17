import { apiRequest } from '@/api/client'
import type { Page } from '@/types/pagination'

export type UserRole = 'admin' | 'warehouse_manager' | 'picker' | 'viewer'

export interface User {
  id: string
  email: string
  full_name: string
  role: UserRole
  warehouse_id: string | null
  warehouse_name?: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CreateUserInput {
  email: string
  full_name: string
  role: UserRole
  warehouse_id?: string | null
  password: string
}

export interface UpdateUserInput {
  full_name: string
  role: UserRole
  warehouse_id?: string | null
  is_active?: boolean
}

export interface UserListFilter {
  limit?: number
  after?: string
  search?: string
  role?: UserRole
  warehouse_id?: string
  is_active?: boolean
}

function buildQuery(filter: object): string {
  const params = new URLSearchParams()
  const entries = Object.entries(
    filter as Record<string, string | number | boolean | undefined>,
  )
  for (const [key, value] of entries) {
    if (value !== undefined && value !== null && value !== '') {
      params.set(key, String(value))
    }
  }
  const query = params.toString()
  return query ? `?${query}` : ''
}

export function listUsers(filter: UserListFilter = {}) {
  return apiRequest<Page<User>>(`/users${buildQuery(filter)}`)
}

export function getUser(id: string) {
  return apiRequest<User>(`/users/${id}`)
}

export function createUser(input: CreateUserInput) {
  return apiRequest<User>('/users', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateUser(id: string, input: UpdateUserInput) {
  return apiRequest<User>(`/users/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export function deactivateUser(id: string) {
  return apiRequest<void>(`/users/${id}`, { method: 'DELETE' })
}

export function resetPassword(id: string, password: string) {
  return apiRequest<void>(`/users/${id}/password-reset`, {
    method: 'POST',
    body: JSON.stringify({ password }),
  })
}
