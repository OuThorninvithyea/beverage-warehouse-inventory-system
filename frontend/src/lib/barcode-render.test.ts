import { describe, expect, it } from 'vitest'

import { buildBarcodeSymbol, encodeBarcodeModules } from './barcode-render'

// Expected module patterns for two seeded values, produced by an independent
// encoder and confirmed by decoding the rendered symbol with ZXing — the same
// decoder the scanner route uses. They pin the output exactly, so a change in
// the parity tables cannot slip through.
const EAN13_8841000001015 =
  '10101101110011101001100101001110100111000110101010111001011100101100110111001011001101001110101'
const UPCA_049000012347 =
  '10100011010100011000101100011010001101000110101010111001011001101101100100001010111001000100101'

describe('encodeBarcodeModules', () => {
  it('encodes a seeded EAN-13 product barcode', () => {
    expect(encodeBarcodeModules('8841000001015')).toBe(EAN13_8841000001015)
  })

  it('encodes UPC-A as EAN-13 with a leading zero', () => {
    expect(encodeBarcodeModules('049000012347')).toBe(UPCA_049000012347)
  })

  it('produces 95 modules, guard patterns included', () => {
    expect(encodeBarcodeModules('8841000001015')).toHaveLength(95)
  })

  it('starts and ends with a guard and carries a centre guard', () => {
    const modules = encodeBarcodeModules('8841000001015') as string
    expect(modules.startsWith('101')).toBe(true)
    expect(modules.endsWith('101')).toBe(true)
    expect(modules.slice(45, 50)).toBe('01010')
  })

  it('refuses values that are not printable symbols', () => {
    // A wrong check digit has no legitimate symbol, so it must not render.
    expect(encodeBarcodeModules('8841000001016')).toBeNull()
    expect(encodeBarcodeModules('884100000101')).toBeNull()
    expect(encodeBarcodeModules('88410000O1015')).toBeNull()
    expect(encodeBarcodeModules('')).toBeNull()
  })

  it('tolerates surrounding whitespace the way the validator does', () => {
    expect(encodeBarcodeModules('  8841000001015  ')).toBe(EAN13_8841000001015)
  })
})

describe('buildBarcodeSymbol', () => {
  it('merges module runs into bars and adds a quiet zone', () => {
    const symbol = buildBarcodeSymbol('8841000001015', 2, 60)
    expect(symbol).not.toBeNull()
    const value = symbol!

    // 95 modules plus 9 quiet modules each side, at 2 units per module.
    expect(value.width).toBe((95 + 18) * 2)
    expect(value.height).toBe(60)
    expect(value.value).toBe('8841000001015')

    // No bar may start inside the quiet zone.
    expect(Math.min(...value.bars.map((bar) => bar.x))).toBeGreaterThanOrEqual(9 * 2)
    // Every bar covers at least one module and stays inside the symbol.
    for (const bar of value.bars) {
      expect(bar.width).toBeGreaterThanOrEqual(2)
      expect(bar.x + bar.width).toBeLessThanOrEqual(value.width)
    }

    // Reconstructing the pattern from the bars must reproduce the encoding,
    // which proves the run merging is lossless.
    let rebuilt = ''
    let cursor = 9 * 2
    for (const bar of value.bars) {
      rebuilt += '0'.repeat((bar.x - cursor) / 2)
      rebuilt += '1'.repeat(bar.width / 2)
      cursor = bar.x + bar.width
    }
    rebuilt += '0'.repeat((value.width - 9 * 2 - cursor) / 2)
    expect(rebuilt).toBe(EAN13_8841000001015)
  })

  it('returns null for an unprintable value', () => {
    expect(buildBarcodeSymbol('8841000001016')).toBeNull()
  })
})
