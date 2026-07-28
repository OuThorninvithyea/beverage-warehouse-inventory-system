# BWIMS QA test plan

## Test objective

Verify that each weekly increment satisfies its acceptance criteria without
breaking the backend, database, frontend, barcode workflow, security boundary,
or documented setup.

## Test environments

| Environment | Purpose |
| --- | --- |
| Local Go process | Unit tests, service tests and static analysis |
| Docker Compose | PostgreSQL migrations and integrated smoke tests |
| Desktop or Android browser with USB/Bluetooth scanner | Required hardware-scanner behavior and timing |
| Real HTTPS phone browser | Optional camera permission and EAN/UPC scan |

## Week 6–7 acceptance tests

| ID | Area | Test | Expected result | Evidence |
| --- | --- | --- | --- | --- |
| BE-01 | Backend | `GET /health` | HTTP 200 success envelope | Captured response |
| BE-02 | Backend | `GET /ready` with PostgreSQL running | HTTP 200 and `database: up` | Captured response |
| BE-03 | Backend | `go test ./...` | All packages pass | Terminal output |
| BE-04 | Backend | `go vet ./...` | No findings | Terminal output |
| DB-01 | Database | Run all up migrations | Migration reaches latest version | Migration output |
| DB-02 | Database | Inspect required tables and roles | 12 domain tables and four roles | SQL output |
| DB-03 | Database | Update/delete movement or audit row | PostgreSQL rejects operation | SQL output |
| DB-04 | Database | Duplicate product/location barcode | Unique constraint rejects duplicate | SQL output |
| FE-01 | Frontend | Start Vite application | Application loads without console error | Screenshot |
| FE-02 | Frontend | Open `/`, `/login`, `/barcode-test` | All routes render | Screenshots |
| FE-03 | Frontend | `npm run typecheck` | Pass | Terminal output |
| FE-04 | Frontend | `npm run build` | Production build succeeds | Terminal output |
| AUTH-01 | Auth | Login with valid seeded account | Access and refresh tokens returned | Redacted response |
| AUTH-02 | Auth | Login with wrong password | HTTP 401 standard error | Captured response |
| AUTH-03 | Auth | Access protected route without token | HTTP 401 | Captured response |
| AUTH-04 | Auth | Viewer calls admin-only route | HTTP 403 | Captured response |
| AUTH-05 | Auth | Refresh valid token | Old refresh token rotates | Integration test |
| AUTH-06 | Auth | Reuse rotated refresh token | HTTP 401 | Integration test |
| BC-01 | Barcode | Validate known EAN-13/UPC-A values | Valid checksum accepted | Automated test |
| BC-02 | Barcode | Submit the same scanned value repeatedly | Each submission remains lookup-only | Screen recording |
| BC-03 | Barcode | Scan a real beverage using a USB/Bluetooth keyboard-wedge scanner | Input receives the printed value plus Enter | Hardware-scanner evidence |
| BC-04 | Barcode | Optionally scan a real beverage with a phone camera | Value matches printed barcode | Optional phone evidence |
| BC-05 | Barcode | Scan without confirming movement | Inventory remains unchanged | API/DB evidence |

## Test levels

- Unit tests cover validation, token claims, RBAC decisions, barcode checksums,
  and service behavior with mocks.
- Integration tests cover PostgreSQL migrations, refresh-token rotation,
  constraints, transactions, and API/database behavior.
- Smoke tests verify Compose startup, readiness, routes and frontend proxying.
- Manual exploratory tests cover hardware-scanner input, real devices,
  responsive layouts, accessibility and scanner ergonomics. Camera permission
  testing is optional.

## Defect severity

| Severity | Meaning |
| --- | --- |
| Critical | Security bypass, data loss, incorrect inventory or unusable system |
| High | Core acceptance criterion fails with no practical workaround |
| Medium | Feature works incorrectly but a safe workaround exists |
| Low | Cosmetic, wording or low-impact usability problem |

An issue is Done only after its acceptance criteria pass, evidence is attached,
the pull request is reviewed and merged, and no Critical or High defect remains.
