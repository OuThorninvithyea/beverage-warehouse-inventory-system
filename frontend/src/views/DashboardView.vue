<script setup lang="ts">
import Card from 'primevue/card'

const foundationItems = [
  { value: '3', label: 'Warehouses planned' },
  { value: '4', label: 'Access roles defined' },
  { value: '12', label: 'Core database tables' },
]
</script>

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
