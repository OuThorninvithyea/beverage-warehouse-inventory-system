# Figma AI (First Draft) prompts — BWIMS screens

Generated 2026-08-18. Each prompt is self-contained and meant to be pasted
into Figma AI / First Draft one at a time — it generates one screen per
prompt, so a single giant "design the whole app" prompt produces weaker
results than focused ones.

Grounded in `docs/user-journeys-and-wireframes.md` (approved navigation
model, journeys, ASCII wireframes, accessibility requirements) and the real
API contracts in `docs/api-contract.md` — field names, roles, and states
below match what's actually implemented, not invented.

**Shared style direction for every screen** (repeated in each prompt so
each one stands alone): clean, utilitarian warehouse-operations style. Dense
data tables over decorative whitespace. High-contrast status colors (red for
low/out of stock, amber for expiring/expired, green for healthy). Large
touch targets (minimum 44×44px) since floor staff use phones/tablets with
gloves. WCAG AA contrast. Status is never conveyed by color alone — pair
every color cue with an icon or text label. Left sidebar navigation on
desktop, bottom tab bar on mobile.

---

## 1. Login

```
Design a login screen for BWIMS (Beverage Warehouse Inventory Management
System), a B2B warehouse operations web app. Clean, utilitarian style —
this is operational software for warehouse staff, not a consumer app.

Layout: centered card on a plain background, app name "BWIMS" top-left or
above the card.

Card contents:
- "Welcome back" heading
- Email field (labeled "Email")
- Password field (labeled "Password", with show/hide toggle)
- Primary "Sign in" button, full width of the card
- Inline error state below the button: generic message like "Invalid email
  or password" (never reveals whether the account exists)
- Loading state: button shows a spinner and disables while submitting

No "forgot password" or "sign up" links — this app has no self-service
registration; accounts are admin-provisioned only.

Style: high contrast, WCAG AA compliant, large tap targets (min 44x44px),
keyboard-focus-visible outlines on all fields and the button.
```

---

## 2. Application shell + Dashboard

```
Design the main application shell and dashboard for BWIMS, a warehouse
inventory web app. Utilitarian, data-dense operational style, not a
marketing/SaaS aesthetic.

Layout: left sidebar navigation (desktop) with these items, each with a
simple line icon: Dashboard, Products, Inventory, Movements, Reports. Below
the nav, at the bottom: current warehouse selector dropdown, then user
avatar/name with role badge (Admin / Warehouse Manager / Picker / Viewer)
and a logout option.

Top of main content area: page title "Dashboard" and the warehouse
selector repeated/visible.

Main content: a row of 3 stat cards — "Inventory Value", "Low Stock Items"
(count, red accent if > 0), "Expiring Soon" (count, amber accent if > 0).
Below that: a "Recent Movements" table/list showing type (receive/pick/
transfer/adjust as colored badges), product name, quantity, location,
timestamp, and who performed it.

Include 4 visual states as separate frames or annotated variants: loading
(skeleton placeholders for the stat cards and table rows), empty (friendly
empty-state illustration + text when no data exists yet), error (retry
button with an error message), and populated (realistic sample data: e.g.
"Coca-Cola 330ml Can", "Sprite 1.5L Bottle", quantities like 240, 18, 5).

Role note: a Viewer sees this same dashboard read-only, with no action
buttons anywhere. Other roles see identical content here (no admin-only
elements on the dashboard itself).

Style: clean utilitarian, dense tables, high-contrast status colors (never
color alone — pair with icon/label), large touch targets, WCAG AA.
```

---

## 3. Products & Categories (list + form)

