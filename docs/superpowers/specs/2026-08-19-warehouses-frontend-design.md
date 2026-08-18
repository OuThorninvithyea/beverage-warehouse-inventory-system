# Warehouses & Locations Frontend Design

**Status:** Approved for implementation on 2026-08-19
**Scope:** First frontend vertical slice — the reference pattern for Catalog, Users, and Inventory screens that follow.

## Goal

Build the Warehouses list/detail screens against the already-complete
`GET/POST/PUT/DELETE /api/v1/warehouses` and
`GET/POST/PUT/DELETE /api/v1/warehouses/:warehouse_id/locations` API
(`docs/api-contract.md`). This is the first screen with real interactive
functionality in the frontend — everything before this (login, dashboard
shell, barcode test) either has no data or is a placeholder.

## Why Warehouses first

Of the four candidate first slices (Warehouses, Dashboard, Catalog,
Movements), Warehouses is the only one with a completely gap-free backend.
Dashboard needs reporting APIs that don't exist yet and has an unresolved
RBAC conflict (see `docs/figma-ai-prompts.md`'s dashboard review). Catalog
and Movements are reasonable second/third slices once this one establishes
the pattern.

## Existing code this builds on

Already implemented and not being changed:
- `api/client.ts` — `apiRequest<T>()`, native `fetch`, not Axios (a real
  deviation from the proposal's tech stack list — documented, not fixed;
  this design follows the code that actually exists).
- `stores/auth.ts` — Pinia composition-API store pattern to mirror.
- `router/index.ts` — auth guard already redirects unauthenticated users
  to `/login`; new routes just need `meta` left as default (protected).
- `layouts/AppLayout.vue` — sidebar nav; currently has `<span
  class="nav-link--disabled">` placeholders for "Inventory" and
  "Movements" but no "Warehouses" entry at all yet.

## Files

```
frontend/src/
  types/pagination.ts                # Page<T> = { items: T[]; page: { next_cursor: string | null; has_more: boolean } }
  api/warehouses.ts                  # typed fetch calls, mirrors api/auth.ts style
  stores/warehouses.ts               # Pinia store: list state, loading/error, CRUD actions
  components/WarehouseFormDialog.vue  # create/edit warehouse modal (PrimeVue Dialog)
  components/LocationFormDialog.vue   # create/edit location modal
  views/WarehousesListView.vue        # route: /warehouses
  views/WarehouseDetailView.vue       # route: /warehouses/:warehouseId (shows its locations)
```

Modified:
- `layouts/AppLayout.vue` — add a real "Warehouses" `RouterLink`; leave
  "Inventory"/"Movements" disabled placeholders as-is (still true).
- `router/index.ts` — add the two new routes as `AppLayout` children.
- `package.json` — add `@vue/test-utils` and `jsdom` as dev dependencies.
- `vite.config.ts` (or a new `vitest.config.ts`) — set `test.environment: 'jsdom'`.

## Data model (matches `docs/api-contract.md` exactly)

```ts
interface Warehouse {
  id: string
  code: string
  name: string
  address: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

interface Location {
  id: string
  warehouse_id: string
  code: string
  zone: string | null
  aisle: string | null
  rack: string | null
  shelf: string | null
  barcode: string | null
  is_pickable: boolean
  is_active: boolean
  created_at: string
  updated_at: string
}
```

`api/warehouses.ts` exposes: `listWarehouses(filter)`, `getWarehouse(id)`,
`createWarehouse(input)`, `updateWarehouse(id, input)`,
`deactivateWarehouse(id)`, and the location equivalents scoped under a
warehouse id. List filters: `limit`, `after` (cursor), `search`,
`is_active` (both), `is_pickable` (locations only) — passed as query
params, matching the contract precisely.

## RBAC (matches the real API rules)

| Role | Read | Warehouse create/edit/deactivate | Location create/edit/deactivate |
| --- | --- | --- | --- |
| `admin` | All warehouses | Yes | Yes |
| `warehouse_manager` | Assigned warehouse only (server-scoped) | No | Yes, within assigned warehouse |
| `picker` / `viewer` | Assigned warehouse only (server-scoped) | No | No |

The frontend does **not** re-implement the warehouse-scoping — the API
already returns only what the caller's role permits. The frontend only
needs to hide/show action buttons: `canManageWarehouses = role === 'admin'`,
`canManageLocations = role === 'admin' || role === 'warehouse_manager'`.

## Screens

### `WarehousesListView.vue`

Search input (debounced, hits `search` filter), `is_active` toggle filter,
"Add Warehouse" button (`v-if="canManageWarehouses"`). PrimeVue `DataTable`
with columns Code / Name / Address / Status / row actions (Edit,
Deactivate — both `v-if="canManageWarehouses"`). Row click navigates to
`WarehouseDetailView`. States: loading skeleton, empty ("No warehouses
yet" + the add button if permitted), error (retry button), populated.
"Load more" button at the bottom instead of a numbered paginator, since
the API cursor has no page-number concept.

### `WarehouseDetailView.vue`

Header: warehouse code/name/address, Edit button (`canManageWarehouses`).
Below: a Locations `DataTable` (Code / Zone / Aisle / Rack / Shelf /
Barcode / Pickable / Status / actions), "Add Location" button
(`canManageLocations`). Same four states as the list view. Location
barcode field shows a validity hint reusing the existing
`lib/barcode.ts` checksum validator if a value is entered (consistent
with the barcode-test screen's validation, not a new implementation).

### `WarehouseFormDialog.vue` / `LocationFormDialog.vue`

PrimeVue `Dialog`, controlled by a `v-model:visible` + an `input`/`null`
prop (null = create mode, populated = edit mode). Client-side validation:
required `code`/`name` for warehouse, required `code` for location.
Submits call the store action; on the API's `*_CODE_CONFLICT` /
`*_BARCODE_CONFLICT` errors, show the conflict inline on the relevant
field rather than a generic toast.

## Testing

Add `@vue/test-utils` + jsdom (`test.environment: 'jsdom'`). Tests:
- `api/warehouses.test.ts` — each function calls `apiRequest` with the
  right path/method/body (mocking `apiRequest`, matching how a real unit
  test would isolate the network layer).
- `stores/warehouses.test.ts` — store actions update state correctly,
  handle loading/error, given a mocked API layer.
- `WarehousesListView.spec.ts` — mounts with `@vue/test-utils`: renders
  rows from store state; "Add Warehouse" hidden for a non-admin session;
  shown for admin.
- `WarehouseFormDialog.spec.ts` — required-field validation blocks submit;
  conflict error surfaces on the right field.

No E2E (Playwright) yet — proposal scopes that to Phase 6, not reached.

## Out of scope for this slice

Catalog, Users, Inventory/Movements screens (separate slices). Barcode
label printing. Any reporting/dashboard data. Locking `WarehousesListView`
behind a route guard beyond the existing global auth guard (no
per-route role restriction needed — the API already scopes reads; write
actions are just hidden buttons, and a non-admin hitting the write API
directly would get a real `403 FORBIDDEN` from the server regardless of
what the UI shows).
