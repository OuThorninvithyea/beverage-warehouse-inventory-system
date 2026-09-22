<script setup lang="ts">
import type { ChartData, ChartOptions } from 'chart.js'
import {
  ArrowLeftRight,
  ArrowRight,
  ArrowUpDown,
  Boxes,
  Download,
  PackageX,
  TriangleAlert,
  Upload,
  Warehouse,
} from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'

import BaseChart from '@/components/BaseChart.vue'
import PickFormDialog from '@/components/PickFormDialog.vue'
import ReceiveFormDialog from '@/components/ReceiveFormDialog.vue'
import TransferFormDialog from '@/components/TransferFormDialog.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import type { MovementType, ValuationRow } from '@/api/reports'
import { useAuthStore } from '@/stores/auth'
import { useCatalogStore } from '@/stores/catalog'
import { useInventoryStore } from '@/stores/inventory'
import { useReportsStore } from '@/stores/reports'
import { useWarehousesStore } from '@/stores/warehouses'

const auth = useAuthStore()
const catalogStore = useCatalogStore()
const inventoryStore = useInventoryStore()
const reportsStore = useReportsStore()
const warehouseStore = useWarehousesStore()

const receiveVisible = ref(false)
const pickVisible = ref(false)
const transferVisible = ref(false)

const windowDays = ref('30')
const warehouseId = ref('all')

const windowOptions = [
  { value: '7', label: 'Last 7 days' },
  { value: '30', label: 'Last 30 days' },
  { value: '90', label: 'Last 90 days' },
]

const canReceiveOrPick = computed(() =>
  ['admin', 'warehouse_manager', 'picker'].includes(auth.user?.role ?? ''),
)
const canTransfer = computed(() => ['admin', 'warehouse_manager'].includes(auth.user?.role ?? ''))
// Reports are an admin/manager view server-side, so pickers and viewers never
// see an empty report shell they are not allowed to fill.
const canReadReports = computed(() =>
  ['admin', 'warehouse_manager'].includes(auth.user?.role ?? ''),
)
const isAdmin = computed(() => auth.user?.role === 'admin')

const dashboard = computed(() => reportsStore.dashboard)

// Movement types keep a fixed order so each one keeps its colour when the
// window changes: colour follows the entity, never its rank.
const movementOrder: MovementType[] = ['receive', 'pick', 'transfer', 'adjust']
const movementLabels: Record<MovementType, string> = {
  receive: 'Receive',
  pick: 'Pick',
  transfer: 'Transfer',
  adjust: 'Adjust',
}

function formatNumber(value: string | number | undefined, fractionDigits = 0): string {
  const amount = Number(value ?? 0)
  if (Number.isNaN(amount)) return '—'
  return amount.toLocaleString('en-US', {
    minimumFractionDigits: fractionDigits,
    maximumFractionDigits: fractionDigits,
  })
}

const reservedShare = computed(() => {
  const total = Number(dashboard.value?.total_quantity ?? 0)
  if (total <= 0) return 0
  return Math.min(100, (Number(dashboard.value?.reserved_quantity ?? 0) / total) * 100)
})

// Movements by day is a stacked bar: the reader compares the day's total and
// its composition at the same time.
const movementChartData = computed<ChartData<'bar'>>(() => {
  const rows = reportsStore.movementSummary?.rows ?? []
  const days = [...new Set(rows.map((row) => row.day))].sort()
  return {
    labels: days.map((day) => day.slice(5)),
    datasets: movementOrder.map((type, index) => ({
      label: movementLabels[type],
      data: days.map(
        (day) => rows.find((row) => row.day === day && row.movement_type === type)?.movements ?? 0,
      ),
      backgroundColor: `var(--chart-${index + 1})`,
      borderRadius: 4,
      // A 2px surface gap stops adjacent stack segments bleeding together.
      borderWidth: 2,
      borderColor: 'var(--card)',
      borderSkipped: false,
    })),
  }
})

const movementChartOptions = computed<ChartOptions<'bar'>>(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: { mode: 'index', intersect: false },
  scales: {
    x: { stacked: true, grid: { display: false }, ticks: { maxRotation: 0 } },
    y: { stacked: true, beginAtZero: true, ticks: { precision: 0 } },
  },
  plugins: {
    // Four series share the screen, so the legend is mandatory: identity must
    // never rest on colour alone.
    legend: { position: 'bottom', labels: { boxWidth: 10, usePointStyle: true } },
  },
}))

