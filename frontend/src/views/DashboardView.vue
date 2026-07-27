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
    <div class="page-heading">
      <div>
        <span class="status-pill">Foundation ready</span>
        <h1>Inventory overview</h1>
        <p>The shell is ready for live inventory, movement, and expiry data.</p>
      </div>
    </div>

    <div class="metric-grid">
      <Card v-for="item in foundationItems" :key="item.label" class="metric-card">
        <template #content>
          <strong>{{ item.value }}</strong>
          <span>{{ item.label }}</span>
        </template>
      </Card>
    </div>

    <Card class="roadmap-card">
      <template #title>Implementation path</template>
      <template #content>
        <div class="roadmap">
          <div>
            <span>Weeks 7–8</span>
            <strong>Authentication and master data</strong>
          </div>
          <div>
            <span>Weeks 9–10</span>
            <strong>Inventory and stock movements</strong>
          </div>
          <div>
            <span>Weeks 11–12</span>
            <strong>Alerts, reports, and dashboards</strong>
          </div>
        </div>
      </template>
    </Card>
  </section>
</template>
