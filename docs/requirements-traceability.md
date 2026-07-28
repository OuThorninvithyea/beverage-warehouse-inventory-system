# BWIMS requirements traceability

This register connects the approved proposal to the Week 6–15 Gantt,
implementation issues, tests, and evidence. Status means implementation status,
not GitHub issue status.

## Functional requirements

| Requirement | Proposal outcome | Gantt work | GitHub / evidence | Verification | Status |
| --- | --- | --- | --- | --- | --- |
| FR-1 | JWT authentication and refresh tokens | 2.1 | Week 7 auth module | Auth service and HTTP tests | Implemented |
| FR-2 | Four-role RBAC | 2.1 | Week 7 auth module | Role middleware tests | Implemented |
| FR-3 | Admin user management | 2.2 | Week 8 domain APIs | API integration tests | Planned |
| FR-4 | Warehouse management | 2.2 | Week 7–8 domain APIs | API integration tests | Planned |
| FR-5 | Hierarchical locations with barcodes | 1.2, 2.2 | `database-erd.md` | Migration plus CRUD tests | Schema ready |
| FR-6 | Product CRUD | 2.2 | Week 7–8 domain APIs | API integration tests | Planned |
| FR-7 | Categories | 1.2, 2.2 | `database-erd.md` | Migration plus CRUD tests | Schema ready |
| FR-8 | Barcode product lookup | 1.4, 2.2 | `barcode-validation.md` | EAN validation and API tests | Test setup ready |
| FR-9 | Product search | 2.2 | Week 8 domain APIs | Search and pagination tests | Planned |
| FR-10 | Inventory balances | 1.2, 3.1 | `inventory_balances` | Constraint and movement tests | Schema ready |
| FR-11 | Lot and expiry tracking | 1.2, 3.1 | `lots` | Expiry ordering tests | Schema ready |
| FR-12 | Expiry alerts | 4.2 | Week 11–12 reporting | Alert API tests | Planned |
| FR-13 | Transactional receive | 3.1 | Week 9 inventory module | Transaction integration tests | Planned |
| FR-14 | FEFO pick | 3.2 | Week 9–10 movement module | Mixed-lot ordering tests | Planned |
| FR-15 | Transfer stock | 3.2 | Week 9–10 movement module | Atomic source/destination test | Planned |
| FR-16 | Stock adjustment | 3.2 | Week 9–10 movement module | Reason and audit tests | Planned |
| FR-17 | Movement history | 1.2, 3.6 | Immutable movement schema | Read and immutability tests | Schema ready |
| FR-18 | FIFO cost layers | 1.2, 3.2 | `cost_layers` | Oldest-layer consumption tests | Schema ready |
| FR-19 | Prevent expired-lot picking | 3.2 | Week 9–10 movement service | Boundary-date tests | Planned |
| FR-20 | Barcode label generation | 4.1 | Week 11 barcode phase | Image response tests | Planned |
| FR-21 | Hardware barcode scanning with optional phone camera | 1.4, 4.1 | `/barcode-test` | Real USB/Bluetooth scanner evidence | Implementation ready |

## Non-functional requirements

| Requirement | Target | Design / evidence | Verification |
| --- | --- | --- | --- |
| NFR-1 Performance | API p95 below 200 ms | pgx pool, indexes, bounded pagination | Load test before Week 13 |
| NFR-2 Reliability | Atomic stock operations | PostgreSQL transactions and constraints | Failure/rollback integration tests |
| NFR-3 Security | RS256 JWT, bcrypt, rate limiting, HTTPS | Week 7 auth and deployment configuration | Auth negative tests and security review |
| NFR-4 Maintainability | Modular, testable architecture | Route → Handler → Service → Repository | Tests, vet and code review |
| NFR-5 Usability | Scan-to-operation below 10 seconds | Focused hardware-scanner input and confirmation flow | Timed hardware-scanner test |
| NFR-6 Auditability | Immutable history | Append-only movement and audit triggers | Update/delete rejection tests |
| NFR-7 Accessibility | Correct role boundaries and usable UI | RBAC plus responsive/accessibility notes | Keyboard, contrast and role tests |

## Known proposal clarifications

- FEFO selects the physical lot with the nearest valid expiry date.
- FIFO consumes accounting cost layers by receipt time. FEFO and FIFO are
  separate rules and must not be merged.
- Temperature zones are recorded as location metadata; automated IoT
  temperature monitoring is outside the MVP.
- A barcode scan only produces a lookup value. Inventory changes require an
  authenticated, validated and explicitly confirmed movement request.
- The live Gantt chart, not the proposal's older phase numbering, controls the
  Week 6–15 delivery schedule.
