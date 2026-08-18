import { apiRequest } from '@/api/client'
import type { Page } from '@/types/pagination'

export interface Warehouse {
  id: string
  code: string
  name: string
  address: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface WarehouseInput {
  code: string
  name: string
  address?: string | null
  is_active?: boolean
}

export interface WarehouseListFilter {
  limit?: number
  after?: string
  search?: string
  is_active?: boolean
}

function buildQuery(filter: object): string {
  const params = new URLSearchParams()
  const entries = Object.entries(
    filter as Record<string, string | number | boolean | undefined>,
  )
  for (const [key, value] of entries) {
    if (value !== undefined) {
      params.set(key, String(value))
    }
  }
  const query = params.toString()
  return query ? `?${query}` : ''
}

export function listWarehouses(filter: WarehouseListFilter = {}) {
  return apiRequest<Page<Warehouse>>(`/warehouses${buildQuery(filter)}`)
}

export function getWarehouse(id: string) {
  return apiRequest<Warehouse>(`/warehouses/${id}`)
}

export function createWarehouse(input: WarehouseInput) {
  return apiRequest<Warehouse>('/warehouses', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateWarehouse(id: string, input: WarehouseInput) {
  return apiRequest<Warehouse>(`/warehouses/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export function deactivateWarehouse(id: string) {
  return apiRequest<void>(`/warehouses/${id}`, { method: 'DELETE' })
}
