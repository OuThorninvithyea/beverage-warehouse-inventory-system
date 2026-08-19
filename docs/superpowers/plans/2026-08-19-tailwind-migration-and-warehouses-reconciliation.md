# Tailwind Migration & Warehouses Figma Reconciliation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate all hand-written CSS in the frontend to Tailwind CSS, and reconcile the Warehouses & Locations screens to match the real Figma design using tokens pulled from Figma.

**Architecture:** Tailwind CSS v4 via its native Vite plugin (CSS-first `@theme` config, no `tailwind.config.js`). Existing screens (Login, Dashboard, Barcode test, AppLayout shell, NotFound) get a 1:1 visual lift-and-shift from hand-written CSS to Tailwind utility classes — same colors/spacing/layout, no redesign. The Warehouses & Locations screens get rebuilt with the real Figma-derived tokens (colors, radius, JetBrains Mono for codes) and structural elements the current build is missing (breadcrumb, icon-only row actions, correct status-badge styling). PrimeVue components keep their own Aura theme styling untouched — Tailwind only styles the page structure around them.

**Tech Stack:** Tailwind CSS v4, `@tailwindcss/vite`, `primeicons` (new — needed for icon-only row actions), Vue 3, PrimeVue 5, Vitest.

Design doc: `docs/superpowers/specs/2026-08-19-tailwind-migration-and-warehouses-reconciliation-design.md`.

---

## File Structure

```
frontend/
  vite.config.ts                        # MODIFY — add @tailwindcss/vite plugin
  index.html                             # MODIFY — add Google Fonts link (Inter + JetBrains Mono)
  package.json                           # MODIFY — add tailwindcss, @tailwindcss/vite, primeicons
  src/
    main.ts                              # MODIFY — import primeicons CSS
    assets/main.css                      # REWRITE — @theme tokens + minimal base layer, all component CSS removed
    layouts/AppLayout.vue                # MODIFY — Tailwind utilities (lift-and-shift)
    views/LoginView.vue                  # MODIFY — Tailwind utilities (lift-and-shift)
    views/DashboardView.vue              # MODIFY — Tailwind utilities (lift-and-shift)
    views/BarcodeTestView.vue            # MODIFY — Tailwind utilities (lift-and-shift)
    views/NotFoundView.vue               # MODIFY — Tailwind utilities (lift-and-shift)
    views/WarehousesListView.vue         # MODIFY — Figma reconciliation
    views/WarehouseDetailView.vue        # MODIFY — Figma reconciliation
    components/WarehouseFormDialog.vue   # MODIFY — scoped CSS → Tailwind utilities
    components/LocationFormDialog.vue    # MODIFY — scoped CSS → Tailwind utilities
```

---

## Task 1: Install Tailwind, configure the design tokens

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/vite.config.ts`
- Modify: `frontend/index.html`
- Modify: `frontend/src/main.ts`
- Rewrite: `frontend/src/assets/main.css`

- [ ] **Step 1: Install dependencies**

```bash
cd frontend
npm install -D tailwindcss @tailwindcss/vite
npm install primeicons
```

- [ ] **Step 2: Add the Tailwind Vite plugin**

Open `frontend/vite.config.ts`. Add the import and plugin, leaving everything else (the `test` block added in the Warehouses slice) untouched:

```ts
import { fileURLToPath, URL } from 'node:url'

import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      '/health': 'http://localhost:8080',
      '/ready': 'http://localhost:8080',
    },
  },
  test: {
    environment: 'jsdom',
  },
})
```

- [ ] **Step 3: Add web fonts**

Open `frontend/index.html`. Add a Google Fonts link for Inter and JetBrains Mono in `<head>`:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta name="description" content="Beverage Warehouse Inventory Management System" />
    <title>BWIMS</title>
    <link rel="preconnect" href="https://fonts.googleapis.com" />
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
    <link
      href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500&display=swap"
      rel="stylesheet"
    />
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
```

- [ ] **Step 4: Import primeicons CSS**

Open `frontend/src/main.ts`. Add the import alongside the existing `main.css` import:

```ts
import Aura from '@primeuix/themes/aura'
import { createPinia } from 'pinia'
import PrimeVue from 'primevue/config'
import { createApp } from 'vue'

import App from './App.vue'
import router from './router'
import './assets/main.css'
import 'primeicons/primeicons.css'

const pinia = createPinia()

createApp(App)
  .use(pinia)
  .use(PrimeVue, {
    theme: {
      preset: Aura,
      options: {
        darkModeSelector: '.bwims-dark',
      },
    },
  })
  .use(router)
  .mount('#app')
```

- [ ] **Step 5: Rewrite `main.css` with the Tailwind import and design tokens**

