export type SupportedBarcodeFormat = 'EAN-13' | 'UPC-A'

export interface BarcodeValidation {
  normalized: string
  format: SupportedBarcodeFormat | null
  valid: boolean
  message: string
}

function expectedCheckDigit(valueWithoutCheckDigit: string): number {
  const weightedTotal = [...valueWithoutCheckDigit].reduce((total, digit, index) => {
    const value = Number(digit)
    const distanceFromRight = valueWithoutCheckDigit.length - 1 - index
    return total + value * (distanceFromRight % 2 === 0 ? 3 : 1)
  }, 0)

  return (10 - (weightedTotal % 10)) % 10
}

export function validateBarcode(value: string): BarcodeValidation {
  const normalized = value.trim().replace(/\s+/g, '')

  if (!/^\d+$/.test(normalized)) {
    return {
      normalized,
      format: null,
      valid: false,
      message: 'Use digits only for EAN-13 or UPC-A.',
    }
  }

  const format =
    normalized.length === 13 ? 'EAN-13' : normalized.length === 12 ? 'UPC-A' : null

  if (!format) {
    return {
      normalized,
      format: null,
      valid: false,
      message: 'The barcode must contain 12 UPC-A digits or 13 EAN-13 digits.',
    }
  }

  const body = normalized.slice(0, -1)
  const actual = Number(normalized.at(-1))
  const valid = expectedCheckDigit(body) === actual

  return {
    normalized,
    format,
    valid,
    message: valid ? `${format} checksum is valid.` : `${format} checksum is invalid.`,
  }
}
