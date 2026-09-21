import { apiRequest } from '@/api/client'
import type { Page } from '@/types/pagination'

export interface StockBalance {
  id: string
  location_id: string
  product_id: string
  lot_id: string | null
  quantity: string
  reserved_quantity: string
  available_quantity: string
  created_at: string
  updated_at: string
}

export interface BalanceListFilter {
  limit?: number
  after?: string
  warehouse_id?: string
  location_id?: string
  product_id?: string
  lot_id?: string
}

export interface Lot {
  id: string
  product_id: string
  lot_number: string
  expiration_date: string | null
  created_at: string
  available_quantity?: string
}

export type ExpiryStatus = 'expired' | 'expiring'

export interface ExpiryAlert {
  product_id: string
  sku: string
  product_name: string
  lot_id: string
  lot_number: string
  expiration_date: string
  days_remaining: number
  status: ExpiryStatus
  warehouse_id: string
  warehouse_code: string
  location_id: string
  location_code: string
  quantity: string
  reserved_quantity: string
  available_quantity: string
}

export interface ExpiryAlertFilter {
  within_days?: number
  warehouse_id?: string
  limit?: number
}

export interface ReceiveInput {
  location_id: string
  product_id: string
  quantity: string
  unit_cost: string
  lot_number?: string | null
  expiration_date?: string | null
  reference?: string | null
  notes?: string | null
}

export interface PickInput {
  location_id: string
  product_id: string
  quantity: string
  lot_id?: string | null
  reference?: string | null
  notes?: string | null
}

export interface TransferInput {
  product_id: string
  lot_id?: string | null
  quantity: string
  from_location_id: string
  to_location_id: string
  reference?: string | null
  notes?: string | null
}

export interface AdjustInput {
  location_id: string
  product_id: string
  lot_id?: string | null
  direction: 'increase' | 'decrease'
  quantity: string
  notes: string
}

export interface StockMovement {
  id: string
  movement_type: 'receive' | 'pick' | 'transfer' | 'adjust'
  product_id: string
  lot_id: string | null
  from_location_id: string | null
  to_location_id: string | null
  quantity: string
  unit_cost: string | null
  reference: string | null
  notes: string | null
  performed_by: string | null
  created_at: string
}

export interface MovementListFilter {
  limit?: number
  after?: string
  product_id?: string
  location_id?: string
  movement_type?: 'receive' | 'pick' | 'transfer' | 'adjust'
  from?: string
  to?: string
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

export function listBalances(filter: BalanceListFilter = {}) {
  return apiRequest<Page<StockBalance>>(`/inventory${buildQuery(filter)}`)
}

export function listLots(productId: string) {
  return apiRequest<Lot[]>(`/inventory/products/${productId}/lots`)
}

export function listExpiryAlerts(filter: ExpiryAlertFilter = {}) {
  return apiRequest<ExpiryAlert[]>(`/inventory/alerts${buildQuery(filter)}`)
}

export function receiveStock(input: ReceiveInput) {
  return apiRequest<{ movement: StockMovement; balance: StockBalance }>(
    '/inventory/movements/receive',
    {
      method: 'POST',
      body: JSON.stringify(input),
    },
  )
}

export function pickStock(input: PickInput) {
  return apiRequest<{ movements: StockMovement[]; total_quantity: string }>(
    '/inventory/movements/pick',
    {
      method: 'POST',
      body: JSON.stringify(input),
    },
  )
}

export function transferStock(input: TransferInput) {
  return apiRequest<{
    movement: StockMovement
    source_balance: StockBalance
    destination_balance: StockBalance
  }>('/inventory/movements/transfer', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function adjustStock(input: AdjustInput) {
  return apiRequest<{ movement: StockMovement; balance: StockBalance }>(
    '/inventory/movements/adjust',
    {
      method: 'POST',
      body: JSON.stringify(input),
    },
  )
}

export function listMovements(filter: MovementListFilter = {}) {
  return apiRequest<Page<StockMovement>>(
    `/inventory/movements${buildQuery(filter)}`,
  )
}

export function getMovement(movementId: string) {
  return apiRequest<StockMovement>(`/inventory/movements/${movementId}`)
}