Replace the entire contents of `frontend/src/assets/main.css`:

```css
@import "tailwindcss";

@theme {
  /* Figma-derived tokens (Warehouses & Locations reconciliation) */
  --color-primary: #316bf3;
  --color-primary-hover: #0051d5;
  --color-ink: #1b1b1d;
  --color-ink-muted: #45474c;
  --color-ink-faint: #8590a6;
  --color-success-bg: #dcfce7;
  --color-success-text: #16a34a;
  --color-danger-bg: #fee2e2;
  --color-danger-text: #dc2626;
  --color-warning-bg: #fef3c7;
  --color-warning-text: #d97706;
  --color-border: #e2e8f0;
  --color-border-strong: #c5c6cd;
  --color-surface: #f8fafc;
  --font-mono-code: "JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, monospace;

  /* Existing brand tokens (lift-and-shift screens, values preserved exactly) */
  --color-brand-navy: #0f2947;
  --color-brand-amber: #ffbd59;
  --color-brand-ink: #172033;
  --color-brand-surface: #f4f7fb;
  --color-brand-muted: #65738a;

  --font-sans: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}

@layer base {
  body {
    font-family: var(--font-sans);
    color: var(--color-brand-ink);
    background: var(--color-brand-surface);
    min-width: 320px;
    min-height: 100vh;
    font-synthesis: none;
    text-rendering: optimizeLegibility;
  }

  a {
    color: inherit;
    text-decoration: none;
  }
}
```

- [ ] **Step 6: Verify the build still runs**

Run: `cd frontend && npm run build`
Expected: PASS. The app will look broken/unstyled at this point since no template has been converted yet — that's expected; this step only confirms Tailwind itself compiles without errors.

- [ ] **Step 7: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/vite.config.ts frontend/index.html frontend/src/main.ts frontend/src/assets/main.css
git commit -m "chore: install Tailwind CSS and define design tokens from Figma"
```

---

## Task 2: Migrate `AppLayout.vue`

**Files:**
- Modify: `frontend/src/layouts/AppLayout.vue`

- [ ] **Step 1: Replace the template with Tailwind utility classes**

```vue
<script setup lang="ts">
import Button from 'primevue/button'
import { RouterLink, RouterView } from 'vue-router'
import { useRouter } from 'vue-router'

import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

async function signOut() {
  await auth.signOut()
  await router.push({ name: 'login' })
}
</script>

<template>
  <div class="grid min-h-screen grid-cols-[260px_minmax(0,1fr)] max-[800px]:grid-cols-1">
    <aside class="flex flex-col gap-8 bg-brand-navy p-6 text-[#f7f9fc] max-[800px]:gap-4">
      <div class="flex items-center gap-3">
        <span
          class="grid h-[42px] w-[42px] place-items-center rounded-[12px] bg-brand-amber font-extrabold text-brand-navy"
        >BW</span>
        <div class="grid">
          <strong>BWIMS</strong>
          <small class="text-[#a8bdd5]">Warehouse control</small>
        </div>
      </div>

      <nav
        aria-label="Primary navigation"
        class="grid gap-[0.4rem] max-[800px]:grid-cols-4 max-[800px]:overflow-x-auto max-[520px]:grid-cols-2"
      >
        <RouterLink
          to="/"
          class="rounded-[10px] px-[0.9rem] py-[0.8rem] [&.router-link-active]:bg-brand-amber [&.router-link-active]:font-bold [&.router-link-active]:text-brand-ink"
        >Dashboard</RouterLink>
        <RouterLink
          to="/warehouses"
          class="rounded-[10px] px-[0.9rem] py-[0.8rem] [&.router-link-active]:bg-brand-amber [&.router-link-active]:font-bold [&.router-link-active]:text-brand-ink"
        >Warehouses</RouterLink>
        <RouterLink
          to="/barcode-test"
          class="rounded-[10px] px-[0.9rem] py-[0.8rem] [&.router-link-active]:bg-brand-amber [&.router-link-active]:font-bold [&.router-link-active]:text-brand-ink"
        >Barcode test</RouterLink>
        <span class="cursor-not-allowed rounded-[10px] px-[0.9rem] py-[0.8rem] text-[#9fb1c6]">Inventory</span>
        <span class="cursor-not-allowed rounded-[10px] px-[0.9rem] py-[0.8rem] text-[#9fb1c6]">Movements</span>
      </nav>

      <div
        class="mt-auto grid gap-[0.35rem] rounded-[12px] border border-white/12 bg-white/5 p-4 text-[#dce7f2] max-[800px]:hidden"
      >
        <small class="text-[#a8bdd5]">Week 11</small>
        <span>Backend complete through inventory & movements. Warehouses is the first live frontend screen.</span>
      </div>
    </aside>

    <div class="min-w-0">
      <header
        class="flex min-h-[76px] items-center justify-between border-b border-[#dde4ee] bg-white px-8 py-4 max-[520px]:px-4"
      >
        <div class="grid gap-[0.2rem]">
          <small class="uppercase tracking-[0.08em] text-brand-muted">Beverage warehouse</small>
          <strong class="max-[520px]:text-[0.9rem]">Inventory Management System</strong>
        </div>
        <div class="flex items-center gap-[0.8rem]">
          <span v-if="auth.user" class="grid text-right text-[0.9rem] font-[650] text-brand-ink">
            {{ auth.user.full_name }}
            <small class="font-medium capitalize text-brand-muted">{{ auth.user.role.replace('_', ' ') }}</small>
          </span>
          <Button label="Sign out" size="small" severity="secondary" @click="signOut" />
        </div>
      </header>

      <main class="p-[clamp(1.25rem,3vw,2.5rem)]">
        <RouterView />
      </main>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Run tests to verify nothing broke**

