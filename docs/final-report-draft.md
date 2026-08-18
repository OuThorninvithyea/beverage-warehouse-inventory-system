# BWIMS final report (draft — living document)

**Status: DRAFT.** This is being written incrementally as the project
progresses, not reconstructed from memory at the end. Sections marked
`[TO FILL AT PROJECT END]` are intentionally left open — writing them now
would mean guessing at outcomes that haven't happened yet.

**Last updated:** 2026-08-18 (Week 11 of 15)

## 1. Project summary

BWIMS (Beverage Warehouse Inventory Management System) is a full-stack web
application for beverage distributors: multi-warehouse inventory tracking,
lot/batch expiry management, FEFO picking, and barcode-driven floor
operations. Full context: `Beverage Warehouse Inventory Management System
Proposal` (Google Doc, linked from `README.md`).

## 2. What was delivered (as of this update)

### Backend — delivered

| Module | Scope | Evidence |
| --- | --- | --- |
| Foundation | Go/Fiber server, PostgreSQL schema, Docker Compose, health checks | `docs/week-6-foundation-validation.md` |
| Auth | RS256 JWT, refresh-token rotation, 4-role RBAC | `docs/week-7-validation.md` |
| Warehouses & Locations | Full CRUD, hierarchical locations | PR #10 |
| Catalog | Categories, Products, barcode lookup (UPC-A/EAN-13) | `docs/week-8-catalog-validation.md`, PR #11 |
| Admin User Management | CRUD, password reset, last-admin lockout, self-protection | `docs/week-9-user-management-validation.md`, PR #12 |
| Inventory & Movements | Balances, lots, receive, FEFO pick, transfer, adjust, FIFO cost layers | `docs/week-10-inventory-validation.md`, PR #13 |
| FR-19 expired-lot policy | Allow-with-audit-flag (revised from proposal's "prevent") | PR #14 |

All of the above have real automated tests (unit, HTTP, and PostgreSQL
integration tests) and are either merged to `main` or in open, CI-passing
pull requests. This is not a claim of "done" without evidence — every row
above links to a validation document with actual command output and dates.

### Backend — not delivered

- Low-stock and expiry alerts (FR-11, FR-12)
- Barcode label generation (FR-20)
- Reporting: inventory valuation, expiry reports, movement summaries,
  product velocity, CSV export (FR-23–27)
- Dashboard aggregation endpoints (FR-28)

### Frontend

Not started. Login page and application shell exist from Week 7; no
screens exist for warehouses, catalog, users, or inventory — despite all
of those having working backend APIs since Weeks 8–10. `docs/figma-ai-prompts.md`
contains ready-to-use design prompts for all 12 planned screens, generated
2026-08-18, not yet run through a design tool.

### Documentation

- `docs/api-contract.md` — endpoint reference, kept current with each merged module
- `docs/database-erd.md` — schema documentation
- `docs/requirements-traceability.md` — FR/NFR status against the proposal
- `docs/deployment-guide.md` — how to run the system (this document's sibling)
- Week-by-week validation docs with real test output

### Testing

Backend: unit tests, HTTP handler tests, and real PostgreSQL integration
tests per module (not mocked). Frontend: 4 Vitest component tests exist
(unchanged since Week 6); no additional frontend tests, since no additional
frontend screens exist yet. No E2E (Playwright) tests exist — the proposal
scoped these to Phase 6, not yet reached.

## 3. Deviations from the proposal

Documented so a reader comparing this report to the proposal doesn't find
unexplained gaps:

1. **FR-19 policy reversed.** The proposal's Functional Requirements table
   says FEFO pick logic "prevents picking expired lots." The team's actual
   decision (2026-08-18) was the opposite: allow picking an expired lot,
   but write an immutable audit record (`EXPIRED_LOT_PICK` in
   `audit_records`) instead of blocking it. Rationale: a hard block has no
   escape hatch for legitimate disposal/write-off workflows; an audit flag
   preserves visibility without operational friction. See
   `docs/requirements-traceability.md`'s "Known proposal clarifications"
   section for the full note.
2. **Stock adjustment reason is free text, not a category.** The proposal
   implies a structured reason (damage/spillage/expiry write-off) for
   FR-16. The implementation uses a free-text `notes` field instead.
3. **sqlc was not used**, despite being named in the proposal's technology
   stack and in Linear issue PRO-13's acceptance criteria. Every backend
   module uses hand-written `pgx` queries in its own `repository.go`
   instead. This was a direction change made during implementation, not an
   oversight — flagged on PRO-13 in Linear.

## 4. Schedule vs. the Gantt chart

As of Week 11 (this update), the following tasks were scheduled to
complete by Week 8 or Week 10 and have not: 2.3 (Kong, domain SQL/seed/
indexes, 30%), 2.4 (Chanraksa, core CRUD frontend, 35%), 2.5 (Hong, core UI
components, 30%), 2.6 (Sinat, QA/docs, 50%), 2.7 (Phal, API integration
testing, 0%), 3.3 (Kong, inventory locking/perf, 0%), 3.4 (Chanraksa,
inventory frontend, 0%), 3.5 (Hong, mobile UX, 0%), 3.6 (Sinat, movement
QA, 0%). The backend track (rows 2.1, 2.2, 3.1, 3.2) is current or ahead.
Live tracking: the project Gantt chart, linked from the proposal's Section
14, and Linear project "Beverage Warehouse Inventory Management System."

## 5. Risks realized or mitigated (vs. proposal Section 11)

| Risk (from proposal) | Status |
| --- | --- |
| R1 Concurrent stock inconsistency | Mitigated — row-level `FOR UPDATE` locking implemented in all four movement types, verified by transaction-scoped repository tests |
| R2 Incorrect FEFO/expiry handling | Mitigated — integration test explicitly proves FEFO (physical) and FIFO (cost) diverge correctly in a mixed-lot scenario |
| R3 Auth/security failure | Mitigated — RS256, bcrypt, RBAC tested per role per endpoint |
| R4 Hardware scanner incompatibility | **Unresolved** — real hardware evidence still not captured (issue #6); this is the one risk from the register that remains genuinely open |
| R5 Schedule slippage | **Realized** — see Section 4 above; nine tasks are past their scheduled week |
| R6 Deployment/hosting failure | Not yet applicable — no deployment has been attempted |

## 6. Lessons learned so far

`[TO FILL AT PROJECT END — write this from what actually happened, not aspirationally]`

Candidates to revisit when finishing this section: the test-database
pollution incident (an inventory-module integration test accidentally used
the `admin` role for its throwaway test user, which — combined with
`stock_movements`' append-only trigger making that user permanently
undeletable — silently broke an unrelated test's "last active admin"
invariant across ~7 test runs before being caught and fixed); the backend
racing ahead of frontend by several weeks; and whether "allow with audit
flag" for FR-19 held up once evaluated against real usage.

## 7. Remaining work

See `docs/requirements-traceability.md` for the authoritative FR/NFR list.
Summary: frontend (all screens), reporting/alerts backend, barcode label
generation, physical scanner hardware evidence, E2E tests, and an actual
deployment.

## 8. Conclusion

`[TO FILL AT PROJECT END]`
