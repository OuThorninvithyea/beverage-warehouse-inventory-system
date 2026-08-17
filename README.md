# Beverage Warehouse Inventory Management System

BWIMS is a full-stack inventory system for beverage distributors. It is designed for multi-warehouse stock control, barcode scanning, lot and expiry tracking, FEFO picking, FIFO valuation, stock movements, auditability, and reporting.

## Project timeline

Development is planned from **Week 6 through Week 15**. The approved Google Gantt chart is the schedule source of truth; GitHub issues and pull requests are the day-to-day evidence of progress.

| Period | Delivery focus |
| --- | --- |
| Week 6 | Project foundation, ERD, API contract, UI wireframes, QA plan, barcode research |
| Weeks 7–8 | Authentication, RBAC, users, warehouses, locations, products and core UI |
| Weeks 9–10 | Inventory, lots, receive, FEFO pick, transfer, adjustment and FIFO costing |
| Weeks 11–12 | Expiry alerts, reporting APIs, dashboards and integration |
| Week 13 | End-to-end integration, responsive improvements and stabilization |
| Weeks 14–15 | QA, UAT, documentation, deployment and final presentation |

## Team responsibilities

| Member | Primary responsibility |
| --- | --- |
| Ou Thorninvithyea | Project manager and backend lead |
| Sinat Chantha | QA, documentation and progress tracking |
| Chanraksa | Frontend lead |
| Kong Dymond | Database and DevOps |
| Hong | UI/UX and responsive design |
| Phal Monyvan | Barcode integration and integration testing |

## Proposed technology

- Backend: Go, Fiber v3, pgx and sqlc
- Database: PostgreSQL
- Frontend: Vue 3, TypeScript, PrimeVue, Pinia and Vite
- Development environment: Docker Compose
- Delivery workflow: GitHub Projects, issues, branches, pull requests and Slack updates

## Repository workflow

1. Select an issue from the current weekly iteration.
2. Move it from `Ready` to `In Progress`.
3. Create a branch such as `feat/12-health-endpoint`.
4. Implement and test only the issue scope.
5. Open a pull request linked to the issue.
6. Move the issue through `Code Review` and `QA Testing`.
7. Merge only after the acceptance criteria and Definition of Done are met.

Detailed working agreements are in [CONTRIBUTING.md](CONTRIBUTING.md), and the progress workflow is in [docs/project-management.md](docs/project-management.md).

Backend setup and architecture are documented in [backend/README.md](backend/README.md),
and frontend setup is documented in [frontend/README.md](frontend/README.md).
The [database ERD](docs/database-erd.md) and
[API contract](docs/api-contract.md) define the shared backend, frontend, and QA
foundation. The latest command evidence is recorded in
[Week 6 foundation validation](docs/week-6-foundation-validation.md).

## Current Week 7 goal

Week 6 foundation work is implemented and Week 7 completes authentication:

- Docker Compose starts the backend and PostgreSQL.
- Database migrations run successfully.
- `GET /health` returns HTTP 200.
- The Vue frontend loads.
- The API response format is agreed.
- ERD, wireframes, acceptance criteria and the QA plan are ready.
- RS256 JWT login, rotating refresh tokens and four-role RBAC are working.
- The barcode test route is hardware-scanner first and ready for physical
  USB/Bluetooth scanner evidence. Phone-camera scanning is optional.

The real beverage hardware scan remains a manual evidence requirement and is
not marked complete until the printed and captured barcode values are recorded.

## Documents

- `Beverage_Warehouse_Proposal.pdf` — project proposal
- `beverage_standalone.html` and `proposal.html` — proposal source/export versions
- `plan.md` — detailed technical backlog
- `docs/requirements-traceability.md` — proposal-to-test traceability
- `docs/qa-test-plan.md` — QA strategy and Week 6–7 acceptance tests
- `docs/user-journeys-and-wireframes.md` — approved journeys and wireframes
- `docs/barcode-validation.md` — hardware-scanner decision and evidence form

Do not commit secrets. Copy future `.env.example` files to `.env` locally and keep real credentials outside Git.
