# Dashboard Figma Style Reconciliation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restyle `DashboardView.vue`'s existing content to match the Figma "Operational Dashboard" visual language — style only, no new data.

**Architecture:** Add two more Figma-derived status tokens to `main.css`'s `@theme` block (info blue, accent purple — for future Movements screen use). Restyle the 3 stat cards and 3 roadmap cards in `DashboardView.vue` to Figma's bordered-card look, reorder the stat cards' label/value hierarchy, and switch the two one-off color pairs (`#dff8eb`/`#176c43` pill, `#b86600` roadmap label) to existing tokens.

**Tech Stack:** Tailwind CSS v4 (already installed), existing `@theme` tokens from the Warehouses reconciliation.

Design doc: `docs/superpowers/specs/2026-08-19-dashboard-figma-style-design.md`.

---

## Task 1: Add the two new status tokens

**Files:**
- Modify: `frontend/src/assets/main.css`

- [ ] **Step 1: Add the tokens**

In the `@theme` block, add these two lines after `--color-warning-text: #d97706;`:

```css
  --color-info-bg: #dbeafe;
  --color-info-text: #2563eb;
  --color-accent-bg: #f3e8ff;
  --color-accent-text: #9333ea;
```

- [ ] **Step 2: Verify the build still compiles**

Run: `cd frontend && npm run build`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/assets/main.css
git commit -m "chore: add info/accent status tokens from Figma"
```

---

## Task 2: Restyle `DashboardView.vue`

**Files:**
- Modify: `frontend/src/views/DashboardView.vue`

- [ ] **Step 1: Replace the template**

The `<script setup>` block (the `foundationItems` array) is unchanged.

```vue
<template>
  <section>
    <div class="mb-6 flex items-start justify-between">
      <div>
        <span class="inline-flex rounded-full bg-success-bg px-[0.65rem] py-[0.35rem] text-[0.8rem] font-bold text-success-text">Foundation ready</span>
        <h1 class="mb-[0.35rem] mt-3 text-[clamp(1.8rem,4vw,2.6rem)]">Inventory overview</h1>
        <p class="m-0 text-brand-muted">The shell is ready for live inventory, movement, and expiry data.</p>
      </div>
    </div>

    <div class="mb-4 grid grid-cols-3 gap-4 max-[800px]:grid-cols-1">
      <Card
        v-for="item in foundationItems"
        :key="item.label"
        class="rounded-[12px] border border-border shadow-none"
      >
        <template #content>
          <div class="grid gap-2">
            <span class="text-xs font-medium uppercase tracking-wide text-ink-faint">{{ item.label }}</span>
            <strong class="text-[2rem] text-ink">{{ item.value }}</strong>
          </div>
        </template>
      </Card>
    </div>

    <Card class="mt-4 rounded-[12px] border border-border shadow-none">
      <template #title>Implementation path</template>
      <template #content>
        <div class="grid grid-cols-3 gap-4 max-[800px]:grid-cols-1">
          <div class="grid gap-[0.4rem] rounded-[12px] border border-border p-4">
            <span class="text-[0.82rem] font-bold text-warning-text">Weeks 7–8</span>
            <strong class="text-ink">Authentication and master data</strong>
          </div>
          <div class="grid gap-[0.4rem] rounded-[12px] border border-border p-4">
            <span class="text-[0.82rem] font-bold text-warning-text">Weeks 9–10</span>
            <strong class="text-ink">Inventory and stock movements</strong>
          </div>
          <div class="grid gap-[0.4rem] rounded-[12px] border border-border p-4">
            <span class="text-[0.82rem] font-bold text-warning-text">Weeks 11–12</span>
            <strong class="text-ink">Alerts, reports, and dashboards</strong>
          </div>
        </div>
      </template>
    </Card>
  </section>
</template>
```

- [ ] **Step 2: Run the full test suite to verify nothing broke**

Run: `cd frontend && npx vitest run`
Expected: PASS (42 tests — no test mounts `DashboardView` directly, confirms no build-wide regression).

- [ ] **Step 3: Typecheck and build**

Run: `cd frontend && npm run typecheck && npm run build`
Expected: both PASS.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/views/DashboardView.vue
git commit -m "refactor: restyle DashboardView.vue to match Figma card style"
```

---

## Self-Review Notes

**Spec coverage:** bordered-card style ✓, label-then-value stat card hierarchy ✓, pill color switched to `success-*` tokens ✓, roadmap label switched to `warning-text` ✓, two new tokens added for future use ✓. No new data, no movements table — matches the design doc's explicit scope boundary.

**Placeholder scan:** none — both tasks have complete, copy-pasteable code.
