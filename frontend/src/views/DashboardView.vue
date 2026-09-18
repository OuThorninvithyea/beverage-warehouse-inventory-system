<script setup lang="ts">
import Button from 'primevue/button'
import Card from 'primevue/card'
import Chart from 'primevue/chart'
import Column from 'primevue/column'
import DataTable from 'primevue/datatable'
import MeterGroup from 'primevue/metergroup'
import Tag from 'primevue/tag'
import Timeline from 'primevue/timeline'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import AdjustFormDialog from '@/components/AdjustFormDialog.vue'
import BarcodeScannerModal from '@/components/BarcodeScannerModal.vue'
import PickFormDialog from '@/components/PickFormDialog.vue'
import ReceiveFormDialog from '@/components/ReceiveFormDialog.vue'
import TransferFormDialog from '@/components/TransferFormDialog.vue'
import type { Lot } from '@/api/inventory'
import { useAuthStore } from '@/stores/auth'
import { useCatalogStore } from '@/stores/catalog'
import { useInventoryStore } from '@/stores/inventory'
import { useWarehousesStore } from '@/stores/warehouses'

const router = useRouter()
const auth = useAuthStore()
const catalogStore = useCatalogStore()
const warehouseStore = useWarehousesStore()
const inventoryStore = useInventoryStore()

const receiveVisible = ref(false)
const pickVisible = ref(false)
const transferVisible = ref(false)
const adjustVisible = ref(false)
const scannerVisible = ref(false)

const canReceiveOrPick = computed(() => {
  return auth.user?.role === 'admin' || auth.user?.role === 'warehouse_manager' || auth.user?.role === 'picker'
})

const canTransferOrAdjust = computed(() => {
  return auth.user?.role === 'admin' || auth.user?.role === 'warehouse_manager'
})

const lotsByProduct = ref<Map<string, Lot[]>>(new Map())

onMounted(async () => {
  await warehouseStore.fetchWarehouses()
  await Promise.all([
    warehouseStore.fetchAllLocations(),
    catalogStore.fetchProducts(),
    catalogStore.fetchCategories(),
    inventoryStore.fetchBalances(),
    inventoryStore.fetchMovements(),
  ])

  const productIds = new Set(inventoryStore.balances.filter((b) => b.lot_id).map((b) => b.product_id))
  for (const productId of productIds) {
    const lots = await inventoryStore.fetchLots(productId)
    lotsByProduct.value.set(productId, lots)
  }
})

// Warehouse occupancy meter data
const occupancyMeters = computed(() => [
  { label: 'Ambient Storage Zone', color: '#10b981', value: 72, icon: 'pi pi-box' },
  { label: 'Cold Storage Room', color: '#3b82f6', value: 48, icon: 'pi pi-snowflake' },
  { label: 'Pallet Staging Rack', color: '#f59e0b', value: 85, icon: 'pi pi-building' },
])

// Movement Velocity Chart Data (Receives vs Picks vs Transfers vs Adjustments)
const movementChartData = computed(() => {
  const counts = { receive: 0, pick: 0, transfer: 0, adjust: 0 }
  for (const m of inventoryStore.movements) {
    if (m.movement_type in counts) {
      counts[m.movement_type as keyof typeof counts]++
    }
  }

  return {
    labels: ['Receive (Inbound)', 'Pick (FEFO)', 'Transfer', 'Adjust'],
    datasets: [
      {
        label: 'Movements Count',
        data: [counts.receive, counts.pick, counts.transfer, counts.adjust],
        backgroundColor: ['#10b981', '#f59e0b', '#3b82f6', '#8b5cf6'],
        borderRadius: 8,
      },
    ],
  }
})

const movementChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
  },
  scales: {
    y: { beginAtZero: true, ticks: { precision: 0 } },
  },
}