```
Design a Products list and edit-form screen for BWIMS, a warehouse
inventory web app. Utilitarian, data-dense operational style.

Layout: left sidebar nav (same as dashboard, "Products" highlighted active).
Top bar: page title "Products", a search input (placeholder "Search by SKU,
name, or barcode"), a Category filter dropdown, an Active/Inactive filter
toggle, a "Scan barcode" button (camera icon) and a primary "Add Product"
button (admin/warehouse_manager only — hide this button in a Picker/Viewer
variant).

Table columns: SKU, Barcode, Name, Category, Unit, Lot-Tracked (yes/no
badge), Status (Active/Inactive badge), and a row action menu (Edit,
Deactivate).

Below or overlapping the table: a slide-in form drawer for
create/edit, with fields: SKU (text), Name (text), Category (searchable
dropdown, optional — shows "No category" if cleared), Unit (text, e.g.
"case", "each"), Barcode (text, optional, with a small "valid EAN-13/UPC-A"
checkmark indicator once a valid value is entered), Is Lot Tracked
(toggle), Is Active (toggle). Cancel and Save buttons. Inline validation
errors under each invalid field (e.g. "SKU already exists").

Also design a compact "Categories" side-panel or secondary tab within the
same screen: a simple hierarchical list of categories (name, parent shown
indented) with add/edit/deactivate actions, since categories are optional
parents for products.

States: empty table (no products yet, with an "Add your first product"
prompt), loading skeleton rows, and populated with realistic beverage
products (e.g. "Coca-Cola 330ml Can" / SKU COKE-330-CAN, "Sprite 1.5L
Bottle" / SKU SPRITE-1500).

Style: clean utilitarian, dense table, high-contrast status colors, large
touch targets, WCAG AA.
```

---

## 4. Barcode scan (mobile)

```
Design a mobile barcode-scanning screen for BWIMS, a warehouse inventory
app used on phones by floor staff. Full-screen camera-first layout.

Layout: top bar with a back arrow and title "Scan barcode". Below it, a
large camera preview filling most of the screen, with a rectangular scan
frame/reticle overlaid in the center and subtle corner brackets.

Below the camera preview: a result card that appears once a barcode is
decoded — shows the decoded value (e.g. "8851234567890"), a checkmark icon
with "Valid EAN-13", and two buttons: "Use value" (primary) and "Rescan"
(secondary).

Below that: a manual entry fallback — a text input with placeholder "Or
enter barcode manually" and a submit button, for when the camera can't be
used (USB scanner input also lands in this same field, since USB scanners
type digits + Enter like a keyboard).

Error state variant: camera permission denied — show a friendly message
explaining camera access is needed, with a "Try manual entry instead"
button.

Note: this screen only produces a lookup value — it never changes
inventory by itself. Confirming a receive/pick action happens on a
separate confirmation screen after this one.

Style: clean utilitarian, large touch targets (min 44x44px, this is used
one-handed on a warehouse floor), high contrast for outdoor/bright
warehouse lighting, WCAG AA.
```

---

## 5. Warehouses & Locations

```
Design a Warehouses and Locations management screen for BWIMS, a warehouse
inventory web app. Admin and Warehouse Manager only — Pickers and Viewers
never see this screen.

Layout: left sidebar nav. Top bar: page title "Warehouses", search input,
"Add Warehouse" button (admin only).

Warehouse list: cards or table rows showing Code, Name, Address, Status
(Active/Inactive badge), and a "View Locations" action.

Drill-down / detail view for a selected warehouse: header showing the
warehouse name/code, then a Locations table with columns Code, Zone,
Aisle, Rack, Shelf, Barcode, Pickable (yes/no badge), Status
(Active/Inactive), row actions (Edit, Deactivate). "Add Location" button
(admin or the warehouse's assigned manager).

Form drawer for warehouse create/edit: Code, Name, Address, Is Active
toggle.

Form drawer for location create/edit: Code, Zone, Aisle, Rack, Shelf,
Barcode (optional, globally unique), Is Pickable toggle, Is Active toggle.

States: empty (no warehouses yet), loading skeleton, populated with
realistic data (e.g. warehouse code "PP-01" / "Phnom Penh Main Warehouse",
locations like "A-01-R02-S03").

Style: clean utilitarian, dense tables, high-contrast status colors, large
touch targets, WCAG AA.
```

---

## 6. Users (admin)

```
Design a User Management screen for BWIMS, a warehouse inventory web app.
Admin-only — every part of this screen, including viewing it, requires the
admin role.

Layout: left sidebar nav. Top bar: page title "Users", search input
(placeholder "Search by name or email"), Role filter dropdown (Admin /
Warehouse Manager / Picker / Viewer), Warehouse filter dropdown, Active
filter toggle, "Add User" button.

Table columns: Full Name, Email, Role (colored badge per role), Assigned
Warehouse (or "All warehouses" for admins), Status (Active/Inactive),
row action menu (Edit, Reset Password, Deactivate).

Form drawer for create: Email, Full Name, Role (dropdown), Warehouse
(searchable dropdown, disabled/hidden if Role is Admin since admins aren't
warehouse-scoped), Password (with a helper text "minimum 12 characters").

Form drawer for edit: same fields minus password, plus Is Active toggle.
Include a "Reset Password" secondary action that opens a small modal with
just a new password field.

Important safety states to depict: an inline warning banner when trying to
deactivate or demote the account you're currently logged in as ("You
cannot deactivate or change your own admin role — ask another
administrator"), and an error toast for "This would leave zero active
administrators" if attempting to deactivate/demote the last remaining
admin.

States: empty, loading skeleton, populated with sample users across all 4
roles.

Style: clean utilitarian, dense table, high-contrast status colors, large
touch targets, WCAG AA.
```

