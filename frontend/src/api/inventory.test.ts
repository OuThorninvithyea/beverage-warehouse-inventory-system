import { beforeEach, describe, expect, it, vi } from 'vitest'

import {
  adjustStock,
  listBalances,
  listLots,
  listMovements,
  pickStock,
  receiveStock,
  transferStock,
} from '@/api/inventory'

vi.mock('@/api/client', () => ({
  apiRequest: vi.fn(),
}))

import { apiRequest } from '@/api/client'

const mockApiRequest = apiRequest as unknown as ReturnType<typeof vi.fn>

describe('inventory API client', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  it('listBalances includes query filters', async () => {
    mockApiRequest.mockResolvedValueOnce({ items: [], page: { next_cursor: null, has_more: false } })
    await listBalances({ warehouse_id: 'wh-1', location_id: 'loc-1' })
    expect(mockApiRequest).toHaveBeenCalledWith('/inventory?warehouse_id=wh-1&location_id=loc-1')
  })

  it('listLots requests product lots endpoint', async () => {
    mockApiRequest.mockResolvedValueOnce([])
    await listLots('prod-1')
    expect(mockApiRequest).toHaveBeenCalledWith('/inventory/products/prod-1/lots')
  })

  it('receiveStock posts receive payload', async () => {
    mockApiRequest.mockResolvedValueOnce({ movement: {}, balance: {} })
    const payload = {
      location_id: 'loc-1',
      product_id: 'prod-1',
      quantity: '50.000',
      unit_cost: '1.2500',
    }
    await receiveStock(payload)
    expect(mockApiRequest).toHaveBeenCalledWith('/inventory/movements/receive', {
      method: 'POST',
      body: JSON.stringify(payload),
    })
  })

  it('pickStock posts FEFO pick payload', async () => {
    mockApiRequest.mockResolvedValueOnce({ movements: [], total_quantity: '10.000' })
    const payload = {
      location_id: 'loc-1',
      product_id: 'prod-1',
      quantity: '10.000',
    }
    await pickStock(payload)
    expect(mockApiRequest).toHaveBeenCalledWith('/inventory/movements/pick', {
      method: 'POST',
      body: JSON.stringify(payload),
    })
  })

  it('transferStock posts transfer payload', async () => {
    mockApiRequest.mockResolvedValueOnce({ movement: {}, source_balance: {}, destination_balance: {} })
    const payload = {
      product_id: 'prod-1',
      from_location_id: 'loc-1',
      to_location_id: 'loc-2',
      quantity: '5.000',
    }
    await transferStock(payload)
    expect(mockApiRequest).toHaveBeenCalledWith('/inventory/movements/transfer', {
      method: 'POST',
      body: JSON.stringify(payload),
    })
  })

  it('adjustStock posts adjustment payload', async () => {
    mockApiRequest.mockResolvedValueOnce({ movement: {}, balance: {} })
    const payload = {
      location_id: 'loc-1',
      product_id: 'prod-1',
      direction: 'increase' as const,
      quantity: '2.000',
      notes: 'Cycle count correction',
    }
    await adjustStock(payload)
    expect(mockApiRequest).toHaveBeenCalledWith('/inventory/movements/adjust', {
      method: 'POST',
      body: JSON.stringify(payload),
    })
  })

  it('listMovements requests movement log with filters', async () => {
    mockApiRequest.mockResolvedValueOnce({ items: [], page: { next_cursor: null, has_more: false } })
    await listMovements({ movement_type: 'receive' })
    expect(mockApiRequest).toHaveBeenCalledWith('/inventory/movements?movement_type=receive')
  })
})
