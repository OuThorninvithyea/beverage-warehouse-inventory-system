# Gantt reconciliation — Week 15 (deadline week)

The Google Gantt chart is the schedule source of truth, but it stopped being
updated: its `CURRENT PROJECT WEEK` cell still reads **7** while the project is
at its **Week 15** deadline, and its phase rollups disagree with their own
children (Phase 3 shows 0% while tasks 3.1 and 3.2 are both 100%).

This document reconciles every WBS row against what is actually in the
repository, so the chart, `requirements-traceability.md` and the final report
agree. Apply the values in the last section to the sheet.

**Reconciled on:** 2026-09-22
**Overall completion:** ~66% across the 24 leaf tasks (the chart's stale values
average ~51%)

## Row-by-row assessment

| WBS | Task | Chart | Actual | Evidence and what is missing |
| --- | --- | --- | --- | --- |
| 0.1 | Requirements, acceptance criteria, test plan & tracking | 100% | 100% | `qa-test-plan.md`, `requirements-traceability.md` |
| 0.2 | Technical scope, API contract & backend planning | 100% | 100% | `api-contract.md`, `plan.md` |
| 0.3 | User journeys, UI requirements & initial wireframes | 100% | 100% | `user-journeys-and-wireframes.md` |
| 1.1 | Backend architecture and Go/Fiber scaffold | 100% | 100% | `internal/app`, `internal/platform`, module pattern |
| 1.2 | ERD, PostgreSQL schema, migrations, sqlc & Docker | 100% | 100% | `database-erd.md`, 5 migrations, Docker Compose. sqlc replaced by hand-written pgx — documented deviation, not a gap |
| 1.3 | Vue, TypeScript, PrimeVue, Pinia & routing scaffold | 100% | 100% | Frontend app shell, router guards, Pinia stores |
| 1.4 | Hardware barcode requirements, device research & API test setup | 90% | 90% | `barcode-validation.md` library comparison. Missing: physical device evidence |
| 2.1 | JWT, refresh-token and RBAC backend | 100% | 100% | `week-7-validation.md`, auth module tests |
| 2.2 | User, warehouse, location and product APIs | 100% | 100% | Weeks 8–9 validation docs, PRs #10–#12 |
| 2.3 | Domain SQL queries, seed data and indexes | 30% | **90%** | Indexes and constraints in migrations 000003–000005; demo seed data in `cmd/seed` with derived balances and cost layers; queries live in per-module repositories instead of sqlc |
| 2.4 | Login, navigation and core CRUD frontend | 35% | **50%** | Login, app shell, warehouses and locations screens. Missing: products, categories, users screens |
| 2.5 | Core UI components and responsive layouts | 30% | **50%** | Tailwind migration, Figma-aligned cards, form dialogs, responsive breakpoints in 3 views. Missing: component set for inventory screens |
| 2.6 | Authentication/CRUD QA, API documentation & tracking | 50% | **80%** | `api-contract.md` covers all 10 endpoint groups including inventory; week 6–10 validation docs. Missing: QA sign-off for weeks 11+ |
| 2.7 | API integration and acceptance testing | 0% | **10%** | 4 PostgreSQL integration test suites exist, but they are per-module repository tests, not the cross-module acceptance pass this row describes |
| 3.1 | Lots, inventory snapshots and receive workflow | 100% | 100% | `week-10-inventory-validation.md`, PR #13 |
| 3.2 | FEFO pick, transfer, adjust and FIFO costing | 100% | 100% | Same, plus FR-19 audit flag in PR #14 |
| 3.3 | Inventory locking, constraints and performance | 0% | **70%** | Row-level `FOR UPDATE` locking in all four movement types (5 sites in `inventory/repository.go`), non-negative and reserved-quantity constraints, append-only triggers. Missing: the load test for NFR-1 |
| 3.4 | Inventory and stock-movement frontend screens | 0% | 0% | Correct. No screens exist despite the APIs being live since Week 10 |
| 3.5 | Mobile warehouse UX and inventory indicators | 0% | **10%** | Responsive breakpoints only. No mobile flows, no stock indicators |
| 3.6 | Movement QA, traceability and documentation | 0% | **60%** | `week-10-inventory-validation.md`, traceability rows FR-10 to FR-19, API contract section. Missing: QA execution record against the movement flows |
| 4.1 | Hardware barcode and scan-flow integration | 0% | **60%** | `/barcode-test` route with camera, manual and USB paths; checksum validation; seeded barcodes; printable verified test sheet. Missing: real-device evidence and scan-to-operation wiring |
| 4.2 | Expiry alerts and reporting APIs | 0% | 0% | Correct. FR-12 and the reporting endpoints are not started |
| 5.1 | Frontend integration, dashboard, E2E tests & responsive polish | 0% | 0% | Correct. The dashboard still renders hardcoded placeholder counts; no E2E suite |
| 6.1 | Deployment, UAT, final documentation & presentation | 0% | **20%** | `deployment-guide.md` and the living final report exist. Nothing has been deployed; no UAT has run |

## Recalculated phase rollups

Simple average of each phase's children:

| WBS | Phase | Chart | Actual |
| --- | --- | --- | --- |
| 0 | Planning & Requirements | 100% | 100% |
| 1 | System Design & Foundation | 98% | 98% |
| 2 | Authentication & Core Domain | 35% | **69%** |
| 3 | Inventory & Stock Movements | 0% | **57%** |
| 4 | Barcode & Reporting | 0% | **30%** |
| 5 | Frontend Integration & Testing | 0% | 0% |
| 6 | Deployment & Documentation | 0% | **20%** |

## What the corrected chart shows

The backend track finished on or ahead of schedule. Everything still open sits
in one of three places:

1. **Frontend delivery** (3.4, 3.5, 5.1, and the remainder of 2.4 and 2.5) —
   the largest gap; eight of twelve planned screens do not exist.
2. **Reporting and alerts** (4.2) — not started.
3. **Closing activities** (2.7, 4.1 evidence, 6.1) — acceptance testing, the
   physical scanner evidence, deployment and UAT.

Risk R5 (schedule slippage) from the proposal register is fully realized: three
of seven phases are past their end week with no work recorded against them.

## Values to apply to the sheet

Column **H** is `PCT COMPLETE`. Paste this block into **H12** and it fills rows
12–42 in order:

```
100%
100%
100%
100%
98%
100%
100%
100%
90%
69%
100%
100%
90%
50%
50%
80%
10%
57%
100%
100%
70%
0%
10%
60%
30%
60%
0%
0%
0%
20%
20%
```

Also update:

- **D6** `CURRENT PROJECT WEEK`: `7` → `15`
- The timeline header label `WEEK 6 • CURRENT` should move to the Week 15
  column, which is already marked `WEEK 15 • DEADLINE`.

After applying these, re-check that this document, the chart and
`requirements-traceability.md` still tell the same story before the final
report is submitted.