---

## 7. Inventory balances

```
Design an Inventory balances screen for BWIMS, a warehouse inventory web
app — this is the "current stock levels" view, read-only for all roles.

Layout: left sidebar nav, "Inventory" highlighted active. Top bar: page
title "Inventory", filters for Location (dropdown), Product (searchable
dropdown), Lot (dropdown, only enabled once a product is picked), and the
current warehouse context shown/locked for non-admin roles.

Table columns: Product Name & SKU, Location Code, Lot Number (or "—" for
non-lot-tracked products), Expiration Date (highlighted amber if within 30
days, red if already past), Quantity, Reserved Quantity, Available
Quantity (quantity minus reserved, shown bold), a low-stock indicator icon
if available quantity is below a threshold.

Include a small lot-detail expandable row or side panel showing all lots
for a product across locations, ordered by nearest expiration first (FEFO
order) — this is what a picker consults before manually overriding the
system's automatic lot choice.

States: empty (no stock at all), loading skeleton, populated with
realistic data including at least one expiring-soon row (amber) and one
already-expired row (red) to show both visual treatments.

Style: clean utilitarian, very dense table (this screen may show hundreds
of rows), high-contrast status colors, WCAG AA, sortable column headers.
```

---

## 8. Movement: Receive

```
Design a "Receive Inventory" screen for BWIMS, a warehouse inventory web
app. Used by Admin, Warehouse Manager, and Picker roles on desktop or
tablet.

Layout: left sidebar nav, "Movements" highlighted. Top bar: page title
"Receive Inventory", with tabs or a segmented control for the four
movement types (Receive active, Pick / Transfer / Adjust as inactive
tabs).

Form, single column, generous spacing since this may be used on a tablet
with gloves:
1. Product (searchable dropdown, or "Scan barcode" button that opens the
   camera scan flow)
2. Destination Location (searchable dropdown, scoped to the user's
   warehouse)
3. Quantity (large numeric input with +/- stepper buttons)
4. Unit Cost (numeric input, currency-style)
5. Lot Number (text input, only shown/required if the selected product is
   lot-tracked — show a note "This product doesn't require lot tracking"
   if not)
6. Expiration Date (date picker, optional, only relevant if lot-tracked)
7. Reference (text input, optional, e.g. PO number)
8. Notes (textarea, optional)

Below the form: a confirmation summary card that appears before final
submit, restating Product, Lot, Expiry, Location, and Quantity, with
"Cancel" and "Confirm Receive" buttons — this is a stock-changing action
and requires explicit confirmation, not just the form submit.

Success state: a toast/banner showing "Received 50 cases — new balance:
120 cases at Location A-01" with a reference number.

Style: clean utilitarian, large touch targets (min 44x44px, tablet/glove
friendly), high contrast, WCAG AA.
```

---

## 9. Movement: FEFO Pick

```
Design a "Pick Inventory" screen for BWIMS, a warehouse inventory web app.
Used by Admin, Warehouse Manager, and Picker roles, often on a phone on
the warehouse floor.

Layout: same movement-type tab pattern as Receive, "Pick" tab active.

Form:
1. Product (searchable dropdown or "Scan barcode" button)
2. Source Location (searchable dropdown)
3. Quantity to pick (large numeric input with +/- steppers)
4. An optional "Choose specific lot" toggle — when off (default), show a
   note: "System will pick automatically, oldest-expiring stock first
   (FEFO)". When toggled on, show a list of available lots for this
   product at this location, each row showing Lot Number, Expiration Date,
   Available Quantity, sorted soonest-expiring first, as selectable radio
   rows.
5. Reference (optional text)
6. Notes (optional textarea)

After submitting, show a results summary: since one pick can draw from
multiple lots, list each lot consumed as a row (Lot Number, Expiration
Date, Quantity Taken), with a total at the bottom. If any consumed lot was
already expired, show a small amber/red "Expired lot" badge next to that
row with a tooltip "This pick includes stock past its expiration date —
flagged for review", since expired-lot picks are allowed but logged for
audit, not blocked.

Error state: "Not enough available stock" banner if the requested quantity
can't be fulfilled, with the actually-available quantity shown.

Style: clean utilitarian, large touch targets (min 44x44px, this is used
one-handed on a phone), high contrast for warehouse lighting, WCAG AA.
```

