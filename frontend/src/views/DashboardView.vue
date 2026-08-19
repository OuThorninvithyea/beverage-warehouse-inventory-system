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
