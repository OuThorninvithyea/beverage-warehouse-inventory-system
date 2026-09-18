<script setup lang="ts">
import {
  AlertTriangle,
  ArrowLeftRight,
  ArrowRight,
  BarChart3,
  Boxes,
  Building2,
  Camera,
  Database,
  Download,
  History,
  PieChart,
  Tags,
  Upload,
} from 'lucide-vue-next'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import AdjustFormDialog from '@/components/AdjustFormDialog.vue'
import BarcodeScannerModal from '@/components/BarcodeScannerModal.vue'
import PickFormDialog from '@/components/PickFormDialog.vue'
import ReceiveFormDialog from '@/components/ReceiveFormDialog.vue'
import TransferFormDialog from '@/components/TransferFormDialog.vue'
import BaseChart from '@/components/BaseChart.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
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
  { label: 'Ambient Storage Zone', color: '#10b981', value: 72 },
  { label: 'Cold Storage Room', color: '#3b82f6', value: 48 },
  { label: 'Pallet Staging Rack', color: '#f59e0b', value: 85 },
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
    legend: { position: 'bottom' as const },
  },
}

function productName(productId: string): string {
  return catalogStore.products.find((p) => p.id === productId)?.name ?? 'Item'
}

// Timeline feed
const movementTimeline = computed(() => {
  return inventoryStore.movements.slice(0, 5).map((m) => ({
    id: m.id,
    status: `${m.movement_type.toUpperCase()}: ${productName(m.product_id)} (${m.quantity})`,
    date: formatDate(m.created_at),
    icon: m.movement_type === 'receive' ? Download : m.movement_type === 'pick' ? Upload : ArrowLeftRight,
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
    <div class="flex flex-wrap items-end justify-between gap-4">
      <div class="grid gap-2">
        <Badge variant="outline" class="w-fit border-emerald-500/30 bg-emerald-500/10 text-emerald-700">
          BWIMS Live Control Center
        </Badge>
        <h1 class="text-3xl font-semibold tracking-tight">Warehouse Analytics &amp; Operations</h1>
        <p class="text-sm text-muted-foreground">
          Welcome back, {{ auth.user?.full_name }}. Real-time stock velocity, capacity, and FEFO expiry insights.
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button variant="outline" @click="scannerVisible = true">
          <Camera class="size-4" />
          Scan Barcode
        </Button>
        <Button v-if="canReceiveOrPick" @click="receiveVisible = true">
          <Download class="size-4" />
          Receive Stock
        </Button>
        <Button v-if="canReceiveOrPick" variant="secondary" @click="pickVisible = true">
          <Upload class="size-4" />
          FEFO Pick
        </Button>
      </div>
    </div>

    <!-- Stat cards -->
    <div class="grid grid-cols-4 gap-4 max-[1024px]:grid-cols-2 max-[640px]:grid-cols-1">
      <Card class="border-l-4 border-l-brand-amber">
        <CardContent class="flex items-center justify-between">
          <div class="grid gap-1">
            <span class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Active Warehouses</span>
            <div class="text-3xl font-semibold">{{ warehouseStore.warehouses.length }}</div>
          </div>
          <div class="grid size-12 place-items-center rounded-xl bg-muted">
            <Building2 class="size-5" />
          </div>
        </CardContent>
      </Card>

      <Card class="border-l-4 border-l-sky-500">
        <CardContent class="flex items-center justify-between">
          <div class="grid gap-1">
            <span class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Catalog Products</span>
            <div class="text-3xl font-semibold">{{ catalogStore.products.length }}</div>
          </div>
          <div class="grid size-12 place-items-center rounded-xl bg-sky-500/10 text-sky-600">
            <Boxes class="size-5" />
          </div>
        </CardContent>
      </Card>

      <Card class="border-l-4 border-l-emerald-500">
        <CardContent class="flex items-center justify-between">
          <div class="grid gap-1">
            <span class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Stock Balances</span>
            <div class="text-3xl font-semibold">{{ inventoryStore.balances.length }}</div>
          </div>
          <div class="grid size-12 place-items-center rounded-xl bg-emerald-500/10 text-emerald-600">
            <Database class="size-5" />
          </div>
        </CardContent>
      </Card>

      <Card class="border-l-4 border-l-amber-500">
        <CardContent class="flex items-center justify-between">
          <div class="grid gap-1">
            <span class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Total Movements</span>
            <div class="text-3xl font-semibold">{{ inventoryStore.movements.length }}</div>
          </div>
          <div class="grid size-12 place-items-center rounded-xl bg-amber-500/10 text-amber-600">
            <History class="size-5" />
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- Occupancy -->
    <Card>
      <CardHeader>
        <CardTitle class="flex items-center gap-2 text-base">
          <PieChart class="size-4 text-brand-amber" />
          Warehouse Storage Zone Occupancy
        </CardTitle>
      </CardHeader>
      <CardContent class="grid gap-4">
        <div v-for="meter in occupancyMeters" :key="meter.label" class="grid gap-1.5">
          <div class="flex items-center justify-between text-sm">
            <span class="font-medium">{{ meter.label }}</span>
            <span class="text-muted-foreground">{{ meter.value }}%</span>
          </div>
          <div class="h-2 overflow-hidden rounded-full bg-muted">
            <div
              class="h-full rounded-full transition-all"
              :style="{ width: `${meter.value}%`, backgroundColor: meter.color }"
            />
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- Charts -->
    <div class="grid grid-cols-3 gap-6 max-[1024px]:grid-cols-1">
      <Card class="col-span-2 max-[1024px]:col-span-1">
        <CardHeader class="flex-row items-center justify-between space-y-0">
          <CardTitle class="flex items-center gap-2 text-base">
            <BarChart3 class="size-4 text-emerald-600" />
            Stock Movement Breakdown
          </CardTitle>
          <Badge variant="outline" class="border-emerald-500/30 bg-emerald-500/10 text-emerald-700">Real-time</Badge>
        </CardHeader>
        <CardContent>
          <div class="h-[260px]">
            <BaseChart type="bar" :data="movementChartData" :options="movementChartOptions" />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle class="flex items-center gap-2 text-base">
            <Tags class="size-4 text-sky-600" />
            Product Categories
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div class="h-[260px]">
            <BaseChart type="doughnut" :data="categoryChartData" :options="categoryChartOptions" />
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- Activity + watchlist -->
    <div class="grid grid-cols-3 gap-6 max-[1024px]:grid-cols-1">
      <Card>
        <CardHeader>
          <CardTitle class="flex items-center gap-2 text-base">
            <History class="size-4 text-brand-amber" />
            Live Activity Feed
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div v-if="movementTimeline.length === 0" class="py-6 text-center text-xs text-muted-foreground">
            No recent activity recorded.
          </div>
          <ol v-else class="relative grid gap-4 border-l pl-6">
            <li v-for="entry in movementTimeline" :key="entry.id" class="relative grid gap-0.5">
              <span
                class="absolute -left-[31px] grid size-5 place-items-center rounded-full text-white"
                :style="{ backgroundColor: entry.color }"
              >
                <component :is="entry.icon" class="size-3" />
              </span>
              <strong class="text-xs">{{ entry.status }}</strong>
              <small class="text-xs text-muted-foreground">{{ entry.date }}</small>
              <small v-if="entry.reference" class="font-mono text-xs text-muted-foreground">{{ entry.reference }}</small>
            </li>
          </ol>
        </CardContent>
      </Card>

      <Card class="col-span-2 max-[1024px]:col-span-1">
        <CardHeader class="flex-row items-center justify-between space-y-0">
          <CardTitle class="flex items-center gap-2 text-base">
            <AlertTriangle class="size-4 text-amber-600" />
            Low Stock &amp; FEFO Expiry Watchlist
          </CardTitle>
          <Button variant="ghost" size="sm" @click="router.push({ name: 'inventory' })">
            Inventory Control
            <ArrowRight class="size-4" />
          </Button>
        </CardHeader>
        <CardContent>
          <div class="overflow-hidden rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Product</TableHead>
                  <TableHead>Location</TableHead>
                  <TableHead>Available Qty</TableHead>
                  <TableHead>Expiration Date</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-if="lowStockWatchlist.length === 0">
                  <TableCell :colspan="4" class="py-8 text-center text-xs text-muted-foreground">
                    All inventory levels and expiration dates are in optimal shape!
                  </TableCell>
                </TableRow>

                <TableRow v-for="row in lowStockWatchlist" v-else :key="row.id">
                  <TableCell>
                    <div class="grid leading-tight">
                      <strong class="text-xs">{{ row.product_name }}</strong>
                      <small class="font-mono text-[11px] text-muted-foreground">SKU: {{ row.product_sku }}</small>
                    </div>
                  </TableCell>
                  <TableCell>
                    <span class="rounded border bg-muted px-1.5 py-0.5 font-mono text-[11px] font-semibold">
                      {{ row.location_code }}
                    </span>
                  </TableCell>
                  <TableCell>
                    <span
                      class="text-xs font-semibold"
                      :class="parseFloat(row.quantity) < 10 ? 'text-destructive' : ''"
                    >
                      {{ row.quantity }}
                    </span>
                  </TableCell>
                  <TableCell>
                    <span
                      v-if="row.expiration_date"
                      class="text-xs font-medium"
                      :class="isExpiringSoon(row.expiration_date) ? 'font-semibold text-amber-600' : 'text-muted-foreground'"
                    >
                      {{ row.expiration_date }}
                    </span>
                    <span v-else class="text-xs text-muted-foreground">&mdash;</span>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>
        </CardContent>
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

    <BarcodeScannerModal v-model:visible="scannerVisible" @select="onBarcodeScanned" />
  </div>
</template>