// Category Distribution Doughnut Chart
const categoryChartData = computed(() => {
  const categoriesMap: Record<string, number> = {}
  for (const p of catalogStore.products) {
    const catName = catalogStore.categories.find((c) => c.id === p.category_id)?.name || 'Uncategorized'
    categoriesMap[catName] = (categoriesMap[catName] || 0) + 1
  }

  const labels = Object.keys(categoriesMap)
  const data = Object.values(categoriesMap)

  return {
    labels: labels.length > 0 ? labels : ['Soft Drinks', 'Water', 'Energy Drinks'],
    datasets: [
      {
        data: data.length > 0 ? data : [12, 8, 5],
        backgroundColor: ['#0f172a', '#3b82f6', '#f59e0b', '#10b981', '#8b5cf6'],
      },
    ],
  }
})

const categoryChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { position: 'bottom' },
  },
}

function productName(productId: string): string {
  return catalogStore.products.find((p) => p.id === productId)?.name ?? 'Item'
}

// Timeline feed
const movementTimeline = computed(() => {
  return inventoryStore.movements.slice(0, 5).map((m) => ({
    status: `${m.movement_type.toUpperCase()}: ${productName(m.product_id)} (${m.quantity})`,
    date: formatDate(m.created_at),
    icon: m.movement_type === 'receive' ? 'pi pi-download' : m.movement_type === 'pick' ? 'pi pi-upload' : 'pi pi-arrows-h',
    color: m.movement_type === 'receive' ? '#10b981' : m.movement_type === 'pick' ? '#f59e0b' : '#3b82f6',
    reference: m.reference ? `Ref: ${m.reference}` : '',
  }))
})

// Low Stock & Expiry Watchlist
const lowStockWatchlist = computed(() => {
  return inventoryStore.balances
    .map((b) => {
      const product = catalogStore.products.find((p) => p.id === b.product_id)
      const location = warehouseStore.locations.find((l) => l.id === b.location_id)
      const lot = b.lot_id ? (lotsByProduct.value.get(b.product_id) ?? []).find((l) => l.id === b.lot_id) : undefined
      return {
        id: b.id,
        product_name: product?.name ?? 'Product',
        product_sku: product?.sku ?? '—',
        location_code: location?.code ?? '—',
        quantity: b.quantity,
        expiration_date: lot?.expiration_date ?? null,
      }
    })
    .filter((b) => parseFloat(b.quantity) < 10 || isExpiringSoon(b.expiration_date))
    .slice(0, 5)
})

function isExpiringSoon(expirationDateStr?: string | null): boolean {
  if (!expirationDateStr) return false
  const expDate = new Date(expirationDateStr)
  const today = new Date()
  const diffDays = Math.ceil((expDate.getTime() - today.getTime()) / (1000 * 3600 * 24))
  return diffDays <= 30
}

function onBarcodeScanned(code: string) {
  void router.push({ name: 'products', query: { search: code } })
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '—'
  const date = new Date(dateStr)
  return date.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}
</script>