// Top movers is magnitude, not identity, so one sequential hue is correct and
// a categorical palette would imply differences that are not there.
const velocityChartData = computed<ChartData<'bar'>>(() => {
  const rows = reportsStore.velocity?.rows ?? []
  return {
    labels: rows.map((row) => row.sku),
    datasets: [
      {
        label: 'Picked quantity',
        data: rows.map((row) => Number(row.picked_quantity)),
        backgroundColor: 'var(--chart-1)',
        borderRadius: 4,
      },
    ],
  }
})

const velocityChartOptions = computed<ChartOptions<'bar'>>(() => ({
  indexAxis: 'y',
  responsive: true,
  maintainAspectRatio: false,
  scales: {
    x: { beginAtZero: true, grid: { display: false } },
    y: { grid: { display: false } },
  },
  // One series: the card title names it, so a legend box would be noise.
  plugins: { legend: { display: false } },
}))

// Every valuation row carries meaning, so this stays a table rather than
// becoming more chart colours.
type SortKey = 'sku' | 'remaining_quantity' | 'total_value'
const sortKey = ref<SortKey>('total_value')
const sortDescending = ref(true)

const sortedValuation = computed<ValuationRow[]>(() => {
  const rows = [...(reportsStore.valuation?.rows ?? [])]
  rows.sort((left, right) => {
    if (sortKey.value === 'sku') {
      return sortDescending.value
        ? right.sku.localeCompare(left.sku)
        : left.sku.localeCompare(right.sku)
    }
    const difference = Number(left[sortKey.value]) - Number(right[sortKey.value])
    return sortDescending.value ? -difference : difference
  })
  return rows
})

function toggleSort(key: SortKey) {
  if (sortKey.value === key) {
    sortDescending.value = !sortDescending.value
    return
  }
  sortKey.value = key
  sortDescending.value = true
}

// Share of the largest row, for the in-cell bar that makes the table read as a
// ranking without adding a second chart.
const largestValue = computed(() =>
  Math.max(1, ...sortedValuation.value.map((row) => Number(row.total_value))),
)

async function load() {
  if (!canReadReports.value) return
  await reportsStore.fetchAll({
    days: Number(windowDays.value),
    warehouse_id: warehouseId.value === 'all' ? undefined : warehouseId.value,
  })
}

async function refreshAfterMovement() {
  await Promise.all([inventoryStore.fetchBalances(), inventoryStore.fetchMovements(), load()])
}

onMounted(async () => {
  await Promise.all([
    warehouseStore.fetchWarehouses(),
    catalogStore.fetchProducts(),
    warehouseStore.fetchAllLocations(),
    load(),
  ])
})

watch([windowDays, warehouseId], load)
</script>

