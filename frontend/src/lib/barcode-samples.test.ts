import { describe, expect, it } from 'vitest'

import { validateBarcode } from './barcode'
import {
  invalidSamples,
  validLocationSamples,
  validProductSamples,
} from './barcode-samples'

describe('barcode samples', () => {
  const valid = [...validProductSamples, ...validLocationSamples]

  it.each(valid)('accepts the seeded value $value', (sample) => {
    const result = validateBarcode(sample.value)
    expect(result.valid).toBe(true)
    expect(result.format).toBe(sample.format)
  })

  it.each(invalidSamples)('rejects $value', (sample) => {
    expect(validateBarcode(sample.value).valid).toBe(false)
  })

  it('does not repeat a value across the sample sets', () => {
    const values = [...valid, ...invalidSamples].map((sample) => sample.value)
    expect(new Set(values).size).toBe(values.length)
  })
})