Run: `cd frontend && npx vitest run`
Expected: PASS (all 42 tests — this view isn't directly unit-tested, but other views mount within `AppLayout`-adjacent contexts should be unaffected).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/layouts/AppLayout.vue
git commit -m "refactor: migrate AppLayout.vue to Tailwind utilities"
```

---

## Task 3: Migrate `LoginView.vue`

**Files:**
- Modify: `frontend/src/views/LoginView.vue`

- [ ] **Step 1: Replace the template**

Only the `<template>` block changes — the `<script setup>` block stays exactly as-is.

```vue
<template>
  <main
    class="grid min-h-screen place-items-center content-center bg-brand-navy p-6"
    style="background-image: radial-gradient(circle at top left, rgb(255 189 89 / 35%), transparent 35%)"
  >
    <Card class="w-full max-w-[430px]">
      <template #title>Welcome to BWIMS</template>
      <template #subtitle>Use your assigned warehouse account</template>
      <template #content>
        <form class="grid gap-3" @submit.prevent="submit">
          <label for="email" class="mt-[0.4rem] font-[650]">Email address</label>
          <InputText id="email" v-model="email" type="email" autocomplete="email" />

          <label for="password" class="mt-[0.4rem] font-[650]">Password</label>
          <InputText
            id="password"
            v-model="password"
            type="password"
            autocomplete="current-password"
          />

          <Message v-if="errorMessage" severity="error">{{ errorMessage }}</Message>
          <Button
            type="submit"
            label="Sign in"
            :loading="auth.loading"
            :disabled="!email.trim() || !password"
          />
          <small class="text-center text-brand-muted">Access is controlled by your admin, manager, picker or viewer role.</small>
        </form>
      </template>
    </Card>
  </main>
</template>
```

- [ ] **Step 2: Run tests to verify nothing broke**

Run: `cd frontend && npx vitest run`
Expected: PASS (42 tests — no test mounts `LoginView` directly, but confirms no build-wide regression).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/LoginView.vue
git commit -m "refactor: migrate LoginView.vue to Tailwind utilities"
```

---

## Task 4: Migrate `DashboardView.vue`

**Files:**
- Modify: `frontend/src/views/DashboardView.vue`

- [ ] **Step 1: Replace the template**

Only the `<template>` block changes.

```vue
<template>
  <section>
    <div class="mb-6 flex items-start justify-between">
      <div>
        <span class="inline-flex rounded-full bg-[#dff8eb] px-[0.65rem] py-[0.35rem] text-[0.8rem] font-bold text-[#176c43]">Foundation ready</span>
        <h1 class="mb-[0.35rem] mt-3 text-[clamp(1.8rem,4vw,2.6rem)]">Inventory overview</h1>
        <p class="m-0 text-brand-muted">The shell is ready for live inventory, movement, and expiry data.</p>
      </div>
    </div>

    <div class="mb-4 grid grid-cols-3 gap-4 max-[800px]:grid-cols-1">
      <Card v-for="item in foundationItems" :key="item.label">
        <template #content>
          <div class="grid gap-[0.3rem]">
            <strong class="text-[2rem] text-brand-navy">{{ item.value }}</strong>
            <span class="text-brand-muted">{{ item.label }}</span>
          </div>
        </template>
      </Card>
    </div>

    <Card class="mt-4">
      <template #title>Implementation path</template>
      <template #content>
        <div class="grid grid-cols-3 gap-4 max-[800px]:grid-cols-1">
          <div class="grid gap-[0.4rem] rounded-[12px] bg-brand-surface p-4">
            <span class="text-[0.82rem] font-bold text-[#b86600]">Weeks 7–8</span>
            <strong>Authentication and master data</strong>
          </div>
          <div class="grid gap-[0.4rem] rounded-[12px] bg-brand-surface p-4">
            <span class="text-[0.82rem] font-bold text-[#b86600]">Weeks 9–10</span>
            <strong>Inventory and stock movements</strong>
          </div>
          <div class="grid gap-[0.4rem] rounded-[12px] bg-brand-surface p-4">
            <span class="text-[0.82rem] font-bold text-[#b86600]">Weeks 11–12</span>
            <strong>Alerts, reports, and dashboards</strong>
          </div>
        </div>
      </template>
    </Card>
  </section>
</template>
```

- [ ] **Step 2: Run tests to verify nothing broke**

Run: `cd frontend && npx vitest run`
Expected: PASS (42 tests).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/DashboardView.vue
git commit -m "refactor: migrate DashboardView.vue to Tailwind utilities"
```

---

## Task 5: Migrate `BarcodeTestView.vue`

**Files:**
- Modify: `frontend/src/views/BarcodeTestView.vue`

- [ ] **Step 1: Replace the template**

Only the `<template>` block changes — the `<script setup>` block stays exactly as-is.

```vue
<template>
  <section>
    <div class="mb-6 flex items-start justify-between">
      <div>
        <span class="inline-flex rounded-full bg-[#dff8eb] px-[0.65rem] py-[0.35rem] text-[0.8rem] font-bold text-[#176c43]">Week 6 device test</span>
        <h1 class="mb-[0.35rem] mt-3 text-[clamp(1.8rem,4vw,2.6rem)]">Barcode camera validation</h1>
        <p class="m-0 text-brand-muted">Decode and validate a barcode without changing inventory.</p>
      </div>
    </div>

    <div class="grid grid-cols-2 gap-4 max-[800px]:grid-cols-1">
      <Card>
        <template #title>Camera scanner</template>
        <template #content>
          <div class="grid gap-[0.85rem]">
            <video ref="video" class="min-h-[280px] w-full rounded-[12px] bg-[#091a2d] object-cover" muted playsinline />

            <label for="camera">Camera</label>
            <Select
              id="camera"
              v-model="selectedCamera"
              :options="cameras"
              option-label="label"
              option-value="value"
              placeholder="Choose a camera"
              :disabled="scanning"
            />

            <div class="flex flex-wrap gap-[0.65rem]">
              <Button v-if="!scanning" label="Start camera" @click="startScanner" />
              <Button
                v-else
                label="Stop camera"
                severity="secondary"
                @click="stopScanner"
              />
              <Button
                v-if="locked"
                label="Rescan"
                severity="secondary"
                outlined
                @click="rescan"
              />
            </div>

            <Message v-if="cameraError" severity="error">{{ cameraError }}</Message>
            <small>
              Camera access requires HTTPS, except on <code>localhost</code>.
            </small>
          </div>
        </template>
      </Card>

      <Card>
        <template #title>Captured value</template>
        <template #content>
          <div class="grid gap-[0.85rem]">
            <label for="manual-barcode">Manual or USB scanner input</label>
            <InputText
              id="manual-barcode"
              v-model="manualValue"
              inputmode="numeric"
              autocomplete="off"
              placeholder="Scan or enter EAN-13 / UPC-A"
              @keyup.enter="useManualValue"
            />
            <Button
              label="Validate value"
              severity="secondary"
              :disabled="manualValue.trim().length === 0"
              @click="useManualValue"
            />

            <div v-if="candidate" class="grid gap-[0.4rem] rounded-[12px] bg-brand-surface p-4">
              <small>Captured barcode</small>
              <strong class="overflow-wrap-anywhere text-[1.4rem] tracking-[0.08em] text-brand-navy">{{ validation.normalized }}</strong>
              <Message :severity="validation.valid ? 'success' : 'warn'">
                {{ validation.message }}
              </Message>
            </div>

            <Message severity="info">
              This screen only produces a lookup value. It cannot receive, pick,
              transfer or adjust inventory.
            </Message>
          </div>
        </template>
      </Card>
    </div>
  </section>
</template>
```

Note: `overflow-wrap-anywhere` isn't a default Tailwind utility — it's included via Tailwind v4's expanded typography utilities, but if the build reports it unknown, replace it with the arbitrary form `[overflow-wrap:anywhere]` instead.

- [ ] **Step 2: Run tests to verify nothing broke**

Run: `cd frontend && npx vitest run`
Expected: PASS (42 tests).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/BarcodeTestView.vue
git commit -m "refactor: migrate BarcodeTestView.vue to Tailwind utilities"
```

---

## Task 6: Migrate `NotFoundView.vue`

**Files:**
- Modify: `frontend/src/views/NotFoundView.vue`

- [ ] **Step 1: Replace the template**

```vue
<template>
  <main
    class="grid min-h-screen place-items-center content-center bg-brand-navy p-6 text-center text-white"
    style="background-image: radial-gradient(circle at top left, rgb(255 189 89 / 35%), transparent 35%)"
  >
    <span class="text-[4rem] font-extrabold text-brand-amber">404</span>
    <h1 class="my-2">Page not found</h1>
    <p class="mb-5 text-[#c6d4e2]">The page may not be part of the current BWIMS milestone.</p>
    <Button label="Return to dashboard" as="router-link" to="/" />
  </main>
</template>
```

- [ ] **Step 2: Run tests to verify nothing broke**

Run: `cd frontend && npx vitest run`
Expected: PASS (42 tests).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/NotFoundView.vue
git commit -m "refactor: migrate NotFoundView.vue to Tailwind utilities"
```

---

## Task 7: Reconcile `WarehousesListView.vue` with Figma

**Files:**
- Modify: `frontend/src/views/WarehousesListView.vue`

- [ ] **Step 1: Replace the template and remove the scoped `<style>` block**

The `<script setup>` block is unchanged. Replace the `<template>` and delete the trailing `<style scoped>` block entirely.

```vue
<template>
  <div class="flex flex-col gap-4">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-semibold text-ink">Warehouses</h1>
      <Button
        v-if="canManageWarehouses"
        label="Add Warehouse"
        data-testid="add-warehouse"
        @click="openCreateDialog"
      />
    </div>

    <InputText
      v-model="search"
      placeholder="Search warehouses..."
      data-testid="warehouse-search"
      class="max-w-[320px]"
    />

    <p v-if="store.error" class="text-danger-text" data-testid="warehouses-error">{{ store.error }}</p>

    <DataTable
      v-else
      :value="store.warehouses"
      :loading="store.loading"
      data-key="id"
      class="overflow-hidden rounded-[12px] border border-border"
      @row-click="(event: { data: Warehouse }) => openDetail(event.data)"
    >
      <template #empty>
        <p>No warehouses yet.</p>
      </template>
      <Column field="code" header="Code">
        <template #body="{ data }">
          <span class="font-mono-code text-[0.85rem]">{{ data.code }}</span>
        </template>
      </Column>
      <Column field="name" header="Name" />
      <Column field="address" header="Address" />
      <Column header="Status">
        <template #body="{ data }">
          <span
            class="rounded-badge px-2 py-1 text-xs font-medium"
            :class="data.is_active ? 'bg-success-bg text-success-text' : 'bg-danger-bg text-danger-text'"
          >{{ data.is_active ? 'Active' : 'Inactive' }}</span>
        </template>
      </Column>
      <Column v-if="canManageWarehouses" header="Actions">
        <template #body="{ data }">
          <Button
            icon="pi pi-pencil"
            size="small"
            severity="secondary"
            text
            rounded
            aria-label="Edit warehouse"
            data-testid="edit-warehouse"
            @click.stop="openEditDialog(data)"
          />
          <Button
            v-if="data.is_active"
            icon="pi pi-ban"
            size="small"
            severity="danger"
            text
            rounded
            aria-label="Deactivate warehouse"
            data-testid="deactivate-warehouse"
            @click.stop="deactivate(data)"
          />
        </template>
      </Column>
    </DataTable>

    <Button
      v-if="store.hasMore"
      label="Load more"
      severity="secondary"
      data-testid="load-more"
      @click="store.loadMoreWarehouses(search)"
    />

    <WarehouseFormDialog v-model:visible="dialogVisible" :warehouse="editingWarehouse" />
  </div>
</template>
```

`font-mono-code` and `rounded-badge` reference the `--font-mono-code` and `--radius-badge` theme tokens defined in Task 1 — Tailwind v4 auto-generates a `font-mono-code`/`rounded-badge` utility from any `--font-*`/`--radius-*` custom property in `@theme`.

- [ ] **Step 2: Run tests to verify nothing broke**

Run: `cd frontend && npx vitest run src/views/WarehousesListView.spec.ts`
Expected: PASS (4 tests — none assert on button text or CSS classes, only `data-testid` and cell text).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/WarehousesListView.vue
git commit -m "refactor: reconcile WarehousesListView.vue with Figma design tokens"
```

---

## Task 8: Reconcile `WarehouseDetailView.vue` with Figma

**Files:**
- Modify: `frontend/src/views/WarehouseDetailView.vue`

- [ ] **Step 1: Replace the template and remove the scoped `<style>` block**

The `<script setup>` block is unchanged. Replace the `<template>` and delete the trailing `<style scoped>` block entirely.

```vue
<template>
  <div class="flex flex-col gap-4">
    <p v-if="warehouseError" class="text-danger-text" data-testid="warehouse-error">{{ warehouseError }}</p>
    <template v-else-if="warehouse">
      <nav class="text-sm text-ink-faint">
        <RouterLink to="/warehouses" class="hover:underline">Warehouses</RouterLink>
        <span class="mx-1">›</span>
        <span class="text-ink">{{ warehouse.name }}</span>
      </nav>

      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-ink">{{ warehouse.name }}</h1>
          <p class="text-ink-muted">
            <span class="font-mono-code">{{ warehouse.code }}</span>
            · {{ warehouse.address ?? 'No address on file' }}
          </p>
        </div>
        <Button
          v-if="canManageWarehouses"
          label="Edit Warehouse"
          data-testid="edit-warehouse"
          @click="warehouseDialogVisible = true"
        />
      </div>

      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <h2 class="text-lg font-semibold text-ink">Locations</h2>
          <span class="rounded-full bg-surface px-2 py-0.5 text-xs font-medium text-ink-faint">
            {{ store.locations.length }} shown
          </span>
        </div>
        <Button
          v-if="canManageLocations"
          label="Add Location"
          data-testid="add-location"
          @click="openCreateLocationDialog"
        />
      </div>

      <p v-if="store.locationsError" class="text-danger-text" data-testid="locations-error">
        {{ store.locationsError }}
      </p>
      <DataTable
        v-else
        :value="store.locations"
        :loading="store.locationsLoading"
        data-key="id"
        class="overflow-hidden rounded-[12px] border border-border"
      >
        <template #empty>
          <p>No locations yet.</p>
        </template>
        <Column field="code" header="Location code">
          <template #body="{ data }">
            <span class="font-mono-code text-[0.85rem]">{{ data.code }}</span>
          </template>
        </Column>
        <Column field="zone" header="Zone" />
        <Column field="aisle" header="Aisle" />
        <Column field="rack" header="Rack" />
        <Column field="shelf" header="Shelf" />
        <Column field="barcode" header="Barcode">
          <template #body="{ data }">
            <span v-if="data.barcode" class="font-mono-code text-[0.85rem]">{{ data.barcode }}</span>
            <span v-else class="text-ink-faint">—</span>
          </template>
        </Column>
        <Column header="Pickable">
          <template #body="{ data }">
            <span
              class="rounded-badge px-2 py-1 text-xs font-medium"
              :class="data.is_pickable ? 'bg-success-bg text-success-text' : 'bg-danger-bg text-danger-text'"
            >{{ data.is_pickable ? 'Yes' : 'No' }}</span>
          </template>
        </Column>
        <Column header="Status">
          <template #body="{ data }">
            <span
              class="rounded-badge px-2 py-1 text-xs font-medium"
              :class="data.is_active ? 'bg-success-bg text-success-text' : 'bg-danger-bg text-danger-text'"
            >{{ data.is_active ? 'Active' : 'Inactive' }}</span>
          </template>
        </Column>
        <Column v-if="canManageLocations" header="Actions">
          <template #body="{ data }">
            <Button
              icon="pi pi-pencil"
              size="small"
              severity="secondary"
              text
              rounded
              aria-label="Edit location"
              data-testid="edit-location"
              @click="openEditLocationDialog(data)"
            />
            <Button
              v-if="data.is_active"
              icon="pi pi-ban"
              size="small"
              severity="danger"
              text
              rounded
              aria-label="Deactivate location"
              data-testid="deactivate-location"
              @click="deactivateLocationRow(data)"
            />
          </template>
        </Column>
      </DataTable>

      <Button
        v-if="store.locationsHasMore"
        label="Load more"
        severity="secondary"
        data-testid="load-more-locations"
        @click="store.loadMoreLocations(warehouseId)"
      />

      <WarehouseFormDialog
        v-model:visible="warehouseDialogVisible"
        :warehouse="warehouse"
        @saved="handleWarehouseSaved"
      />
      <LocationFormDialog
        v-model:visible="locationDialogVisible"
        :warehouse-id="warehouseId"
        :location="editingLocation"
      />
    </template>
  </div>
</template>
```

Add `RouterLink` to the existing `vue-router` import in `<script setup>`:

```ts
import { useRoute, RouterLink } from 'vue-router'
```

(Combine with whatever's already imported from `vue-router` in that file — currently just `useRoute`.)

- [ ] **Step 2: Run tests to verify nothing broke**

Run: `cd frontend && npx vitest run src/views/WarehouseDetailView.spec.ts`
Expected: PASS (4 tests). The breadcrumb's `RouterLink` needs `vue-router`'s mocked module (already mocked in the spec) to also export `RouterLink` — check the mock:

```ts
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { warehouseId: 'wh-1' } }),
}))
```

This mock doesn't include `RouterLink`, so add it as a stub in the same mock so the component doesn't crash on an undefined component:

```ts
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { warehouseId: 'wh-1' } }),
  RouterLink: { template: '<a><slot /></a>' },
}))
```

Update `frontend/src/views/WarehouseDetailView.spec.ts` with this addition, then re-run the test.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/WarehouseDetailView.vue frontend/src/views/WarehouseDetailView.spec.ts
git commit -m "refactor: reconcile WarehouseDetailView.vue with Figma design tokens"
```

---

## Task 9: Migrate the form dialogs' scoped CSS to Tailwind

**Files:**
- Modify: `frontend/src/components/WarehouseFormDialog.vue`
- Modify: `frontend/src/components/LocationFormDialog.vue`

- [ ] **Step 1: Update `WarehouseFormDialog.vue`**

Replace every `<div class="form-field">` with `<div class="mb-4 flex flex-col gap-1">` and every `<div class="form-field form-field--inline">` with `<div class="mb-4 flex flex-row items-center gap-3">`. Delete the trailing `<style scoped>` block entirely.

The four occurrences in this file (code, name, address, active) become:

```vue
<template>
  <Dialog
    :visible="visible"
    modal
    :header="isEdit ? 'Edit warehouse' : 'Add warehouse'"
    @update:visible="(value: boolean) => emit('update:visible', value)"
  >
    <div class="mb-4 flex flex-col gap-1">
      <label for="warehouse-code">Code</label>
      <InputText id="warehouse-code" v-model="form.code" :invalid="codeError !== null" />
      <Message v-if="codeError" severity="error" size="small" variant="simple">{{ codeError }}</Message>
    </div>
    <div class="mb-4 flex flex-col gap-1">
      <label for="warehouse-name">Name</label>
      <InputText id="warehouse-name" v-model="form.name" :invalid="nameError !== null" />
      <Message v-if="nameError" severity="error" size="small" variant="simple">{{ nameError }}</Message>
    </div>
    <div class="mb-4 flex flex-col gap-1">
      <label for="warehouse-address">Address</label>
      <InputText id="warehouse-address" v-model="form.address" />
    </div>
    <div class="mb-4 flex flex-row items-center gap-3">
      <label for="warehouse-active">Active</label>
      <ToggleSwitch id="warehouse-active" v-model="form.is_active" />
    </div>
    <Message v-if="generalError" severity="error" size="small">{{ generalError }}</Message>
    <template #footer>
      <Button label="Cancel" severity="secondary" data-testid="cancel" @click="emit('update:visible', false)" />
      <Button label="Save" :loading="submitting" data-testid="submit" @click="submit" />
    </template>
  </Dialog>
</template>
```

(The `<script setup>` block is unchanged; only the `<template>` markup and removal of `<style scoped>` change.)

- [ ] **Step 2: Update `LocationFormDialog.vue`**

Same substitution pattern across all seven fields (code, zone, aisle, rack, shelf, barcode, pickable, active):

```vue
<template>
  <Dialog
    :visible="visible"
    modal
    :header="isEdit ? 'Edit location' : 'Add location'"
    @update:visible="(value: boolean) => emit('update:visible', value)"
  >
    <div class="mb-4 flex flex-col gap-1">
      <label for="location-code">Code</label>
      <InputText id="location-code" v-model="form.code" :invalid="codeError !== null" />
      <Message v-if="codeError" severity="error" size="small" variant="simple">{{ codeError }}</Message>
    </div>
    <div class="mb-4 flex flex-col gap-1">
      <label for="location-zone">Zone</label>
      <InputText id="location-zone" v-model="form.zone" />
    </div>
    <div class="mb-4 flex flex-col gap-1">
      <label for="location-aisle">Aisle</label>
      <InputText id="location-aisle" v-model="form.aisle" />
    </div>
    <div class="mb-4 flex flex-col gap-1">
      <label for="location-rack">Rack</label>
      <InputText id="location-rack" v-model="form.rack" />
    </div>
    <div class="mb-4 flex flex-col gap-1">
      <label for="location-shelf">Shelf</label>
      <InputText id="location-shelf" v-model="form.shelf" />
    </div>
    <div class="mb-4 flex flex-col gap-1">
      <label for="location-barcode">Barcode</label>
      <InputText id="location-barcode" v-model="form.barcode" :invalid="barcodeError !== null" />
      <Message v-if="barcodeError" severity="error" size="small" variant="simple">{{ barcodeError }}</Message>
      <small v-else-if="barcodeHint" data-testid="barcode-hint">{{ barcodeHint }}</small>
    </div>
    <div class="mb-4 flex flex-row items-center gap-3">
      <label for="location-pickable">Pickable</label>
      <ToggleSwitch id="location-pickable" v-model="form.is_pickable" />
    </div>
    <div class="mb-4 flex flex-row items-center gap-3">
      <label for="location-active">Active</label>
      <ToggleSwitch id="location-active" v-model="form.is_active" />
    </div>
    <Message v-if="generalError" severity="error" size="small">{{ generalError }}</Message>
    <template #footer>
      <Button label="Cancel" severity="secondary" data-testid="cancel" @click="emit('update:visible', false)" />
      <Button label="Save" :loading="submitting" data-testid="submit" @click="submit" />
    </template>
  </Dialog>
</template>
```

(The `<script setup>` block is unchanged.)

- [ ] **Step 3: Run tests to verify nothing broke**

Run: `cd frontend && npx vitest run src/components/WarehouseFormDialog.spec.ts src/components/LocationFormDialog.spec.ts`
Expected: PASS (8 tests — all assert on `data-testid`, field values, and error text, none on CSS classes).

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/WarehouseFormDialog.vue frontend/src/components/LocationFormDialog.vue
git commit -m "refactor: migrate form dialog layouts to Tailwind utilities"
```

---

## Task 10: Final validation

**Files:** none (verification only)

- [ ] **Step 1: Run the full frontend test suite**

Run: `cd frontend && npm test`
Expected: PASS — all 42 tests (adjusted count if Task 8's `RouterLink` mock addition changes nothing test-count-wise, still 42).

- [ ] **Step 2: Typecheck**

Run: `cd frontend && npm run typecheck`
Expected: PASS.

- [ ] **Step 3: Build**

Run: `cd frontend && npm run build`
Expected: PASS. Check the build output doesn't reference any unresolved Tailwind class (Tailwind v4 silently ignores unknown arbitrary syntax rather than erroring, so also visually spot-check in Step 4).

- [ ] **Step 4: Manual visual verification**

Start the app (`npm run dev`, proxying to the already-running dockerized backend on port 8080) and compare each screen against:
- Pre-migration behavior for Login, Dashboard, Barcode test, AppLayout shell, NotFound — should look visually identical to before this plan (same colors, spacing, layout at desktop and the 800px/520px breakpoints).
- The Figma screenshot (node `1:310`) for Warehouses & Locations — breadcrumb, monospace codes, status badge colors, icon-only row actions should now be visibly present.

Log in as the seeded admin (`admin@bwims.local` / `ChangeMe123!`), walk through: dashboard → warehouses list → warehouse detail → add a location, confirming no visual regressions and no console errors.

- [ ] **Step 5: No commit for this task** — verification checkpoint only.

---

## Self-Review Notes

**Spec coverage check against the design doc:**
- Tailwind v4 with native Vite plugin, CSS-first `@theme` — Task 1. ✓
- Figma-derived tokens (primary, ink tones, success/danger/warning, border, surface, radius, JetBrains Mono) — Task 1's `@theme` block, matches the design doc's token list exactly. ✓
- Existing brand tokens preserved exactly for lift-and-shift screens (navy, amber, ink, surface, muted) — Task 1, then used unchanged in Tasks 2–6. ✓
- PrimeVue components untouched, only page structure restyled — every task only changes `<template>` layout markup around PrimeVue components, never their internal props/theme. ✓
- Breadcrumb on warehouse detail — Task 8. ✓
- JetBrains Mono for codes — Tasks 7 and 8 (`font-mono-code` utility). ✓
- Icon-only row actions — Tasks 7 and 8 (`pi pi-pencil`/`pi pi-ban` icon Buttons replacing text Buttons). ✓
- Status badges using the real success/danger palette — Tasks 7 and 8. ✓
- "N shown" count instead of a fabricated total — Task 8 (`{{ store.locations.length }} shown`). ✓
- Barcode column kept — Task 8, column not removed. ✓
- No stat cards / no "Blocked (Audit)" status / no global topbar redesign — none of these appear in any task; explicitly out of scope per the design doc. ✓
- Google Fonts for Inter + JetBrains Mono — Task 1, Step 3. ✓

**Type/signature consistency check:** `font-mono-code`, `rounded-badge`, `text-danger-text`, `bg-success-bg`, etc. are Tailwind utilities auto-generated from the `--font-mono-code`, `--radius-badge`, `--color-danger-text`, `--color-success-bg` custom properties defined once in Task 1's `@theme` block, and used identically (same utility class names) across Tasks 7, 8, and 9 — no renamed tokens between tasks.

**Placeholder scan:** no TBD/TODO; every step has complete, copy-pasteable code.
