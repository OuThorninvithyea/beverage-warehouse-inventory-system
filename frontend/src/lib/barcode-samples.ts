/**
 * Barcode values for manual testing of the scanner route.
 *
 * The valid entries are the same values the development seeder writes
 * (`backend/cmd/seed/demo_data.go`), so a scan or manual entry resolves to a
 * real product or location once `make seed-demo` has run. The printable
 * versions live in `docs/barcode-test-data.md`.
 */
export interface BarcodeSample {
  value: string
  label: string
  format: 'EAN-13' | 'UPC-A'
}

export const validProductSamples: BarcodeSample[] = [
  { value: '8841000001015', label: 'Angkor Cola Can 330 ml', format: 'EAN-13' },
  { value: '8841000002012', label: 'Kulen Still Water 500 ml', format: 'EAN-13' },
  { value: '8841000003019', label: 'Tonle Orange Juice 1 L', format: 'EAN-13' },
  { value: '8841000005013', label: 'Riverside Lager Can 330 ml', format: 'EAN-13' },
  { value: '049000012347', label: 'Imported Cola Can 355 ml', format: 'UPC-A' },
  { value: '012000012341', label: 'Imported Root Beer Can 355 ml', format: 'UPC-A' },
]

export const validLocationSamples: BarcodeSample[] = [
  { value: '2001000101016', label: 'PP-CENTRAL / A-01-01', format: 'EAN-13' },
  { value: '2001000102013', label: 'PP-CENTRAL / COLD-01', format: 'EAN-13' },
  { value: '2002000101013', label: 'SR-DEPOT / A-01-01', format: 'EAN-13' },
]

/** Values that must be rejected before any lookup is attempted. */
export const invalidSamples: Array<{ value: string; label: string }> = [
  { value: '8841000001016', label: 'EAN-13 with a wrong check digit' },
  { value: '049000012348', label: 'UPC-A with a wrong check digit' },
  { value: '884100000101', label: '12 digits that are not a valid UPC-A' },
  { value: '88410000010155', label: 'Too many digits' },
  { value: '88410000O1015', label: 'Contains a letter' },
]
