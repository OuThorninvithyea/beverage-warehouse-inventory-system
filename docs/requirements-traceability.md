# BWIMS requirements traceability

This register connects the approved proposal to the Week 6–15 Gantt,
implementation issues, tests, and evidence. Status means implementation status,
not GitHub issue status.

## Functional requirements

| Requirement | Proposal outcome | Gantt work | GitHub / evidence | Verification | Status |
| --- | --- | --- | --- | --- | --- |
| FR-1 | JWT authentication and refresh tokens | 2.1 | Week 7 auth module | Auth service and HTTP tests | Implemented |
| FR-2 | Four-role RBAC | 2.1 | Week 7 auth module | Role middleware tests | Implemented |
| FR-3 | Admin user management | 2.2 | Week 9 users module | Service, handler and PostgreSQL repository tests | Implemented |
| FR-4 | Warehouse management | 2.2 | PRO-8 warehouse module | Service, handler and PostgreSQL repository tests | Implemented |
| FR-5 | Hierarchical locations with barcodes | 1.2, 2.2 | PRO-8 warehouse module and `database-erd.md` | Migration, scoped CRUD and barcode-conflict tests | Implemented |
| FR-6 | Product CRUD | 2.2 | PRO-9 catalog module | Service, handler and PostgreSQL lifecycle tests | Implemented |
| FR-7 | Categories | 1.2, 2.2 | PRO-9 catalog module and `database-erd.md` | Hierarchy, cycle, conflict, RBAC and CRUD tests | Implemented |
| FR-8 | Barcode product lookup | 1.4, 2.2 | PRO-9 barcode endpoint and `barcode-validation.md` | UPC-A/EAN-13 checksum, route-order and active lookup tests | Implemented |
| FR-9 | Product search | 2.2 | PRO-9 catalog module | Search/filter and cursor-pagination tests | Implemented |
| FR-10 | Inventory balances | 1.2, 3.1 | Week 9-10 inventory module | Repository and integration tests | Implemented |
| FR-11 | Lot and expiry tracking | 1.2, 3.1 | Week 9-10 inventory module | Repository and integration tests | Implemented |
| FR-12 | Expiry alerts | 4.2 | `GET /api/v1/inventory/alerts` and the Expiry Alerts screen | Service, handler, route-ordering and PostgreSQL integration tests | Implemented |
| FR-13 | Transactional receive | 3.1 | Week 9-10 inventory module | Transaction integration tests | Implemented |
| FR-14 | FEFO pick | 3.2 | Week 9-10 inventory module | Mixed-lot ordering tests | Implemented |
| FR-15 | Transfer stock | 3.2 | Week 9-10 inventory module | Atomic source/destination test | Implemented |
| FR-16 | Stock adjustment | 3.2 | Week 9-10 inventory module | Reason and audit tests | Implemented |
| FR-17 | Movement history | 1.2, 3.6 | Immutable movement schema | Read and immutability tests | Implemented |
| FR-18 | FIFO cost layers | 1.2, 3.2 | Week 9-10 inventory module | Oldest-layer consumption tests | Implemented |
| FR-19 | Expired-lot picking policy | 3.2 | Week 9-10 inventory module | Boundary-date and audit-record tests | Implemented (policy revised) |
| FR-20 | Barcode label generation | 4.1 | Week 11 barcode phase | Image response tests | Planned |
| FR-21 | Phone-camera barcode scanning | 1.4, 4.1 | `/barcode-test` | Real device/browser evidence | Implementation ready |

## Non-functional requirements

| Requirement | Target | Design / evidence | Verification |
| --- | --- | --- | --- |
| NFR-1 Performance | API p95 below 200 ms | pgx pool, indexes, bounded pagination | Load test before Week 13 |
| NFR-2 Reliability | Atomic stock operations | PostgreSQL transactions and constraints | Failure/rollback integration tests |
| NFR-3 Security | RS256 JWT, bcrypt, rate limiting, HTTPS | Week 7 auth and deployment configuration | Auth negative tests and security review |
| NFR-4 Maintainability | Modular, testable architecture | Route → Handler → Service → Repository | Tests, vet and code review |
| NFR-5 Usability | Scan-to-operation below 10 seconds | Responsive scanner and confirmation flow | Timed phone test |
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
- FR-19 was originally titled "prevent expired-lot picking." On 2026-08-18 the
  project owner explicitly changed this policy to "allow with an audit flag":
  picking an already-expired lot succeeds (needed for disposal/internal-use
  workflows) but writes an immutable `EXPIRED_LOT_PICK` entry to
  `audit_records` for later review, rather than being silently permitted or
  hard-blocked. The row name was updated to match; the original "prevent"
  wording no longer reflects the implemented behavior.
