# Tailwind Migration & Warehouses Figma Reconciliation Design

**Status:** Approved for implementation on 2026-08-19
**Scope:** (1) Migrate all hand-written CSS in the frontend to Tailwind CSS. (2) Reconcile the Warehouses & Locations screens with the actual Figma design ("Warehouses & Locations", node `1:310`), using real Figma-derived design tokens.

## Why

The Warehouses & Locations screens built earlier this session were functionally correct but didn't match the Figma design at all — pulling the real Figma screen showed missing breadcrumbs, wrong status-badge styling, an extra Barcode column, text buttons instead of icon actions, and no shared design-token system. Separately, the user decided the frontend should move to Tailwind CSS as its styling system, for all screens, not just Warehouses.

Doing both together is more efficient than two passes: the Warehouses reconciliation needs a token system (colors, radius, fonts) anyway, and that token system belongs in Tailwind's `@theme`, not as ad-hoc CSS custom properties that would need to be re-migrated to Tailwind later.

## What "full migration" means here

- **All hand-written CSS is migrated to Tailwind utility classes.** This means `frontend/src/assets/main.css` (359 lines, shared across `AppLayout.vue`, `LoginView.vue`, `DashboardView.vue`, `BarcodeTestView.vue`, `NotFoundView.vue`) and any scoped `<style>` blocks in our own `.vue` files.
- **PrimeVue components are not touched.** `Button`, `Card`, `Dialog`, `DataTable`, `InputText`, `Message`, `Select`, `Tag`, `ToggleSwitch` etc. keep their own Aura-theme styling — that's a separate library's internal CSS, not something a Tailwind migration replaces. Tailwind styles the page structure around and between PrimeVue components (layout, spacing, headings, custom elements like the sidebar/topbar/stat cards), exactly as the hand-written CSS did before.
- **Existing screens (Login, Dashboard, Barcode test, AppLayout shell, NotFound) get a 1:1 visual lift-and-shift** — same colors, spacing, layout, responsive breakpoints, just expressed as Tailwind utilities instead of custom CSS classes. No redesign for these screens in this pass, even though Login and Dashboard also have Figma frames (`Login Terminal`, `Operational Dashboard`) — reconciling those to Figma is separate, future work.
- **Warehouses & Locations screens get the actual Figma reconciliation** — built fresh with Tailwind using the real design tokens below, matching the Figma screen's structure where our real data supports it.

## Tailwind setup

Tailwind CSS v4 via its native Vite plugin (`@tailwindcss/vite`) — no `tailwind.config.js`, no PostCSS config; theme tokens are declared with `@theme` directly in CSS, which fits this project's existing single `assets/main.css` entry point.

```bash
npm install -D tailwindcss @tailwindcss/vite
```

`vite.config.ts` gets the plugin added alongside the existing `vue()` plugin (test config from the earlier Warehouses slice is untouched).

`assets/main.css` starts with `@import "tailwindcss";` followed by an `@theme` block defining the design tokens below, replacing the old `:root` font/color declarations.

## Design tokens (pulled from Figma via `get_design_context` on node `1:310`)

```css
@theme {
  --color-primary: #316bf3;
  --color-primary-hover: #0051d5;

  --color-ink: #1b1b1d;      /* primary text */
  --color-ink-muted: #45474c; /* secondary text */
  --color-ink-faint: #8590a6; /* muted/placeholder text */

  --color-success-bg: #dcfce7;
  --color-success-text: #16a34a;
  --color-danger-bg: #fee2e2;
  --color-danger-text: #dc2626;
  --color-warning-bg: #fef3c7;
  --color-warning-text: #d97706;

  --color-border: #e2e8f0;
  --color-border-strong: #c5c6cd;
  --color-surface: #f8fafc;

  /* Existing brand colors, preserved as tokens for the lift-and-shift screens */
  --color-brand-navy: #0f2947;
  --color-brand-amber: #ffbd59;

  --radius-card: 12px;
  --radius-badge: 4px;

  --font-sans: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  --font-mono-code: "JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, monospace;
}
```

Inter and JetBrains Mono aren't currently loaded as webfonts (Inter was declared but relies on system availability). `index.html` gets a Google Fonts `<link>` for both, so the monospace code styling in the Warehouses reconciliation renders consistently rather than silently falling back to a generic monospace.

## File-by-file plan

**Lift-and-shift (visual-preserving, Tailwind-only):**
- `assets/main.css` — replaced with the `@theme` block above; all component-specific rules removed (they move into templates as utility classes)
- `layouts/AppLayout.vue` — sidebar/topbar/nav classes → Tailwind utilities, using `--color-brand-navy`/`--color-brand-amber` tokens (current colors, unchanged)
- `views/LoginView.vue` — `.auth-page`/`.login-card`/`.login-form` classes → Tailwind utilities
- `views/DashboardView.vue` — `.page-heading`/`.status-pill`/`.metric-grid`/`.roadmap` classes → Tailwind utilities
- `views/BarcodeTestView.vue` — `.scanner-grid`/`.scanner-panel`/`.scanner-video`/`.button-row`/`.barcode-result` classes → Tailwind utilities
- `views/NotFoundView.vue` — `.not-found` classes → Tailwind utilities
- `index.html` — add Google Fonts link for Inter + JetBrains Mono

**Figma reconciliation (rebuilt with new tokens):**
- `views/WarehousesListView.vue` — page header restyled with token colors/radius; warehouse codes in `font-mono-code`; status badges using `--color-success-*`/`--color-danger-*`; icon-only row actions (edit pencil, deactivate icon) replacing text buttons; search field restyled
- `views/WarehouseDetailView.vue` — breadcrumb (`Warehouses > {warehouse name}`) added; same code/badge/icon-action treatment as the list view; "Locations" section header gets a real-data "N shown" count badge (not a fabricated total — the API has no total-count field, so this reflects loaded items, unlike Figma's literal "482 Results")
- `components/WarehouseFormDialog.vue`, `components/LocationFormDialog.vue` — primary button color aligned to `--color-primary`; no structural changes (Figma didn't show these as separate screens)

**Deliberate deviations from Figma, kept:**
- Barcode column stays in the locations table — real, useful data our API has that the Figma mockup simply didn't design a column for. Removing it would be a functionality regression, not a style fix.
- No stat cards (Total Capacity / Active Pick Paths / Recent Activity) and no "Blocked (Audit)" status — these require backend data (warehouse capacity, pick-path zones, task activity feed, audit-block flag on locations) that doesn't exist yet. Adding fake numbers would be dishonest data, not a style choice.
- Global topbar (warehouse+shift switcher, notification bell, avatar) stays as the current username/role + Sign out — that's a shared shell component affecting every screen, deliberately deferred to its own design pass rather than bundled here.

## Testing

The existing Vitest suites (42 tests) assert on `data-testid` attributes and rendered text content, not CSS class names — confirmed by reviewing the test files. The lift-and-shift and reconciliation should not break any existing test. `WarehousesListView.spec.ts` / `WarehouseDetailView.spec.ts` / the two dialog specs stay as-is; no new test behavior is being added in this pass (this is a styling change, not a functional one), so no new test cases are required beyond re-running the existing suite to confirm nothing broke.

Manual verification: run the app locally, visually compare each migrated screen against its pre-migration screenshot (for the lift-and-shift screens) and against the Figma screenshot (for Warehouses/Locations).

## Out of scope

Redesigning Login/Dashboard to match their Figma frames (separate future work). Global topbar/sidebar redesign. Backend work for stat-card data or audit-block status. Products/Inventory/Movements/Users screens (not built yet).
