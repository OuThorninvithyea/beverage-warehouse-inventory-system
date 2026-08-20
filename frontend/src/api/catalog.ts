import { apiRequest } from '@/api/client'
import type { Page } from '@/types/pagination'

export interface Category {
  id: string
  parent_id: string | null
  name: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CategoryInput {
  name: string
  parent_id?: string | null
  is_active?: boolean
}

export interface CategoryListFilter {
  limit?: number
  after?: string
  search?: string
  parent_id?: string
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

export function listCategories(filter: CategoryListFilter = {}) {
  return apiRequest<Page<Category>>(`/categories${buildQuery(filter)}`)
}

export function getCategory(id: string) {
  return apiRequest<Category>(`/categories/${id}`)
}

export function createCategory(input: CategoryInput) {
  return apiRequest<Category>('/categories', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateCategory(id: string, input: CategoryInput) {
  return apiRequest<Category>(`/categories/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export function deactivateCategory(id: string) {
  return apiRequest<void>(`/categories/${id}`, { method: 'DELETE' })
}

export interface Product {
  id: string
  category_id: string | null
  sku: string
  barcode: string | null
  name: string
  unit: string
  is_lot_tracked: boolean
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface ProductInput {
  category_id?: string | null
  sku: string
  barcode?: string | null
  name: string
  unit: string
  is_lot_tracked?: boolean
  is_active?: boolean
}

export interface ProductListFilter {
  limit?: number
  after?: string
  search?: string
  category_id?: string
  is_active?: boolean
}

export function listProducts(filter: ProductListFilter = {}) {
  return apiRequest<Page<Product>>(`/products${buildQuery(filter)}`)
}

export function getProduct(id: string) {
  return apiRequest<Product>(`/products/${id}`)
}

export function createProduct(input: ProductInput) {
  return apiRequest<Product>('/products', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateProduct(id: string, input: ProductInput) {
  return apiRequest<Product>(`/products/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export function deactivateProduct(id: string) {
  return apiRequest<void>(`/products/${id}`, { method: 'DELETE' })
}