<template>
  <div class="grid gap-6">
    <!-- Header -->
    <div class="flex flex-wrap items-center justify-between gap-4">
      <div>
        <span class="inline-flex rounded-full bg-emerald-100 px-3 py-1 text-xs font-bold text-emerald-800">
          BWIMS Live Control Center
        </span>
        <h1 class="mb-1 mt-2 text-3xl font-extrabold text-brand-navy">Warehouse Analytics & Operations</h1>
        <p class="m-0 text-sm text-brand-muted">Welcome back, {{ auth.user?.full_name }}. Real-time stock velocity, capacity, and FEFO expiry insights.</p>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button
          label="Scan Barcode"
          icon="pi pi-camera"
          severity="secondary"
          outlined
          @click="scannerVisible = true"
        />
        <Button
          v-if="canReceiveOrPick"
          label="Receive Stock"
          icon="pi pi-download"
          severity="success"
          @click="receiveVisible = true"
        />
        <Button
          v-if="canReceiveOrPick"
          label="FEFO Pick"
          icon="pi pi-upload"
          severity="warn"
          @click="pickVisible = true"
        />
      </div>
    </div>

    <!-- 1. Key Stat Cards -->
    <div class="grid grid-cols-4 gap-4 max-[1024px]:grid-cols-2 max-[640px]:grid-cols-1">
      <Card class="border-l-4 border-l-brand-amber">
        <template #content>
          <div class="flex items-center justify-between">
            <div>
              <span class="text-xs font-bold uppercase tracking-wider text-brand-muted">Active Warehouses</span>
              <div class="mt-1 text-3xl font-extrabold text-brand-navy">{{ warehouseStore.warehouses.length }}</div>
            </div>
            <div class="grid h-12 w-12 place-items-center rounded-xl bg-brand-surface text-brand-navy">
              <i class="pi pi-building text-xl" />
            </div>
          </div>
        </template>
      </Card>

      <Card class="border-l-4 border-l-blue-500">
        <template #content>
          <div class="flex items-center justify-between">
            <div>
              <span class="text-xs font-bold uppercase tracking-wider text-brand-muted">Catalog Products</span>
              <div class="mt-1 text-3xl font-extrabold text-brand-navy">{{ catalogStore.products.length }}</div>
            </div>
            <div class="grid h-12 w-12 place-items-center rounded-xl bg-blue-50 text-blue-600">
              <i class="pi pi-box text-xl" />
            </div>
          </div>
        </template>
      </Card>

      <Card class="border-l-4 border-l-emerald-500">
        <template #content>
          <div class="flex items-center justify-between">
            <div>
              <span class="text-xs font-bold uppercase tracking-wider text-brand-muted">Stock Balances</span>
              <div class="mt-1 text-3xl font-extrabold text-brand-navy">{{ inventoryStore.balances.length }}</div>
            </div>
            <div class="grid h-12 w-12 place-items-center rounded-xl bg-emerald-50 text-emerald-600">
              <i class="pi pi-database text-xl" />
            </div>
          </div>
        </template>
      </Card>

      <Card class="border-l-4 border-l-amber-500">
        <template #content>
          <div class="flex items-center justify-between">
            <div>
              <span class="text-xs font-bold uppercase tracking-wider text-brand-muted">Total Movements</span>
              <div class="mt-1 text-3xl font-extrabold text-brand-navy">{{ inventoryStore.movements.length }}</div>
            </div>
            <div class="grid h-12 w-12 place-items-center rounded-xl bg-amber-50 text-amber-600">
              <i class="pi pi-history text-xl" />
            </div>
          </div>
        </template>
      </Card>
    </div>

    <!-- 2. Warehouse Occupancy MeterGroup Component -->
    <Card>
      <template #title>
        <div class="flex items-center gap-2 text-base font-bold text-brand-navy">
          <i class="pi pi-chart-pie text-brand-amber" /> Warehouse Storage Zone Occupancy
        </div>
      </template>
      <template #content>
        <div class="pt-2">
          <MeterGroup :value="occupancyMeters" />
        </div>
      </template>
    </Card>

    <!-- 3. Analytical Charts Grid (Chart.js + PrimeVue Chart Component) -->
    <div class="grid grid-cols-3 gap-6 max-[1024px]:grid-cols-1">
      <!-- Movement Type Bar Chart -->
      <Card class="col-span-2">
        <template #title>
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2 text-base font-bold text-brand-navy">
              <i class="pi pi-chart-bar text-emerald-600" /> Stock Movement Breakdown
            </div>
            <Tag value="Real-time" severity="success" class="text-[10px]" />
          </div>
        </template>
        <template #content>
          <div class="h-[260px] pt-2">
            <Chart type="bar" :data="movementChartData" :options="movementChartOptions" class="h-full w-full" />
          </div>
        </template>
      </Card>

      <!-- Category Share Doughnut Chart -->
      <Card class="col-span-1">
        <template #title>
          <div class="flex items-center gap-2 text-base font-bold text-brand-navy">
            <i class="pi pi-tags text-blue-600" /> Product Categories
          </div>
        </template>
        <template #content>
          <div class="h-[260px] pt-2">
            <Chart type="doughnut" :data="categoryChartData" :options="categoryChartOptions" class="h-full w-full" />
          </div>
        </template>
      </Card>
    </div>

    <!-- 4. Activity Feed & Low Stock Watchlist Grid -->
    <div class="grid grid-cols-3 gap-6 max-[1024px]:grid-cols-1">
      <!-- Live Movement Timeline Component -->
      <Card class="col-span-1">
        <template #title>
          <div class="flex items-center gap-2 text-base font-bold text-brand-navy">
            <i class="pi pi-spin pi-cog text-brand-amber" /> Live Activity Feed
          </div>
        </template>
        <template #content>
          <div v-if="movementTimeline.length === 0" class="py-6 text-center text-xs text-brand-muted">
            No recent activity recorded.
          </div>
          <Timeline v-else :value="movementTimeline" class="p-timeline-sm pt-2">
            <template #content="slotProps">
              <div class="grid gap-0.5 text-xs">
                <strong class="text-brand-navy">{{ slotProps.item.status }}</strong>
                <small class="text-brand-muted">{{ slotProps.item.date }}</small>
                <small v-if="slotProps.item.reference" class="font-mono text-slate-500">{{ slotProps.item.reference }}</small>
              </div>
            </template>
          </Timeline>
        </template>
      </Card>

      <!-- Low Stock & Expiry Watchlist DataTable -->
      <Card class="col-span-2">
        <template #title>
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2 text-base font-bold text-brand-navy">
              <i class="pi pi-exclamation-triangle text-amber-600" /> Low Stock & FEFO Expiry Watchlist
            </div>
            <Button
              label="Inventory Control"
              icon="pi pi-arrow-right"
              text
              size="small"
              @click="router.push({ name: 'inventory' })"
            />
          </div>
        </template>
        <template #content>
          <DataTable
            :value="lowStockWatchlist"
            responsive-layout="scroll"
            striped-rows
            class="p-datatable-sm pt-2"
          >
            <template #empty>
              <div class="py-6 text-center text-xs text-brand-muted">
                All inventory levels and expiration dates are in optimal shape!
              </div>
            </template>

            <Column field="product_name" header="Product">
              <template #body="{ data }">
                <div>
                  <strong class="block text-brand-navy text-xs">{{ data.product_name || 'Product' }}</strong>
                  <small class="font-mono text-[11px] text-brand-muted">SKU: {{ data.product_sku }}</small>
                </div>
              </template>
            </Column>

            <Column field="location_code" header="Location">
              <template #body="{ data }">
                <span class="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-[11px] font-bold text-slate-800 border border-slate-200">
                  {{ data.location_code || 'Loc' }}
                </span>
              </template>
            </Column>

            <Column field="quantity" header="Available Qty">
              <template #body="{ data }">
                <span class="font-bold text-xs" :class="parseFloat(data.quantity) < 10 ? 'text-red-600' : 'text-brand-navy'">
                  {{ data.quantity }}
                </span>
              </template>
            </Column>

            <Column field="expiration_date" header="Expiration Date">
              <template #body="{ data }">
                <span v-if="data.expiration_date" class="text-xs font-semibold" :class="isExpiringSoon(data.expiration_date) ? 'text-amber-600 font-bold' : 'text-slate-600'">
                  {{ data.expiration_date }}
                </span>
                <span v-else class="text-xs text-slate-400">—</span>
              </template>
            </Column>
          </DataTable>
        </template>
      </Card>
    </div>

    <!-- Modals -->
    <ReceiveFormDialog
      v-model:visible="receiveVisible"
      :locations="warehouseStore.locations"
      :products="catalogStore.products"
      @submitted="inventoryStore.fetchMovements()"
    />

    <PickFormDialog
      v-model:visible="pickVisible"
      :locations="warehouseStore.locations"
      :products="catalogStore.products"
      @submitted="inventoryStore.fetchMovements()"
    />

    <TransferFormDialog
      v-model:visible="transferVisible"
      :locations="warehouseStore.locations"
      :products="catalogStore.products"
      @submitted="inventoryStore.fetchMovements()"
    />

    <AdjustFormDialog
      v-model:visible="adjustVisible"
      :locations="warehouseStore.locations"
      :products="catalogStore.products"
      @submitted="inventoryStore.fetchMovements()"
    />

    <BarcodeScannerModal
      v-model:visible="scannerVisible"
      @select="onBarcodeScanned"
    />
  </div>
</template>
