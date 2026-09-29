import { ApiClientError, apiRequest } from '@/api/client'
import type { Product } from '@/api/catalog'
import type { Location } from '@/api/warehouses'

/**
 * One scan button serves every movement form, so a scanned code has to be
 * recognised as either a shelf label or a product carton. Shelf labels are
 * matched against the locations the user can already see; anything else is
 * looked up in the catalog.
 */
export type ScanResult =
  | { kind: 'location'; location: Location }
  | { kind: 'product'; product: Product }
  | { kind: 'unknown'; code: string }

/** Feedback shown under a form's scan button. */
export interface ScanFeedback {
  tone: 'success' | 'warning'
  text: string
}

export async function resolveScan(rawCode: string, locations: Location[]): Promise<ScanResult> {
  const code = rawCode.trim()
  const location = locations.find((row) => row.barcode === code)
  if (location) return { kind: 'location', location }

  try {
    const product = await apiRequest<Product>(`/products/by-barcode/${encodeURIComponent(code)}`)
    return { kind: 'product', product }
  } catch (err: unknown) {
    // Not found is an answer, not a failure: the code is simply not ours yet.
    if (err instanceof ApiClientError && err.status === 404) return { kind: 'unknown', code }
    throw err
  }
}
