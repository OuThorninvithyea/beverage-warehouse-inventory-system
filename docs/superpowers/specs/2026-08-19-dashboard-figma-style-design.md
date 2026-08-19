# Dashboard Figma Style Reconciliation Design

**Status:** Approved for implementation on 2026-08-19
**Scope:** Restyle `DashboardView.vue`'s existing content (3 foundation stat cards + implementation-path roadmap) to match the Figma "Operational Dashboard" screen's visual language (node `1:43`, https://www.figma.com/design/6Jp6M6Tns2EOgsv5B1zBrc/BMS?node-id=1-43). Style only, no new data — same rule as the Warehouses reconciliation.

## Why not a full rebuild

Figma's Dashboard shows Total Inventory Value / Low Stock Items / Expiring Soon stat cards and a "Recent Inventory Movements" table. Checked against `docs/api-contract.md`: the movements table *could* be wired to real data (`GET /api/v1/inventory/movements` exists), but the three stat cards need aggregates the backend doesn't expose yet (no min-stock-threshold concept, no inventory-value aggregate, no cross-product "expiring soon" summary). Per the user's explicit instruction, this pass is style-only — restyling our *existing* dashboard content, not fabricating new cards/tables with data we don't have. Wiring the real movements table is a separate future slice.

## Style changes

- **Cards** get a flat, bordered look matching Figma instead of PrimeVue's default shadow: `border border-border rounded-[12px] shadow-none`.
- **Stat cards** (`foundationItems`) restructured to Figma's label-then-value hierarchy: small uppercase `text-ink-faint` label on top, large `font-bold text-ink` value below — reordered from the current value-then-label layout. No new content added (no icon, no helper-text line — we don't have that data).
- **"Foundation ready" pill** switches from the one-off `#dff8eb`/`#176c43` colors to the existing `success-bg`/`success-text` tokens (already defined, used by the Warehouses status badges) — removes a duplicate color pair.
- **Roadmap mini-cards** get the same bordered/`rounded-[12px]` treatment as the stat cards instead of a flat `bg-brand-surface` fill; the "Weeks X" label switches from the one-off `#b86600` to the existing `warning-text` token.

## New tokens

Two more status colors from Figma's palette, added to `main.css`'s `@theme` block for future use (Figma's Movements screen uses these for "Transfer"/"Adjust" type badges — not used on the Dashboard itself, but part of the same token system):

```css
--color-info-bg: #dbeafe;
--color-info-text: #2563eb;
--color-accent-bg: #f3e8ff;
--color-accent-text: #9333ea;
```

## Testing

No component test exists for `DashboardView.vue` and this is a style-only change with no new behavior — no test changes needed. Verify with `npm run build` (confirms Tailwind compiles the new utilities) and a manual visual check against the Figma screenshot.

## Out of scope

Real "Recent Inventory Movements" table, real stat-card data, backend work for min-stock thresholds / inventory value / expiry aggregates, global topbar redesign (still deferred from the Warehouses reconciliation).
