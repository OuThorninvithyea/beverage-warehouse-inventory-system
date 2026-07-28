# Barcode approach and device validation

## Library comparison

Versions were checked against npm on 28 July 2026.

| Option | Version | Strengths | Limitations |
| --- | --- | --- | --- |
| `@zxing/browser` | 0.2.1 | Focused browser API, EAN/UPC support, camera selection, maintained ZXing core | UI and duplicate handling must be built by the application |
| `html5-qrcode` | 2.3.8 | Built-in scanner UI, camera and image-file modes, broad format support | Larger opinionated UI and more integration styling |
| `@teckel/vue-barcode-reader` | 1.1.8 | Vue wrapper named in the proposal | Older Vue-oriented package and less control than direct ZXing integration |
| Native `BarcodeDetector` | Browser-dependent | No additional decoding dependency | Inconsistent Safari/device support; requires feature detection |

Decision: use `@zxing/browser` for the Week 6 test route because it gives direct
control over camera lifecycle, confirmation, duplicate suppression and the
responsive UI. Keep manual entry and USB keyboard-wedge input as fallbacks.

## Permission and deployment requirements

- Camera access requires a secure context: HTTPS in deployed environments.
- `http://localhost` is allowed by browsers for local development.
- The user must explicitly grant camera permission.
- Camera access can fail because of denial, OS privacy settings, another active
  application, missing rear camera, or embedded-browser restrictions.
- Prefer the rear/environment camera on phones, but allow users to choose.
- Stop the camera stream when leaving the scanner view.
- Never log camera frames or store images.

## Supported validation

- EAN-13: 13 digits with a valid check digit.
- UPC-A: 12 digits with a valid check digit.
- The decoded value is trimmed and validated before lookup.
- An unknown or invalid value shows an error and does not call a stock mutation.

## Duplicate and confirmation behavior

1. The first valid scan locks the result.
2. Repeated frames with the same barcode are ignored.
3. The operator chooses **Use value** or **Rescan**.
4. **Use value** only creates a lookup intent.
5. Receive, pick, transfer or adjustment requires a separate authenticated API
   request and explicit confirmation.

## USB scanner behavior

Most USB scanners use keyboard-wedge mode:

- focus the barcode input;
- scanner sends characters as keyboard events;
- scanner normally sends Enter as a suffix;
- debounce incomplete input and submit on Enter;
- no browser camera or scanning SDK is required.

Configure the scanner for EAN-13/UPC-A and an Enter suffix. Test that ordinary
typing still works and that one scan produces one lookup.

## Real phone test record

This must be completed using an actual beverage package; it cannot be truthfully
replaced by an emulator.

| Field | Result |
| --- | --- |
| Device and OS | Pending |
| Browser and version | Pending |
| Connection | HTTPS or localhost |
| Beverage/product | Pending |
| Printed barcode value | Pending |
| Captured value | Pending |
| First-scan time | Pending |
| Duplicate suppressed | Pending |
| Screenshot/video link | Pending |
| Result | Pending |

Acceptance requires printed and captured values to match, a screenshot or short
recording, and confirmation that the scan alone did not change inventory.
