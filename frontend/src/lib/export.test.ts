import { describe, expect, it, vi } from 'vitest'

import { exportToCSV } from '@/lib/export'

describe('exportToCSV utility', () => {
  it('does nothing when rows are empty', () => {
    const createElementSpy = vi.spyOn(document, 'createElement')
    exportToCSV('test', [])
    expect(createElementSpy).not.toHaveBeenCalled()
  })

  it('triggers a download for non-empty rows', () => {
    const linkMock = {
      setAttribute: vi.fn(),
      click: vi.fn(),
    } as any
    vi.spyOn(document, 'createElement').mockReturnValue(linkMock)
    vi.spyOn(document.body, 'appendChild').mockImplementation(() => linkMock)
    vi.spyOn(document.body, 'removeChild').mockImplementation(() => linkMock)
    URL.createObjectURL = vi.fn().mockReturnValue('blob:http://localhost/test')

    exportToCSV('test-inventory', [{ sku: 'COKE-1', qty: '10' }])
    expect(linkMock.setAttribute).toHaveBeenCalledWith('download', 'test-inventory.csv')
    expect(linkMock.click).toHaveBeenCalled()
  })
})
