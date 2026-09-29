import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'

import { useLocationStock } from './location-stock'

vi.mock('@/api/catalog', () => ({ listProducts: vi.fn() }))
vi.mock('@/api/inventory', () => ({ listBalances: vi.fn(), listLots: vi.fn() }))

import { listProducts } from '@/api/catalog'
import { listBalances, listLots } from '@/api/inventory'

const page = <T>(items: T[], next: string | null = null) => ({
  items,
  page: { next_cursor: next, has_more: next !== null },
})

const product = (id: string, name: string) =>
  ({ id, name, sku: id.toUpperCase(), unit: 'case', is_lot_tracked: true, is_active: true }) as never

const balance = (product_id: string, lot_id: string | null, quantity: string, reserved = '0') =>
  ({
    id: `${product_id}-${lot_id}`,
    location_id: 'loc-a',
    product_id,
    lot_id,
    quantity,
    reserved_quantity: reserved,
    available_quantity: String(Number(quantity) - Number(reserved)),
  }) as never

const flush = async () => {
  for (let i = 0; i < 5; i++) await Promise.resolve()
  await nextTick()
}

beforeEach(() => {
  vi.mocked(listProducts).mockReset()
  vi.mocked(listBalances).mockReset()
  vi.mocked(listLots).mockReset()
  vi.mocked(listProducts).mockResolvedValue(
    page([product('cola', 'Cola'), product('water', 'Water'), product('juice', 'Juice')]),
  )
})

describe('useLocationStock', () => {
  it('offers only products with available stock at the location', async () => {
    vi.mocked(listBalances).mockResolvedValue(
      page([
        balance('cola', 'lot-1', '10'),
        balance('water', 'lot-2', '0'), // empty shelf
        balance('juice', 'lot-3', '5', '5'), // fully reserved
      ]),
    )
    const location = ref('')
    const stock = useLocationStock(location)
    await stock.loadCatalog()
    location.value = 'loc-a'
    await flush()

    expect(stock.stockedProducts.value.map((row) => row.product.id)).toEqual(['cola'])
  })

  it('sums a product across its lots at the location', async () => {
    vi.mocked(listBalances).mockResolvedValue(
      page([balance('cola', 'lot-1', '10'), balance('cola', 'lot-2', '4', '1')]),
    )
    const location = ref('loc-a')
    const stock = useLocationStock(location)
    await stock.loadCatalog()
    await stock.reload()

    expect(stock.availableFor('cola')).toBe(13)
    expect(stock.availableFor('cola', 'lot-2')).toBe(3)
    expect(stock.availableFor('water')).toBe(0)
  })

  it('offers only lots that are on this shelf, earliest expiry first', async () => {
    vi.mocked(listBalances).mockResolvedValue(
      page([balance('cola', 'lot-late', '8'), balance('cola', 'lot-soon', '2')]),
    )
    // lot-elsewhere exists for the product but is stocked in another location.
    vi.mocked(listLots).mockResolvedValue([
      { id: 'lot-late', lot_number: 'L-LATE', expiration_date: '2027-06-01' },
      { id: 'lot-elsewhere', lot_number: 'L-ELSEWHERE', expiration_date: '2026-10-01' },
      { id: 'lot-soon', lot_number: 'L-SOON', expiration_date: '2026-11-01' },
    ] as never)
    const location = ref('loc-a')
    const stock = useLocationStock(location)
    await stock.loadCatalog()
    await stock.reload()

    const lots = await stock.lotsFor('cola')
    expect(lots.map((row) => row.lot.lot_number)).toEqual(['L-SOON', 'L-LATE'])
    expect(lots.map((row) => row.available)).toEqual([2, 8])
  })

  it('hides a stocked product that is no longer active', async () => {
    vi.mocked(listBalances).mockResolvedValue(page([balance('retired', 'lot-9', '12')]))
    const location = ref('loc-a')
    const stock = useLocationStock(location)
    await stock.loadCatalog()
    await stock.reload()

    expect(stock.stockedProducts.value).toEqual([])
  })

  it('follows every page of balances and products', async () => {
    vi.mocked(listProducts)
      .mockResolvedValueOnce(page([product('cola', 'Cola')], 'p2'))
      .mockResolvedValueOnce(page([product('water', 'Water')]))
    vi.mocked(listBalances)
      .mockResolvedValueOnce(page([balance('cola', 'lot-1', '3')], 'b2'))
      .mockResolvedValueOnce(page([balance('water', 'lot-2', '4')]))
    const location = ref('loc-a')
    const stock = useLocationStock(location)
    await stock.loadCatalog()
    await stock.reload()

    expect(stock.stockedProducts.value.map((row) => row.product.id)).toEqual(['cola', 'water'])
  })

  it('ignores a slow response for a location that is no longer selected', async () => {
    let releaseFirst: (value: unknown) => void = () => {}
    vi.mocked(listBalances)
      .mockImplementationOnce(() => new Promise((resolve) => (releaseFirst = resolve)) as never)
      .mockResolvedValueOnce(page([balance('water', 'lot-2', '6')]))
    const location = ref('')
    const stock = useLocationStock(location)
    await stock.loadCatalog()

    location.value = 'loc-slow'
    await flush()
    location.value = 'loc-fast'
    await flush()
    releaseFirst(page([balance('cola', 'lot-1', '9')]))
    await flush()

    expect(stock.stockedProducts.value.map((row) => row.product.id)).toEqual(['water'])
  })
})
