# Catalog (Products & Categories) Frontend Design

**Status:** Approved for implementation on 2026-08-19
**Scope:** Second frontend vertical slice, following the Warehouses & Locations pattern. Builds `GET/POST/PUT/DELETE /api/v1/products` and `/api/v1/categories` (`docs/api-contract.md`) against the real Figma "Products & Categories" screen (node `1:556`).

## What's real vs. flagged (established via Figma comparison)

**Built for real** — table (SKU, Barcode, Product Name, Category, Unit, Lot-Tracked, Status, Actions), search across SKU/name/barcode, Category filter, Status filter (`is_active`), Add Product, per-row Edit/Deactivate icon actions.

**Figma shows, not built** (matches the Warehouses "no fake data" rule):
- 3-state status (Active/Inactive/**Draft**) — API only has boolean `is_active`. Built as 2-state.
- Bulk-select checkboxes — no bulk endpoints exist. Column omitted entirely.
- "Export" button — no export endpoint; a client-side export would only cover the loaded page, misleading against the "1,204 entries" framing. Omitted.
- Numbered pagination / total count — cursor API has no total. "Load more" + honest "N shown" count, same as Warehouses.

**Not in Figma, built anyway:** Category management. Figma only has a Category *filter* dropdown, no CRUD screen — same gap as Warehouses having no list-screen frame. Backend fully supports category CRUD (with `parent_id` hierarchy), so a "Manage Categories" dialog is added, in the same visual style as the Warehouse/Location form dialogs.

## Data model (matches `docs/api-contract.md` / `backend/internal/modules/catalog/model.go` exactly)

```ts
interface Category {
  id: string
  parent_id: string | null
  name: string
  is_active: boolean
  created_at: string
  updated_at: string
}

interface Product {
  id: string
  category_id: string | null
  sku: string
  barcode: string | null
  name: string
  unit: string
  is_lot_tracked: boolean
  is_active: boolean
  created_at: string
  updated_at: string
}
```

**Category name resolution:** `Product` only stores `category_id`, no denormalized name. The products store also loads the full categories list (used for both the filter dropdown and a `categoryName(id)` lookup for the table's Category column) — categories are a small, bounded list, unlike products.

## RBAC (matches the real API rules)

| Role | Read | Create/edit/deactivate |
| --- | --- | --- |
| `admin` | All | Yes |
| `warehouse_manager` | All | Yes |
| `picker` | All | No |
| `viewer` | All | No |

Catalog data is global, not warehouse-scoped (unlike Warehouses/Locations) — every role sees the same product list; only mutation is role-gated. `canManageCatalog = role === 'admin' || role === 'warehouse_manager'`.

## Files

```
frontend/src/
  api/catalog.ts                      # typed fetch calls, mirrors api/warehouses.ts style
  stores/catalog.ts                   # Pinia store: products + categories state, CRUD actions
  components/ProductFormDialog.vue     # create/edit product modal
  components/CategoryFormDialog.vue    # create/edit category modal (list + inline form)
  views/ProductsView.vue               # route: /products
```

Modified:
- `layouts/AppLayout.vue` — add a real "Products" `RouterLink` (replaces nothing disabled; it's a new top-level item alongside Warehouses)
- `router/index.ts` — add the `products` route as an `AppLayout` child

## Screen: `ProductsView.vue`

Header: title + subtitle, "Manage Categories" button (opens `CategoryFormDialog`, `v-if="canManageCatalog"`), "Add Product" button (`v-if="canManageCatalog"`). Search input (debounced 300ms, matches the Warehouses pattern). Category filter `Select` (options from the loaded categories list, includes "All"). Status filter `Select` ("All"/"Active"/"Inactive").

Table columns: SKU (`font-mono-code`), Barcode (`font-mono-code`, em-dash if null), Product Name, Category (resolved name, "—" if `category_id` is null), Unit, Lot-Tracked (icon: check if true, dash if false — reusing the Warehouses pattern for the Pickable column), Status (badge, `success-*`/`danger-*` tokens), Actions (icon-only Edit/Deactivate, `v-if="canManageCatalog"`).

States: loading, empty, error (same conventions as `WarehousesListView`), populated. "Load more" button instead of numbered pagination.

## `ProductFormDialog.vue`

Same `v-model:visible` + `product: Product | null` prop pattern as `WarehouseFormDialog`. Fields: SKU (required), Barcode (optional, reuses `validateBarcode` from `lib/barcode.ts` for the same live-hint treatment as `LocationFormDialog`), Name (required), Unit (required), Category (`Select`, options from the categories list, optional), Lot-Tracked (`ToggleSwitch`), Active (`ToggleSwitch`). Submit calls the store; catches `PRODUCT_SKU_CONFLICT` → inline on SKU field, `PRODUCT_BARCODE_CONFLICT` → inline on Barcode field, `INVALID_BARCODE` → inline on Barcode field, other `ApiClientError` → generic banner.

## `CategoryFormDialog.vue`

Simpler than the product/warehouse dialogs since Figma doesn't design this screen — a single dialog combining a flat list of existing categories (name + status badge + inline edit/deactivate icon actions) with an inline create/edit form (Name required, Parent Category `Select` with "None" option populated from other active categories, Active toggle). Catches `CATEGORY_NAME_CONFLICT` → inline on Name, `CATEGORY_CYCLE` → inline on Parent, `CATEGORY_IN_USE` (on deactivate attempt) → inline banner explaining an active child/product blocks deactivation.

## Testing

Same conventions as the Warehouses slice: `api/catalog.test.ts` (mocked `apiRequest`, one test per function/filter-combination), `stores/catalog.test.ts` (mocked `@/api/catalog`, state/loading/error per action), `ProductsView.spec.ts` (renders rows, RBAC button visibility, error state), `ProductFormDialog.spec.ts` (validation, conflict-error routing, barcode hint). `CategoryFormDialog.spec.ts` covers create/edit/conflict-error paths given it has no Figma reference to visually validate against — correctness here is proven by tests, not a screenshot comparison.

## Out of scope

Bulk actions, export, draft status, a dedicated category hierarchy/tree visualization (the parent dropdown is flat, not nested — sufficient for the current shallow hierarchies in use). Barcode label printing (already out of scope project-wide).
