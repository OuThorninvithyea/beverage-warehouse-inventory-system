import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import * as warehousesApi from '@/api/warehouses'
import { useWarehousesStore } from '@/stores/warehouses'

vi.mock('@/api/warehouses')

const sampleWarehouse: warehousesApi.Warehouse = {
  id: 'wh-1',
  code: 'PP-01',
  name: 'Phnom Penh Main',
  address: null,
  is_active: true,
  created_at: '2026-08-19T00:00:00Z',
  updated_at: '2026-08-19T00:00:00Z',
}

const sampleLocation: warehousesApi.Location = {
  id: 'loc-1',
  warehouse_id: 'wh-1',
  code: 'A-01',
  zone: 'Ambient',
  aisle: 'A-01',
  rack: null,
  shelf: null,
  barcode: null,
  is_pickable: true,
  is_active: true,
  created_at: '2026-08-19T00:00:00Z',
  updated_at: '2026-08-19T00:00:00Z',
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('useWarehousesStore warehouses', () => {
  it('fetchWarehouses populates state from the API', async () => {
    vi.mocked(warehousesApi.listWarehouses).mockResolvedValue({
      items: [sampleWarehouse],
      page: { next_cursor: null, has_more: false },
    })
    const store = useWarehousesStore()
    await store.fetchWarehouses()
    expect(store.warehouses).toEqual([sampleWarehouse])
    expect(store.loading).toBe(false)
    expect(store.error).toBeNull()
  })

  it('fetchWarehouses records a message on failure', async () => {
    vi.mocked(warehousesApi.listWarehouses).mockRejectedValue(new Error('boom'))
    const store = useWarehousesStore()
    await store.fetchWarehouses()
    expect(store.error).toBe('boom')
    expect(store.loading).toBe(false)
  })

  it('loadMoreWarehouses appends using the stored cursor', async () => {
    vi.mocked(warehousesApi.listWarehouses).mockResolvedValueOnce({
      items: [sampleWarehouse],
      page: { next_cursor: 'cursor-1', has_more: true },
    })
    const store = useWarehousesStore()
    await store.fetchWarehouses()

    const secondWarehouse = { ...sampleWarehouse, id: 'wh-2', code: 'SR-01' }
    vi.mocked(warehousesApi.listWarehouses).mockResolvedValueOnce({
      items: [secondWarehouse],
      page: { next_cursor: null, has_more: false },
    })
    await store.loadMoreWarehouses()

    expect(store.warehouses).toEqual([sampleWarehouse, secondWarehouse])
    expect(warehousesApi.listWarehouses).toHaveBeenLastCalledWith({
      search: undefined,
      after: 'cursor-1',
    })
    expect(store.hasMore).toBe(false)
  })

  it('loadMoreWarehouses does nothing when there is no more to load', async () => {
    const store = useWarehousesStore()
    await store.loadMoreWarehouses()
    expect(warehousesApi.listWarehouses).not.toHaveBeenCalled()
  })

  it('createWarehouse prepends the new warehouse to state', async () => {
    const created = { ...sampleWarehouse, id: 'wh-2', code: 'SR-01' }
    vi.mocked(warehousesApi.createWarehouse).mockResolvedValue(created)
    const store = useWarehousesStore()
    const result = await store.createWarehouse({ code: 'SR-01', name: 'Siem Reap' })
    expect(result).toEqual(created)
    expect(store.warehouses[0]).toEqual(created)
  })

  it('deactivateWarehouse marks the matching warehouse inactive in place', async () => {
    vi.mocked(warehousesApi.listWarehouses).mockResolvedValue({
      items: [sampleWarehouse],
      page: { next_cursor: null, has_more: false },
    })
    vi.mocked(warehousesApi.deactivateWarehouse).mockResolvedValue(undefined)
    const store = useWarehousesStore()
    await store.fetchWarehouses()
    await store.deactivateWarehouse('wh-1')
    expect(store.warehouses[0].is_active).toBe(false)
  })
})

describe('useWarehousesStore locations', () => {
  it('fetchLocations populates location state', async () => {
    vi.mocked(warehousesApi.listLocations).mockResolvedValue({
      items: [sampleLocation],
      page: { next_cursor: null, has_more: false },
    })
    const store = useWarehousesStore()
    await store.fetchLocations('wh-1')
    expect(store.locations).toEqual([sampleLocation])
    expect(store.locationsLoading).toBe(false)
  })

  it('loadMoreLocations appends using the stored cursor', async () => {
    vi.mocked(warehousesApi.listLocations).mockResolvedValueOnce({
      items: [sampleLocation],
      page: { next_cursor: 'cursor-1', has_more: true },
    })
    const store = useWarehousesStore()
    await store.fetchLocations('wh-1')

    const secondLocation = { ...sampleLocation, id: 'loc-2', code: 'A-02' }
    vi.mocked(warehousesApi.listLocations).mockResolvedValueOnce({
      items: [secondLocation],
      page: { next_cursor: null, has_more: false },
    })
    await store.loadMoreLocations('wh-1')

    expect(store.locations).toEqual([sampleLocation, secondLocation])
    expect(warehousesApi.listLocations).toHaveBeenLastCalledWith('wh-1', {
      after: 'cursor-1',
    })
  })

  it('createLocation prepends the new location to state', async () => {
    const created = { ...sampleLocation, id: 'loc-2', code: 'A-02' }
    vi.mocked(warehousesApi.createLocation).mockResolvedValue(created)
    const store = useWarehousesStore()
    const result = await store.createLocation('wh-1', { code: 'A-02' })
    expect(result).toEqual(created)
    expect(store.locations[0]).toEqual(created)
  })

  it('deactivateLocation marks the matching location inactive in place', async () => {
    vi.mocked(warehousesApi.listLocations).mockResolvedValue({
      items: [sampleLocation],
      page: { next_cursor: null, has_more: false },
    })
    vi.mocked(warehousesApi.deactivateLocation).mockResolvedValue(undefined)
    const store = useWarehousesStore()
    await store.fetchLocations('wh-1')
    await store.deactivateLocation('wh-1', 'loc-1')
    expect(store.locations[0].is_active).toBe(false)
  })
})
