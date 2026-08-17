# Warehouse Inventory Management System — Implementation Plan

## Executive Summary

**Stack:** Go + Fiber v3 → sqlc/pgx → PostgreSQL || Vue 3 + TypeScript + PrimeVue + Vite
**Scale:** Mid-size, multi-warehouse, barcode scanning, role-based access
**Estimated total effort:** 8–12 weeks (single developer) / 4–6 weeks (2-person team)

---

## 1. Project Phases & Timeline

### Phase 0: Foundation (Week 1)
**Goal:** Project scaffolding, database schema, migrations, CI/CD

| Task | Files | Estimate |
|------|-------|----------|
| Initialize Go module + Fiber v3 server | `go.mod`, `cmd/server/main.go` | 2h |
| PostgreSQL schema (all tables) | `migrations/001_initial_schema.up.sql` | 4h |
| sqlc config + codegen setup | `sqlc.yaml`, `internal/repository/` | 2h |
| golang-migrate integration | `cmd/server/main.go` (auto-migrate) | 1h |
| Configuration (env vars) | `internal/config/config.go` | 1h |
| Health check endpoint | `GET /health` | 0.5h |
| Docker Compose (Postgres + API) | `docker-compose.yml` | 1h |
| **Milestone:** Server starts, connects to DB, runs migrations, `/health` returns 200 | | |

### Phase 1: Auth + Users (Week 1–2)
**Goal:** JWT auth, role-based access, user management

| Task | Endpoints | Estimate |
|------|-----------|----------|
| User model + registration | `POST /api/auth/register` | 2h |
| JWT login (RS256) | `POST /api/auth/login` | 2h |
| Token refresh | `POST /api/auth/refresh` | 1h |
| JWT middleware | `internal/middleware/auth.go` | 2h |
| Role middleware (admin/manager/picker/viewer) | `internal/middleware/rbac.go` | 2h |
| User CRUD (admin only) | `GET/POST/PUT/DELETE /api/users` | 3h |
| Password hashing + validation | `internal/service/user.go` | 1h |
| Login page (Vue) | `frontend/src/views/Login.vue` | 3h |
| Pinia auth store | `frontend/src/stores/auth.ts` | 1h |
| Route guards | `frontend/src/router/guards.ts` | 1h |
| **Milestone:** Users can register, login, receive JWT, protected routes enforce roles | | |

### Phase 2: Core Domain — Warehouses, Products, Inventory (Week 2–4)
**Goal:** Full CRUD for core domain entities, stock tracking

#### Warehouses & Locations (3 days)
| Task | Endpoints | Estimate |
|------|-----------|----------|
| Warehouse CRUD | `GET/POST/PUT/DELETE /api/warehouses` | 2h |
| Location CRUD (zone/aisle/rack/bin) | `GET/POST/PUT/DELETE /api/warehouses/:id/locations` | 3h |
| Warehouse list page (Vue) | `frontend/src/views/Warehouses/` | 3h |
| Location tree component | `frontend/src/components/LocationTree.vue` | 2h |

#### Products & Categories (3 days)
| Task | Endpoints | Estimate |
|------|-----------|----------|
| Category CRUD | `GET/POST/PUT/DELETE /api/categories` | 1.5h |
| Product CRUD (SKU, name, unit, barcode) | `GET/POST/PUT/DELETE /api/products` | 3h |
| Barcode lookup | `GET /api/products/by-barcode/:code` | 1h |
| Product search (SKU, name, barcode) | `GET /api/products?q=&category_id=` | 1h |
| Product list page (PrimeVue DataTable) | `frontend/src/views/Products/` | 4h |
| Product form (VeeValidate + Zod) | `frontend/src/views/Products/ProductForm.vue` | 2h |

#### Inventory (4 days)
| Task | Endpoints | Estimate |
|------|-----------|----------|
| Stock snapshot query (by warehouse, product, location) | `GET /api/inventory?warehouse_id=&product_id=&location_id=` | 2h |
| Stock detail (single item + movements history) | `GET /api/inventory/:stock_id` | 1h |
| Low stock alert query | `GET /api/inventory/alerts?threshold=` | 1h |
| Inventory summary (per warehouse) | `GET /api/inventory/summary?warehouse_id=` | 1h |
| Inventory dashboard (Vue) | `frontend/src/views/Dashboard.vue` | 4h |
| Inventory detail page | `frontend/src/views/Inventory/` | 3h |
| Low stock widgets | `frontend/src/components/LowStockAlert.vue` | 2h |
| **Milestone:** Full warehouse/product/inventory CRUD working, searchable, with Vue UI | | |

