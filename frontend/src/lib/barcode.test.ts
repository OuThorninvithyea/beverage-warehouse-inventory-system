import { describe, expect, it } from 'vitest'

import { validateBarcode } from './barcode'

describe('validateBarcode', () => {
  it('accepts a valid EAN-13 barcode', () => {
    expect(validateBarcode('4006381333931')).toMatchObject({
      format: 'EAN-13',
      valid: true,
    })
  })

  it('accepts a valid UPC-A barcode', () => {
    expect(validateBarcode('036000291452')).toMatchObject({
      format: 'UPC-A',
      valid: true,
    })
  })

  it('rejects an invalid check digit', () => {
    expect(validateBarcode('4006381333932').valid).toBe(false)
  })

  it('rejects unsupported lengths and non-digits', () => {
    expect(validateBarcode('1234').valid).toBe(false)
    expect(validateBarcode('ABC-123').valid).toBe(false)
  })
})
