import { apiRequest } from '@/api/client'

export type MovementType = 'receive' | 'pick' | 'transfer' | 'adjust'

export interface DashboardReport {
  generated_at: string
  active_products: number
  active_warehouses: number
  active_locations: number
  total_quantity: string
  reserved_quantity: string
  available_quantity: string
  stock_value: string
  lots_on_hand: number
  expired_lots: number
  expiring_soon_lots: number
  movements_by_type: Record<MovementType, number>
  movement_window_days: number
}

export interface ValuationRow {
  warehouse_id: string
  warehouse_code: string
  product_id: string
  sku: string
  product_name: string
  remaining_quantity: string
  average_unit_cost: string
  total_value: string
}

export interface ValuationReport {
  generated_at: string
  total_value: string
  rows: ValuationRow[]
}

export interface MovementSummaryRow {
  day: string
  movement_type: MovementType
  movements: number
  quantity: string
}

export interface MovementSummaryReport {
  generated_at: string
  window_days: number
  totals: Record<MovementType, number>
  rows: MovementSummaryRow[]
}

export interface VelocityRow {
  product_id: string
  sku: string
  product_name: string
  picked_quantity: string
  pick_count: number
  daily_average: string
}

export interface VelocityReport {
  generated_at: string
  window_days: number
  rows: VelocityRow[]
}

export interface ReportFilter {
  days?: number
  limit?: number
  warehouse_id?: string
}

function buildQuery(filter: ReportFilter): string {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(filter)) {
    if (value !== undefined && value !== null && value !== '') {
      params.set(key, String(value))
    }
  }
  const query = params.toString()
  return query ? `?${query}` : ''
}

export function getDashboardReport(filter: ReportFilter = {}) {
  return apiRequest<DashboardReport>(`/reports/dashboard${buildQuery(filter)}`)
}

export function getValuationReport(filter: ReportFilter = {}) {
  return apiRequest<ValuationReport>(`/reports/valuation${buildQuery(filter)}`)
}

export function getMovementSummaryReport(filter: ReportFilter = {}) {
  return apiRequest<MovementSummaryReport>(`/reports/movement-summary${buildQuery(filter)}`)
}

export function getVelocityReport(filter: ReportFilter = {}) {
  return apiRequest<VelocityReport>(`/reports/velocity${buildQuery(filter)}`)
}