### Phase 3: Stock Movements (Week 4–5)
**Goal:** The core system — receive, pick, transfer, adjust inventory with audit trail

| Task | Endpoints | Estimate |
|------|-----------|----------|
| Receive inventory | `POST /api/movements/receive` | 3h |
| Pick inventory | `POST /api/movements/pick` | 2h |
| Transfer between locations | `POST /api/movements/transfer` | 3h |
| Stock adjustment (count correction) | `POST /api/movements/adjust` | 2h |
| Movement history query | `GET /api/movements?item_id=&movement_type=&from=&to=` | 1.5h |
| Movement detail | `GET /api/movements/:id` | 0.5h |
| FIFO cost layer consumption (in pick) | `internal/service/cost.go` | 2h |
| Transactional stock update + movement insert | `internal/service/movement.go` | 2h |
| Receive inventory form (Vue) | `frontend/src/views/Movements/Receive.vue` | 3h |
| Pick inventory form (with barcode scan) | `frontend/src/views/Movements/Pick.vue` | 3h |
| Transfer form (source → destination) | `frontend/src/views/Movements/Transfer.vue` | 3h |
| Movement history page (audit trail) | `frontend/src/views/Movements/History.vue` | 3h |
| **Milestone:** All four movement types work, stock quantities are correct, audit trail is immutable | | |

### Phase 4: Barcode Scanning (Week 5–6)
**Goal:** Phone camera scanning + USB scanner integration

| Task | Estimate |
|------|----------|
| Integrate `@teckel/vue-barcode-reader` | 2h |
| Barcode scan component (camera) | 3h |
| Barcode → product lookup flow | 1h |
| Scan-to-receive flow (scan → lookup → receive) | 2h |
| Scan-to-pick flow | 2h |
| Barcode label generation (Go: boombuler) | `GET /api/barcode/generate/:product_id` | 3h |
| USB scanner input handling (keyboard wedge) | `frontend/src/composables/useScanner.ts` | 1h |
| **Milestone:** Scan a barcode → product appears → can receive/pick from same screen | | |

### Phase 5: Reporting & Dashboard (Week 6–7)
**Goal:** KPIs, exports, inventory valuation

| Task | Endpoints | Estimate |
|------|-----------|----------|
| Inventory valuation (FIFO) | `GET /api/reports/valuation?warehouse_id=` | 2h |
| Stock movement summary (by type, date range) | `GET /api/reports/movement-summary` | 1.5h |
| Product velocity (top movers) | `GET /api/reports/velocity` | 1.5h |
| Export to CSV | `GET /api/reports/export/:type?format=csv` | 1h |
| Dashboard page (KPIs + charts) | `frontend/src/views/Dashboard.vue` (enhance) | 4h |
| Reports page | `frontend/src/views/Reports/` | 3h |
| Chart.js or ApexCharts integration | `frontend/src/composables/useCharts.ts` | 2h |
| **Milestone:** Dashboard shows active stock, low stock alerts, recent movements, valuation | | |

### Phase 6: Polish & Hardening (Week 7–8)
| Task | Estimate |
|------|----------|
| Comprehensive input validation (Go + Zod) | 3h |
| Error handling standardization | 2h |
| Rate limiting | 1h |
| Request logging middleware | 1h |
| Vue error boundaries + toast notifications | 2h |
| Loading states + skeleton screens | 2h |
| Responsive design (mobile for floor staff) | 4h |
| Integration tests (Go — database layer) | 4h |
| Component tests (Vue — Vitest) | 3h |
| E2E test (Playwright — critical paths) | 4h |
| **Milestone:** Production-ready, tested, mobile-responsive | | |

---

## 2. Architecture

### Backend (Go + Fiber v3)

```
cmd/server/main.go                 # Entry point, DI wiring
internal/
  config/config.go                  # Env vars, config struct
  handler/                          # HTTP handlers (thin)
    auth.go, user.go, warehouse.go, product.go,
    inventory.go, movement.go, report.go, barcode.go
  service/                          # Business logic
    auth.go, user.go, warehouse.go, product.go,
    inventory.go, movement.go, cost.go, report.go
  repository/                       # sqlc-generated + custom queries
    gen/                            # sqlc output (do not edit)
    custom/                         # Hand-written complex queries
  middleware/
    auth.go, rbac.go, logging.go, ratelimit.go
  router/router.go                  # Route registration
pkg/
  entity/                           # Shared domain structs
  api/                              # Request/Response DTOs
  lib/                              # Helpers (barcode, pagination, etc.)
migrations/                         # *.up.sql, *.down.sql
```