---

## 10. Movement: Transfer

```
Design a "Transfer Inventory" screen for BWIMS, a warehouse inventory web
app. Used by Admin and Warehouse Manager only (not Picker).

Layout: same movement-type tab pattern, "Transfer" tab active.

Form, structured as two clearly visually separated sides — "From" and
"To" — with an arrow icon between them:
- From: Location (searchable dropdown)
- To: Location (searchable dropdown) — show inline validation error if the
  same location is selected as both From and To: "Source and destination
  must be different"
- Product (searchable dropdown, shared across both sides)
- Lot (dropdown, required only if the product is lot-tracked, shows
  available quantity per lot at the From location)
- Quantity (numeric input with +/- steppers, capped visually at the
  available quantity for the selected lot/location)
- Reference (optional text)
- Notes (optional textarea)

Confirmation summary before submit: "Move 10 cases of [Product] from
[Location A] to [Location B]" with Cancel/Confirm buttons.

Success state: shows both updated balances side by side (source balance
now lower, destination balance now higher).

Style: clean utilitarian, high contrast, large touch targets, WCAG AA.
```

---

## 11. Movement: Adjustment

```
Design a "Stock Adjustment" screen for BWIMS, a warehouse inventory web
app. Used by Admin and Warehouse Manager only. This is a sensitive,
audit-relevant action (cycle-count corrections, damage write-offs) so the
design should feel deliberately more cautious than Receive/Pick/Transfer.

Layout: same movement-type tab pattern, "Adjust" tab active.

Form:
1. Product (searchable dropdown)
2. Location (searchable dropdown)
3. Lot (dropdown, required only if lot-tracked)
4. Direction: a clear two-option toggle/segmented control, "Increase" vs
   "Decrease" (not a signed number input — make the direction visually
   unambiguous, e.g. green up-arrow for Increase, red down-arrow for
   Decrease)
5. Quantity (numeric input)
6. Notes — make this feel required even though only Quantity/Direction are
   technically required by the API; add helper text "Explain the reason
   for this adjustment (e.g. damaged stock, cycle count correction)"

Show the current balance alongside the form for context: "Current
quantity: 45 cases" so the user can see the before-state while entering
the adjustment.

Confirmation summary before submit, deliberately using cautious language:
"This will [increase/decrease] stock by X cases. This action is permanent
and will be recorded in the audit trail." Cancel/Confirm buttons, Confirm
styled with a warning tone (not the same casual primary-button style as
Receive).

Error states: "Not enough stock to decrease by this amount" for an
over-decrease attempt.

Style: clean utilitarian, high contrast, large touch targets, WCAG AA, but
slightly more restrained/serious visual tone than the other movement
screens given the audit sensitivity.
```

---

## 12. Movement history

```
Design a Movement History screen for BWIMS, a warehouse inventory web app
— the audit trail view, read-only for all roles (though what's visible may
be scoped by warehouse for non-admins).

Layout: left sidebar nav, "Movements" highlighted, a sub-tab or link to
"History" alongside Receive/Pick/Transfer/Adjust. Top bar: filters for
Movement Type (multi-select: Receive/Pick/Transfer/Adjust), Product
(searchable dropdown), Location, and a date range picker.

Table columns: Type (colored badge — distinct colors per movement type),
Product, Lot (or "—"), From Location (or "—" for receive), To Location (or
"—" for pick), Quantity, Unit Cost (or "—" for pick/transfer/adjust),
Reference, Performed By, Timestamp.

This is an immutable audit log — make that implicit in the design by NOT
including any edit or delete row actions, only a "View details" action
that opens a read-only detail panel.

States: empty, loading skeleton, populated with a realistic mix of all 4
movement types.

Style: clean utilitarian, very dense table (this can grow very large over
time), high-contrast status colors per movement type, WCAG AA, sortable
columns, cursor-based "Load more" pagination at the bottom rather than
numbered pages.
```
