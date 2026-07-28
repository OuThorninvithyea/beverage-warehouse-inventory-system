# Week 7 validation

Validation was run on 28 July 2026 for the Week 6 catch-up and Week 7
authentication/RBAC milestone.

## Backend

```text
go test ./...  -> passed
go vet ./...   -> passed
```

Authentication coverage verifies:

- RS256 access-token signing and validation.
- tampered-token rejection.
- production refusal when RSA keys are absent.
- bcrypt password checking with generic credential errors.
- refresh tokens stored as hashes.
- atomic refresh-token rotation.
- admin allowed and viewer forbidden by role middleware.

## Frontend and barcode

```text
npm test          -> 4 barcode tests passed
npm run typecheck -> passed
npm run build     -> passed
```

The barcode tests cover valid EAN-13, valid UPC-A, invalid checksums,
unsupported lengths and non-digit input. The `/barcode-test` route supports
camera capture, manual entry and USB keyboard-wedge input while explicitly
preventing any inventory mutation.

## Docker and integrated API

```text
docker compose config --quiet -> passed
database migration            -> latest, no change
development admin seed        -> passed
API and frontend image build  -> passed
```

The live smoke test used the frontend Nginx proxy and confirmed:

```json
{
  "login_success": true,
  "role": "admin",
  "access_token_present": true,
  "refresh_token_present": true
}
```

- `GET /api/v1/auth/me` returned the authenticated admin.
- `GET /api/v1/admin/ping` returned `allowed`.
- a temporary `viewer` account received HTTP 403 `FORBIDDEN` from the same
  admin-only endpoint and was removed after the test.
- refresh returned a new access and refresh token.
- reusing the rotated refresh token returned HTTP 401
  `INVALID_REFRESH_TOKEN`.

## Remaining physical evidence

The software side of the barcode test is complete. GitHub Issue #6 and the
Week 6 review remain partially blocked until a real beverage barcode is scanned
with a physical phone and the device/browser/value/screenshot evidence is
recorded in `barcode-validation.md`.
