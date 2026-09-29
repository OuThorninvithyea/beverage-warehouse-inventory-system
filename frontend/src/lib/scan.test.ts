import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/api/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/api/client')>()),
  apiRequest: vi.fn(),
}))

import { ApiClientError, apiRequest } from '@/api/client'
import type { Location } from '@/api/warehouses'

import { resolveScan } from './scan'

const shelf = { id: 'loc-1', code: 'A-01-01', barcode: '2002000101011', is_active: true } as Location

beforeEach(() => vi.mocked(apiRequest).mockClear())

describe('resolveScan', () => {
  it('recognises a shelf label without asking the server', async () => {
    expect(await resolveScan(' 2002000101011 ', [shelf])).toEqual({ kind: 'location', location: shelf })
    expect(apiRequest).not.toHaveBeenCalled()
  })

  it('looks anything else up as a product', async () => {
    vi.mocked(apiRequest).mockResolvedValueOnce({ id: 'p-1', name: 'Angkor Cola' })
    const result = await resolveScan('8841000001015', [shelf])
    expect(result).toEqual({ kind: 'product', product: { id: 'p-1', name: 'Angkor Cola' } })
    expect(apiRequest).toHaveBeenCalledWith('/products/by-barcode/8841000001015')
  })

  it('reports an unknown code instead of failing', async () => {
    vi.mocked(apiRequest).mockRejectedValueOnce(new ApiClientError(404, 'NOT_FOUND', 'not found'))
    expect(await resolveScan('5449000000996', [shelf])).toEqual({ kind: 'unknown', code: '5449000000996' })
  })

  it('passes real failures through', async () => {
    vi.mocked(apiRequest).mockRejectedValueOnce(new ApiClientError(500, 'INTERNAL', 'boom'))
    await expect(resolveScan('5449000000996', [])).rejects.toThrow('boom')
  })
})
