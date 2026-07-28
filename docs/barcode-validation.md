# Hardware barcode approach and device validation

## Primary decision

Use a USB or Bluetooth hardware barcode scanner in keyboard-wedge mode as the
default BWIMS scanning method. The scanner writes digits into the focused
barcode field and sends Enter as a suffix. Manual entry is the fallback.
Phone-camera scanning is optional and is not required for Week 6 acceptance.

## Hardware scanner behavior

Most USB and Bluetooth scanners can operate as keyboard-wedge devices:

- focus the barcode input before scanning;
- the scanner sends characters as keyboard events;
- configure an Enter suffix so one scan submits one value;
- accept EAN-13 and UPC-A digits only;
- ordinary typing must continue to work as a fallback;
- no browser camera permission or scanning SDK is required.

USB scanners connect directly or through USB-OTG where required. Bluetooth
scanners must be paired with the operating system and configured in HID keyboard
mode. The application does not depend on a scanner vendor SDK.

## Supported validation

- EAN-13: 13 digits with a valid check digit.
- UPC-A: 12 digits with a valid check digit.
- The decoded value is trimmed and validated before lookup.
- An unknown or invalid value shows an error and does not call a stock mutation.

## Submission and confirmation behavior

1. The focused field receives one barcode and the Enter suffix.
2. The value is trimmed and checksum validated.
3. One submitted scan produces one lookup intent.
4. Repeated submissions do not create stock movements.
5. Receive, pick, transfer or adjustment requires a separate authenticated API
   request and explicit confirmation.

## Optional phone-camera scanning

The `/barcode-test` route retains `@zxing/browser` as an optional alternative.
Versions were checked against npm on 28 July 2026.

| Option | Version | Strengths | Limitations |
| --- | --- | --- | --- |
| `@zxing/browser` | 0.2.1 | Focused browser API, EAN/UPC support and camera selection | Requires HTTPS, camera permission and device-specific testing |
| `html5-qrcode` | 2.3.8 | Built-in scanner UI, camera and image-file modes | Larger opinionated UI |
| `@teckel/vue-barcode-reader` | 1.1.8 | Vue wrapper named in the original proposal | Older package and less control |
| Native `BarcodeDetector` | Browser-dependent | No decoding dependency | Inconsistent browser support |

Camera access requires HTTPS outside `localhost`, explicit permission and safe
camera lifecycle cleanup. Camera frames and images must never be stored.

## Required hardware test record

Complete this record using an actual hardware scanner and beverage package.
Keyboard simulation or manual typing cannot replace this evidence.

| Field | Result |
| --- | --- |
| Scanner make/model | Pending |
| Connection | USB, USB-OTG or Bluetooth |
| Scanner mode | HID keyboard-wedge |
| Host device and OS | Pending |
| Browser and version | Pending |
| Beverage/product | Pending |
| Printed barcode value | Pending |
| Captured value | Pending |
| Enter suffix submitted | Pending |
| Scan-to-validation time | Pending |
| Inventory unchanged | Pending |
| Screenshot/video link | Pending |
| Result | Pending |

Acceptance requires printed and captured values to match, a screenshot or short
recording, confirmation that the Enter suffix submitted one lookup value, and
evidence that the scan alone did not change inventory.