<template>
  <div class="grid gap-6">
    <div class="flex flex-wrap items-end justify-between gap-4">
      <div class="grid gap-1">
        <p class="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
          Operations overview
        </p>
        <h1 class="text-2xl font-semibold tracking-tight">
          Good morning, {{ auth.user?.full_name }}
        </h1>
        <p class="text-sm text-muted-foreground">
          Keep stock moving safely across
          {{ dashboard?.active_warehouses ?? warehouseStore.warehouses.length }} warehouse(s).
        </p>
      </div>

      <div v-if="canReadReports" class="flex flex-wrap items-center gap-2">
        <Select v-model="windowDays">
          <SelectTrigger class="w-[150px]"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem v-for="option in windowOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </SelectItem>
          </SelectContent>
        </Select>
        <Select v-if="isAdmin" v-model="warehouseId">
          <SelectTrigger class="w-[200px]"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All warehouses</SelectItem>
            <SelectItem
              v-for="warehouse in warehouseStore.warehouses"
              :key="warehouse.id"
              :value="warehouse.id"
            >
              {{ warehouse.code }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>

    <p
      v-if="reportsStore.error"
      class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
    >
      {{ reportsStore.error }}
    </p>

    <!-- Exceptions first: a warehouse dashboard answers "what needs attention
         today" before "how much do we own". The status colour ships with an
         icon and a label, never colour alone. -->
    <RouterLink
      v-if="
        canReadReports &&
        dashboard &&
        (dashboard.expired_lots > 0 || dashboard.expiring_soon_lots > 0)
      "
      to="/alerts"
      class="block"
    >
      <Card class="border-destructive/40 bg-destructive/5 transition-colors hover:bg-destructive/10">
        <CardContent class="flex flex-wrap items-center gap-x-6 gap-y-2">
          <span class="flex items-center gap-2 font-medium text-destructive">
            <TriangleAlert class="size-5" /> Needs attention
          </span>
          <span v-if="dashboard.expired_lots > 0" class="flex items-center gap-2 text-sm">
            <PackageX class="size-4 text-destructive" />
            <strong>{{ dashboard.expired_lots }}</strong> expired
            {{ dashboard.expired_lots === 1 ? 'lot' : 'lots' }} still on hand
          </span>
          <span v-if="dashboard.expiring_soon_lots > 0" class="text-sm">
            <strong>{{ dashboard.expiring_soon_lots }}</strong> expiring within 30 days
          </span>
          <span class="ml-auto flex items-center gap-1 text-sm text-muted-foreground">
            Review alerts <ArrowRight class="size-4" />
          </span>
        </CardContent>
      </Card>
    </RouterLink>

    <template v-if="canReadReports">
      <div v-if="reportsStore.loading && !dashboard" class="grid gap-4 md:grid-cols-4">
        <Skeleton v-for="tile in 4" :key="tile" class="h-28 w-full" />
      </div>

      <!-- A hero figure plus a KPI row: a handful of headline numbers is a
           stat row, never a bar chart. -->
      <div v-else-if="dashboard" class="grid gap-4 lg:grid-cols-4">
        <Card class="lg:col-span-2">
          <CardContent class="grid gap-2">
            <span class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              Stock value (FIFO cost layers)
            </span>
            <strong class="text-5xl font-semibold tabular-nums tracking-tight">
              {{ formatNumber(dashboard.stock_value, 2) }}
            </strong>
            <span class="text-xs text-muted-foreground">
              {{ formatNumber(dashboard.total_quantity) }} units on hand across
              {{ dashboard.lots_on_hand }} lots
            </span>
            <!-- A single ratio against a limit is a meter, not a two-slice pie. -->
            <div class="mt-2 grid gap-1">
              <div class="flex justify-between text-xs text-muted-foreground">
                <span>{{ formatNumber(dashboard.reserved_quantity) }} reserved</span>
                <span>{{ formatNumber(dashboard.available_quantity) }} available</span>
              </div>
              <div class="h-2 overflow-hidden rounded-full bg-muted">
                <div
                  class="h-full rounded-full bg-chart-1"
                  :style="{ width: `${reservedShare}%` }"
                />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent class="flex items-center justify-between">
            <div class="grid gap-1">
              <span class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Active products
              </span>
              <strong class="text-3xl font-semibold tabular-nums">
                {{ dashboard.active_products }}
              </strong>
              <RouterLink
                to="/products"
                class="text-xs text-muted-foreground underline-offset-2 hover:underline"
              >
                View catalog
              </RouterLink>
            </div>
            <div class="grid size-11 place-items-center rounded-xl bg-muted">
              <Boxes class="size-5" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent class="flex items-center justify-between">
            <div class="grid gap-1">
              <span class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Locations in use
              </span>
              <strong class="text-3xl font-semibold tabular-nums">
                {{ dashboard.active_locations }}
              </strong>
              <RouterLink
                to="/warehouses"
                class="text-xs text-muted-foreground underline-offset-2 hover:underline"
              >
                View warehouses
              </RouterLink>
            </div>
            <div class="grid size-11 place-items-center rounded-xl bg-muted">
              <Warehouse class="size-5" />
            </div>
          </CardContent>
        </Card>
      </div>

      <div class="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle class="text-base">Movements by day</CardTitle>
            <CardDescription>
              Receives, picks, transfers and adjustments over the selected window.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Skeleton v-if="reportsStore.loading" class="h-[280px] w-full" />
            <p
              v-else-if="(reportsStore.movementSummary?.rows.length ?? 0) === 0"
              class="py-16 text-center text-sm text-muted-foreground"
            >
              No movements in this window.
            </p>
            <div v-else class="h-[280px]">
              <BaseChart type="bar" :data="movementChartData" :options="movementChartOptions" />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle class="text-base">Top movers by picked quantity</CardTitle>
            <CardDescription>
              Outbound throughput only; a transfer relocates stock without consuming it.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Skeleton v-if="reportsStore.loading" class="h-[280px] w-full" />
            <p
              v-else-if="(reportsStore.velocity?.rows.length ?? 0) === 0"
              class="py-16 text-center text-sm text-muted-foreground"
            >
              Nothing was picked in this window.
            </p>
            <div v-else class="h-[280px]">
              <BaseChart type="bar" :data="velocityChartData" :options="velocityChartOptions" />
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader class="flex-row items-center justify-between space-y-0">
          <div>
            <CardTitle class="text-base">Inventory valuation</CardTitle>
            <CardDescription>
              Remaining FIFO cost layers per product. Total
              {{ formatNumber(reportsStore.valuation?.total_value, 2) }} across the current scope.
            </CardDescription>
          </div>
          <Button variant="ghost" size="sm" as-child>
            <RouterLink to="/inventory">View inventory <ArrowRight class="size-4" /></RouterLink>
          </Button>
        </CardHeader>
        <CardContent>
          <Skeleton v-if="reportsStore.loading" class="h-40 w-full" />
          <div v-else class="overflow-x-auto rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>
                    <button class="inline-flex items-center gap-1" @click="toggleSort('sku')">
                      Product <ArrowUpDown class="size-3" />
                    </button>
                  </TableHead>
                  <TableHead>Warehouse</TableHead>
                  <TableHead class="text-right">
                    <button
                      class="inline-flex items-center gap-1"
                      @click="toggleSort('remaining_quantity')"
                    >
                      On hand <ArrowUpDown class="size-3" />
                    </button>
                  </TableHead>
                  <TableHead class="text-right">Avg unit cost</TableHead>
                  <TableHead class="text-right">
                    <button
                      class="inline-flex items-center gap-1"
                      @click="toggleSort('total_value')"
                    >
                      Value <ArrowUpDown class="size-3" />
                    </button>
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-if="sortedValuation.length === 0">
                  <TableCell colspan="5" class="py-10 text-center text-sm text-muted-foreground">
                    No valued stock in this scope.
                  </TableCell>
                </TableRow>
                <TableRow
                  v-for="row in sortedValuation"
                  v-else
                  :key="`${row.warehouse_id}-${row.product_id}`"
                >
                  <TableCell>
                    <div class="grid">
                      <strong class="text-sm">{{ row.product_name }}</strong>
                      <small class="font-mono text-xs text-muted-foreground">{{ row.sku }}</small>
                    </div>
                  </TableCell>
                  <TableCell><Badge variant="outline">{{ row.warehouse_code }}</Badge></TableCell>
                  <TableCell class="text-right tabular-nums">
                    {{ formatNumber(row.remaining_quantity) }}
                  </TableCell>
                  <TableCell class="text-right tabular-nums text-muted-foreground">
                    {{ formatNumber(row.average_unit_cost, 2) }}
                  </TableCell>
                  <TableCell class="text-right">
                    <div class="grid justify-items-end gap-1">
                      <span class="font-medium tabular-nums">
                        {{ formatNumber(row.total_value, 2) }}
                      </span>
                      <div class="h-1 w-24 overflow-hidden rounded-full bg-muted">
                        <div
                          class="h-full rounded-full bg-chart-1"
                          :style="{ width: `${(Number(row.total_value) / largestValue) * 100}%` }"
                        />
                      </div>
                    </div>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </template>

    <Card>
      <CardHeader>
        <CardTitle class="text-base">Inventory operations</CardTitle>
        <CardDescription>Start the next warehouse task without leaving the dashboard.</CardDescription>
      </CardHeader>
      <CardContent class="grid gap-3 sm:grid-cols-3">
        <Button
          v-if="canReceiveOrPick"
          class="h-auto justify-start gap-3 p-4"
          @click="receiveVisible = true"
        >
          <Download class="size-5" />
          <span class="grid justify-items-start gap-0.5">
            <strong>Receive Stock</strong>
            <small class="font-normal opacity-80">Record inbound stock</small>
          </span>
        </Button>
        <Button
          v-if="canReceiveOrPick"
          variant="secondary"
          class="h-auto justify-start gap-3 p-4"
          @click="pickVisible = true"
        >
          <Upload class="size-5" />
          <span class="grid justify-items-start gap-0.5">
            <strong>Pick Order</strong>
            <small class="font-normal opacity-80">Pick by FEFO</small>
          </span>
        </Button>
        <Button
          v-if="canTransfer"
          variant="outline"
          class="h-auto justify-start gap-3 p-4"
          @click="transferVisible = true"
        >
          <ArrowLeftRight class="size-5" />
          <span class="grid justify-items-start gap-0.5">
            <strong>Transfer Stock</strong>
            <small class="font-normal opacity-80">Move between locations</small>
          </span>
        </Button>
        <p v-if="!canReceiveOrPick && !canTransfer" class="text-sm text-muted-foreground">
          Viewer access is read-only. Transactional actions are disabled.
        </p>
      </CardContent>
    </Card>

    <ReceiveFormDialog
      v-model:visible="receiveVisible"
      :locations="warehouseStore.locations"
      :products="catalogStore.products"
      @submitted="refreshAfterMovement"
    />
    <PickFormDialog
      v-model:visible="pickVisible"
      :locations="warehouseStore.locations"
      :products="catalogStore.products"
      @submitted="refreshAfterMovement"
    />
    <TransferFormDialog
      v-model:visible="transferVisible"
      :locations="warehouseStore.locations"
      :products="catalogStore.products"
      @submitted="refreshAfterMovement"
    />
  </div>
</template>
