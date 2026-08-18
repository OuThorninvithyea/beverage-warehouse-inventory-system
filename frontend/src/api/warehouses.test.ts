import { beforeEach, describe, expect, it, vi } from 'vitest'

import { apiRequest } from '@/api/client'
import {
  createWarehouse,
  deactivateWarehouse,
  getWarehouse,
  listWarehouses,
  updateWarehouse,
} from '@/api/warehouses'

vi.mock('@/api/client', () => ({
  apiRequest: vi.fn(),
}))

const mockedApiRequest = vi.mocked(apiRequest)

beforeEach(() => {
  mockedApiRequest.mockReset()
  mockedApiRequest.mockResolvedValue({} as never)
})

describe('warehouses api', () => {
  it('lists warehouses with no filters and no query string', async () => {
    await listWarehouses()
    expect(mockedApiRequest).toHaveBeenCalledWith('/warehouses')
  })

  it('lists warehouses with filters as query params', async () => {
    await listWarehouses({ search: 'PP', is_active: true, limit: 10 })
    expect(mockedApiRequest).toHaveBeenCalledWith(
      '/warehouses?search=PP&is_active=true&limit=10',
    )
  })

  it('gets a single warehouse by id', async () => {
    await getWarehouse('wh-1')
    expect(mockedApiRequest).toHaveBeenCalledWith('/warehouses/wh-1')
  })

  it('creates a warehouse with a POST body', async () => {
    await createWarehouse({ code: 'PP-01', name: 'Phnom Penh Main' })
    expect(mockedApiRequest).toHaveBeenCalledWith('/warehouses', {
      method: 'POST',
      body: JSON.stringify({ code: 'PP-01', name: 'Phnom Penh Main' }),
    })
  })

  it('updates a warehouse with a PUT body', async () => {
    await updateWarehouse('wh-1', { code: 'PP-01', name: 'Renamed' })
    expect(mockedApiRequest).toHaveBeenCalledWith('/warehouses/wh-1', {
      method: 'PUT',
      body: JSON.stringify({ code: 'PP-01', name: 'Renamed' }),
    })
  })

  it('deactivates a warehouse with DELETE', async () => {
    await deactivateWarehouse('wh-1')
    expect(mockedApiRequest).toHaveBeenCalledWith('/warehouses/wh-1', {
      method: 'DELETE',
    })
  })
})