**Layers:** Handler → Service → Repository → PostgreSQL
**DI pattern:** Constructor injection (no DI framework)
**Error model:** `type AppError struct { Code int; Message string; Err error }` with global `ErrorHandler` in Fiber config

### Frontend (Vue 3 + TypeScript + Vite)

```
frontend/src/
  api/                              # Axios instance + interceptors
    client.ts, auth.ts, products.ts, inventory.ts, movements.ts
  assets/
  components/                       # Reusable components
    ui/                              # Wrappers around PrimeVue
    LocationTree.vue, BarcodeScanner.vue,
    StockMovementForm.vue, LowStockAlert.vue
  composables/                      # Reusable logic
    useAuth.ts, useScanner.ts, useCharts.ts,
    usePagination.ts, useWarehouse.ts
  layouts/
    DefaultLayout.vue, AuthLayout.vue
  router/
    index.ts, guards.ts
  stores/                           # Pinia stores
    auth.ts, products.ts, inventory.ts,
    warehouses.ts, movements.ts, barcode.ts
  types/                            # TypeScript interfaces
  views/                            # Page components
    auth/Login.vue, ForgotPassword.vue
    Dashboard.vue
    Warehouses/List.vue, Detail.vue
    Products/List.vue, ProductForm.vue
    Inventory/Index.vue, Detail.vue
    Movements/Receive.vue, Pick.vue, Transfer.vue, History.vue
    Reports/Index.vue
    Users/List.vue, UserForm.vue
  App.vue
  main.ts
```

### Database Schema (Key Tables)

```
warehouses          — id, name, code, address, is_active
locations           — id, warehouse_id, zone, aisle, rack, shelf, barcode, is_pickable
categories          — id, name, parent_id
products            — id, sku, barcode (UNIQUE), name, category_id, unit, is_lot_tracked
lots                — id, product_id, lot_number, expiration_date
stock               — id, warehouse_id, location_id, product_id, lot_id, qty, reserved_qty
                      UNIQUE(warehouse_id, location_id, product_id, lot_id)
stock_movements     — id, type ENUM(receive|pick|transfer|adjust), product_id, lot_id,
                      from_warehouse_id, to_warehouse_id, from_location_id, to_location_id,
                      qty, reference_type, reference_id, performed_by, notes, created_at
                      [IMMUTABLE — never UPDATE or DELETE]
cost_layers         — id, product_id, lot_id, qty, remaining_qty, unit_cost, received_at
users               — id, email, password_hash, name, role ENUM(admin|warehouse_manager|picker|viewer),
                      warehouse_id (nullable), is_active
refresh_tokens      — id, user_id, token_hash, expires_at
```

---

## 3. API Surface (summary)

| Domain | Method | Path | Roles |
|--------|--------|------|-------|
| **Auth** | POST | `/api/auth/register` | public |
| | POST | `/api/auth/login` | public |
| | POST | `/api/auth/refresh` | authenticated |
| **Users** | GET | `/api/users` | admin |
| | POST | `/api/users` | admin |
| | PUT | `/api/users/:id` | admin |
| | DELETE | `/api/users/:id` | admin |
| **Warehouses** | GET/POST | `/api/warehouses` | admin, manager |
| | GET/PUT/DELETE | `/api/warehouses/:id` | admin, manager |
| | GET/POST | `/api/warehouses/:id/locations` | admin, manager |
| **Products** | GET/POST | `/api/products` | admin, manager |
| | GET/PUT/DELETE | `/api/products/:id` | admin, manager |
| | GET | `/api/products/by-barcode/:code` | authenticated |
| **Inventory** | GET | `/api/inventory?warehouse_id=&product_id=&location_id=` | authenticated |
| | GET | `/api/inventory/:stock_id` | authenticated |
| | GET | `/api/inventory/alerts?threshold=` | admin, manager |
| | GET | `/api/inventory/summary/:warehouse_id` | authenticated |
| **Movements** | POST | `/api/movements/receive` | admin, manager, picker |
| | POST | `/api/movements/pick` | admin, manager, picker |
| | POST | `/api/movements/transfer` | admin, manager |
| | POST | `/api/movements/adjust` | admin, manager |
| | GET | `/api/movements?type=&product_id=&from=&to=` | authenticated |
| | GET | `/api/movements/:id` | authenticated |
| **Reports** | GET | `/api/reports/valuation` | admin, manager |
| | GET | `/api/reports/movement-summary` | admin, manager |
| | GET | `/api/reports/velocity` | admin, manager |
| | GET | `/api/reports/export/:type?format=csv` | admin, manager |

