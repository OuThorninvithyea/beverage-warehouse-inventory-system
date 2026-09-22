/**
 * EAN-13 / UPC-A symbol encoder.
 *
 * Printed labels have to be machine readable, so the bars are generated from
 * the digits rather than drawn decoratively. UPC-A is encoded as EAN-13 with a
 * leading zero, which is how the standard defines it and how ZXing reports it
 * back.
 */
import { validateBarcode } from '@/lib/barcode'

// Left-hand odd (L), left-hand even (G) and right-hand (R) digit patterns.
const LEFT_ODD = [
  '0001101', '0011001', '0010011', '0111101', '0100011',
  '0110001', '0101111', '0111011', '0110111', '0001011',
]
const LEFT_EVEN = [
  '0100111', '0110011', '0011011', '0100001', '0011101',
  '0111001', '0000101', '0010001', '0001001', '0010111',
]
const RIGHT = [
  '1110010', '1100110', '1101100', '1000010', '1011100',
  '1001110', '1010000', '1000100', '1001000', '1110100',
]
// The first digit selects the odd/even parity pattern of the left-hand group.
const PARITY = [
  'LLLLLL', 'LLGLGG', 'LLGGLG', 'LLGGGL', 'LGLLGG',
  'LGGLLG', 'LGGGLL', 'LGLGLG', 'LGLGGL', 'LGGLGL',
]

const QUIET_MODULES = 9

/**
 * Returns the module pattern as a string of '0' and '1', or null when the
 * value is not a valid EAN-13 or UPC-A barcode. A wrong check digit cannot be
 * drawn as a legitimate symbol, so callers must handle null rather than print
 * something unscannable.
 */
export function encodeBarcodeModules(value: string): string | null {
  const validation = validateBarcode(value)
  if (!validation.valid) return null

  const digits = validation.normalized.length === 12
    ? `0${validation.normalized}`
    : validation.normalized

  const parity = PARITY[Number(digits[0])]
  let modules = '101'
  for (let index = 1; index <= 6; index += 1) {
    const digit = Number(digits[index])
    modules += parity[index - 1] === 'L' ? LEFT_ODD[digit] : LEFT_EVEN[digit]
  }
  modules += '01010'
  for (let index = 7; index <= 12; index += 1) {
    modules += RIGHT[Number(digits[index])]
  }
  return `${modules}101`
}

export interface BarcodeBar {
  x: number
  width: number
}

export interface BarcodeSymbol {
  bars: BarcodeBar[]
  width: number
  height: number
  value: string
}

/**
 * Lays the module pattern out as bars, merging runs so the SVG carries one
 * rect per bar instead of one per module. moduleWidth and height are in the
 * caller's units; a quiet zone is added on both sides because a symbol without
 * one is unreliable to scan.
 */
export function buildBarcodeSymbol(
  value: string,
  moduleWidth = 2,
  height = 60,
): BarcodeSymbol | null {
  const modules = encodeBarcodeModules(value)
  if (!modules) return null

  const bars: BarcodeBar[] = []
  let index = 0
  while (index < modules.length) {
    if (modules[index] === '0') {
      index += 1
      continue
    }
    let run = 0
    while (index + run < modules.length && modules[index + run] === '1') {
      run += 1
    }
    bars.push({
      x: (QUIET_MODULES + index) * moduleWidth,
      width: run * moduleWidth,
    })
    index += run
  }

  return {
    bars,
    width: (modules.length + QUIET_MODULES * 2) * moduleWidth,
    height,
    value: validateBarcode(value).normalized,
  }
}
