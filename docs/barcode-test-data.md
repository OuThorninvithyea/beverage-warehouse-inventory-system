# Barcode test data

These are the barcodes the development seeder writes. Once `make seed-demo` has
run, every value below resolves to a real product or location, so the scanner
route, `GET /api/v1/products/by-barcode/{barcode}` and the manual and USB
fallbacks can all be exercised without a printed beverage package.

All values are fictional. Products use the 884 (Cambodia) EAN-13 prefix plus two
UPC-A values for imported goods; locations use the 20x restricted-circulation
prefix that GS1 reserves for internal use. Every check digit is valid, and a Go
test (`backend/cmd/seed/demo_test.go`) and a Vitest test
(`frontend/src/lib/barcode-samples.test.ts`) both enforce that against the same
validators the API and the UI use.

## Where these values live

| Place | Purpose |
| --- | --- |
| `backend/cmd/seed/demo_data.go` | Source of truth, written into PostgreSQL |
| `frontend/src/lib/barcode-samples.ts` | Quick-fill buttons on the barcode test route |
| `docs/barcode-test-sheet.html` | Printable sheet for a real camera scan |
| This document | Reference for manual and QA testing |

## Printing the sheet

Open `docs/barcode-test-sheet.html` in a browser and print it at **100% scale**
with "fit to page" turned off. The bars are drawn in millimetres, so a
correctly printed module is 0.5 mm wide — about 1.5x the EAN-13 nominal width,
which phone cameras read reliably. The bars are generated from the digits
printed beneath them, so the printed code and the expected value cannot drift
apart. Every value on the sheet was verified by decoding the generated pattern
with ZXing, the decoder the scanner route uses.

## Product barcodes

| Barcode | Format | SKU | Product |
| --- | --- | --- | --- |
| 8841000001015 | EAN-13 | BEV-COLA-330 | Angkor Cola Can 330 ml |
| 8841000001022 | EAN-13 | BEV-COLA-1500 | Angkor Cola PET 1.5 L |
| 8841000001039 | EAN-13 | BEV-LEMON-330 | Mekong Lemon Soda Can 330 ml |
| 8841000002012 | EAN-13 | WTR-STILL-500 | Kulen Still Water 500 ml |
| 8841000002029 | EAN-13 | WTR-SPARK-750 | Kulen Sparkling Water 750 ml |
| 8841000003019 | EAN-13 | JUI-ORNG-1000 | Tonle Orange Juice 1 L |
| 8841000003026 | EAN-13 | JUI-MANGO-250 | Tonle Mango Nectar 250 ml |
| 8841000004016 | EAN-13 | ENG-BOOST-250 | Bayon Boost Energy 250 ml |
| 8841000005013 | EAN-13 | BEE-LAGER-330 | Riverside Lager Can 330 ml |
| 8841000005020 | EAN-13 | BEE-STOUT-330 | Riverside Stout Can 330 ml |
| 8841000006010 | EAN-13 | COF-LATTE-240 | Phnom Coffee Latte Can 240 ml |
| 8841000006027 | EAN-13 | DRY-MILK-200 | Siem Dairy Milk 200 ml |
| 8841000007017 | EAN-13 | PKG-CRATE-24 | Returnable Crate 24 Slot (not lot tracked) |
| 049000012347 | UPC-A | IMP-COLA-355 | Imported Cola Can 355 ml |
| 012000012341 | UPC-A | IMP-ROOT-355 | Imported Root Beer Can 355 ml |

## Location barcodes

| Barcode | Warehouse | Location | Notes |
| --- | --- | --- | --- |
| 2001000190010 | PP-CENTRAL | RECV-DOCK | Receiving, not pickable |
| 2001000190027 | PP-CENTRAL | QUAR-01 | Quarantine, not pickable |
| 2001000101016 | PP-CENTRAL | A-01-01 | Ambient rack |
| 2001000101023 | PP-CENTRAL | A-01-02 | Ambient rack |
| 2001000102013 | PP-CENTRAL | COLD-01 | Chilled rack |
| 2002000190017 | SR-DEPOT | RECV-DOCK | Receiving, not pickable |
| 2002000101013 | SR-DEPOT | A-01-01 | Ambient rack |
| 2002000101020 | SR-DEPOT | A-01-02 | Ambient rack |

## Values that must be rejected

Use these to prove that validation happens before any lookup or stock mutation.

| Value | Why it must fail |
| --- | --- |
| 8841000001016 | EAN-13 with a wrong check digit |
| 049000012348 | UPC-A with a wrong check digit |
| 884100000101 | 12 digits that are not a valid UPC-A |
| 88410000010155 | Too many digits |
| 88410000O1015 | Contains a letter |
| 8841000009011 | Valid EAN-13 that is not in the catalog: expect `PRODUCT_NOT_FOUND`, not a validation error |

## Suggested manual test run

1. `make seed-demo`, then sign in as `manager@bwims.local`.
2. Open the barcode test route and press a seeded product value. The captured
   value must show a valid checksum and the correct format.
3. Press an invalid value. The route must report the checksum failure and must
   not call any stock endpoint.
4. Focus the manual input, scan with a USB keyboard-wedge scanner and confirm
   one scan produces exactly one lookup.
5. Print `barcode-test-sheet.html`, scan a code with a phone camera over HTTPS
   or `localhost`, and confirm the captured value matches the printed digits.
6. Repeat step 5 with a real beverage package and record the result in the
   device test table in [barcode-validation.md](barcode-validation.md). The
   generated sheet supports the camera test but does not replace that evidence.