---

## 4. Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| ORM vs sqlc | **sqlc + pgx** | Compile-time type safety, full SQL power (CTEs, window functions), no ORM magic |
| JWT vs sessions | **JWT (RS256)** | Stateless, works with mobile scanners, no Redis dependency |
| RBAC framework | **Hard-coded in Go** | 4 roles, ~25 endpoints — Casbin is overkill |
| Frontend state | **Pinia** | TypeScript-native, Composition API style, modular stores |
| UI library | **PrimeVue** | Best DataTable for 10k+ inventory rows (virtual scroll, export, filtering) |
| SPA serve | **Separate deployment** | Go API + Vue SPA via nginx/CDN. Scales independently, standard CORS |
| Barcode decoding | **Frontend only** | `@teckel/vue-barcode-reader` for camera, USB scanners are keyboard wedge (no JS needed) |
| Audit trail | **Append-only stock_movements** | Immutable source of truth, never updated/deleted, enables reconciliation |
| Multi-warehouse | **warehouse_id on rows** | Simpler than schema-per-warehouse, flexible for cross-warehouse queries |
| FIFO costing | **cost_layers with remaining_qty** | Outbound picks consume oldest layers first, accurate inventory valuation |

---

## 5. Data Flow Diagrams

### Barcode Scan → Receive Flow
```
[USB/Bluetooth Hardware Scanner] → focused input + Enter → validate barcode
  → GET /api/products/by-barcode/{code}
  [Optional Phone Camera] ────────────────────────────────────────┘
  → Product found? No → "Unknown product" error
  → Yes → show product info → operator selects location + quantity
  → POST /api/movements/receive {product_id, warehouse_id, location_id, qty, lot_id?}
  → BEGIN TX
    → INSERT stock_movements (type=receive)
    → UPSERT stock (increment qty)
    → INSERT cost_layers (for FIFO)
  → COMMIT TX
  → Return success + updated stock snapshot
```

### Stock Transfer Flow
```
[Operator selects source warehouse/location] → shows items at source
  → Select item + quantity → select destination warehouse/location
  → POST /api/movements/transfer {product_id, lot_id?, qty, from_w, from_l, to_w, to_l}
  → BEGIN TX
    → Validate source has sufficient stock (qty - reserved_qty >= transfer_qty)
    → INSERT stock_movements (type=transfer, from_w, from_l, to_w, to_l)
    → UPDATE stock at source (decrement qty WHERE location_id = from_l)
    → UPSERT stock at destination (increment qty)
  → COMMIT TX
  → Return success
```

### Auth + RBAC Flow
```
[Login] → POST /api/auth/login {email, password}
  → Verify bcrypt hash → Generate RS256 JWT + refresh token
  → Return {access_token, refresh_token, user}

[Request to protected endpoint] →
  1. Fiber JWT middleware → validate token, extract claims → c.Locals("user", claims)
  2. RBAC middleware → check claims.Role against required roles for this route
  → Allowed? Yes → handler runs
  → Allowed? No → 403 Forbidden

[Token expires] → POST /api/auth/refresh {refresh_token}
  → Validate refresh token → Issue new access token
```

---

## 6. Open Questions (to resolve before Phase 2)

1. **Hardware:** USB/Bluetooth barcode scanners use keyboard-wedge mode by
   default; phone-camera scanning is optional.
2. **Offline mode:** Does the warehouse need offline PWA support? If yes, add IndexedDB sync.
3. **Mobile-first or desktop-first:** Floor staff on phones/tablets vs workstations?
4. **SaaS or on-prem:** Multi-tenant SaaS vs single-company on-prem — affects auth, deployment, data isolation.
5. **Barcode formats:** Just EAN-13 / Code 128, or also QR, GS1 DataMatrix?
6. **ERP integrations:** Need to sync with NetSuite, SAP, Shopify, etc.?
7. **Real-time updates:** WebSocket/SSE for live stock changes across users?
8. **Audit retention:** Keep movement records indefinitely? Affects table partitioning strategy.

---

## 7. Success Criteria

- [ ] Warehouse staff can scan a barcode and complete a receive/pick in under 10 seconds
- [ ] Stock quantities are always correct (reconciled via stock_movements audit trail)
- [ ] FIFO cost layers are consumed correctly (oldest first)
- [ ] All 4 roles have correct access boundaries (viewer can't modify, picker can't delete users)
- [ ] PrimeVue DataTable handles 50k+ product rows with virtual scroll
- [ ] Mobile responsive — floor staff can use on tablet
- [ ] API responses under 200ms for 95th percentile
