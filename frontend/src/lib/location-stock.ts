import { computed, ref, watch, type Ref } from 'vue'

import { listProducts, type Product } from '@/api/catalog'
import { listBalances, listLots, type Lot, type StockBalance } from '@/api/inventory'

/**
 * What can actually leave a location.
 *
 * Picks, transfers and decreasing adjustments are location-scoped on the
 * server: they only see stock in the exact location named. Offering every
 * product (or every lot of a product, from any shelf) lets the operator build
 * a request the server must reject with INSUFFICIENT_STOCK. This loads the
 * location's balances and exposes only what has available quantity there,
 * so an empty product or lot is never offered in the first place.
 *
 * The server still enforces the rule; this only stops the form from offering
 * choices that cannot succeed.
 */

export interface StockedProduct {
  product: Product
  /** Available across every lot at this location: on hand minus reserved. */
  available: number
}

export interface StockedLot {
  lot: Lot
  available: number
}

const PAGE = 100

/** Every active product. The store's list is paginated, so it cannot be used. */
async function loadActiveCatalog(): Promise<Map<string, Product>> {
  const catalog = new Map<string, Product>()
  let after: string | undefined
  do {
    const page = await listProducts({ is_active: true, limit: PAGE, after })
    for (const product of page.items) catalog.set(product.id, product)
    after = page.page.has_more ? (page.page.next_cursor ?? undefined) : undefined
  } while (after)
  return catalog
}

async function loadLocationBalances(locationId: string): Promise<StockBalance[]> {
  const balances: StockBalance[] = []
  let after: string | undefined
  do {
    const page = await listBalances({ location_id: locationId, limit: PAGE, after })
    balances.push(...page.items)
    after = page.page.has_more ? (page.page.next_cursor ?? undefined) : undefined
  } while (after)
  return balances
}

export function useLocationStock(locationId: Ref<string>) {
  const catalog = ref<Map<string, Product>>(new Map())
  const balances = ref<StockBalance[]>([])
  const loading = ref(false)
  const error = ref('')

  // Lot details are per product and do not change while a dialog is open.
  const lotCache = new Map<string, Lot[]>()
  // A slow response for a previous location must not overwrite a newer one.
  let request = 0

  async function loadCatalog() {
    catalog.value = await loadActiveCatalog()
  }

  async function reload() {
    const id = locationId.value
    const current = ++request
    if (!id) {
      balances.value = []
      return
    }
    loading.value = true
    error.value = ''
    try {
      const rows = await loadLocationBalances(id)
      if (current === request) balances.value = rows
    } catch (e: unknown) {
      if (current === request) {
        balances.value = []
        error.value = e instanceof Error ? e.message : 'Could not load stock for this location'
      }
    } finally {
      if (current === request) loading.value = false
    }
  }

  watch(locationId, () => void reload())

  /** Available quantity per product and per lot, ignoring anything at zero. */
  const availability = computed(() => {
    const byProduct = new Map<string, number>()
    const byLot = new Map<string, number>()
    for (const balance of balances.value) {
      const available = Number(balance.available_quantity)
      if (!(available > 0)) continue
      byProduct.set(balance.product_id, (byProduct.get(balance.product_id) ?? 0) + available)
      if (balance.lot_id) {
        byLot.set(balance.lot_id, (byLot.get(balance.lot_id) ?? 0) + available)
      }
    }
    return { byProduct, byLot }
  })

  /**
   * Products with available stock here, alphabetically. Inactive products are
   * excluded even if a stale balance exists, because the server refuses them.
   */
  const stockedProducts = computed<StockedProduct[]>(() =>
    [...availability.value.byProduct.entries()]
      .map(([id, available]) => ({ product: catalog.value.get(id), available }))
      .filter((row): row is StockedProduct => row.product !== undefined)
      .sort((a, b) => a.product.name.localeCompare(b.product.name)),
  )

  function availableFor(productId: string, lotId?: string | null): number {
    if (lotId) return availability.value.byLot.get(lotId) ?? 0
    return availability.value.byProduct.get(productId) ?? 0
  }

  /**
   * Lots of a product that have stock at this location, earliest expiry
   * first, the same order FEFO would draw them.
   */
  async function lotsFor(productId: string): Promise<StockedLot[]> {
    if (!productId) return []
    if (!lotCache.has(productId)) lotCache.set(productId, await listLots(productId))
    return (lotCache.get(productId) ?? [])
      .filter((lot) => (availability.value.byLot.get(lot.id) ?? 0) > 0)
      .map((lot) => ({ lot, available: availability.value.byLot.get(lot.id) ?? 0 }))
      .sort((a, b) =>
        (a.lot.expiration_date ?? '9999-12-31').localeCompare(b.lot.expiration_date ?? '9999-12-31'),
      )
  }

  const allProducts = computed(() =>
    [...catalog.value.values()].sort((a, b) => a.name.localeCompare(b.name)),
  )

  return {
    loading,
    error,
    stockedProducts,
    allProducts,
    availableFor,
    lotsFor,
    loadCatalog,
    reload,
  }
}

/** Formats a quantity for an option label without trailing zeros. */
export function formatQuantity(value: number): string {
  return Number.isInteger(value) ? String(value) : value.toFixed(3).replace(/\.?0+$/, '')
}
