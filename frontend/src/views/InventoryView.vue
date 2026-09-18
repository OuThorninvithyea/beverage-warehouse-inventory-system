<script setup lang="ts">
import {
  AlertTriangle,
  ArrowLeftRight,
  Download,
  FileSpreadsheet,
  MapPin,
  SlidersHorizontal,
  Upload,
} from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'

import AdjustFormDialog from '@/components/AdjustFormDialog.vue'
import PickFormDialog from '@/components/PickFormDialog.vue'
import ReceiveFormDialog from '@/components/ReceiveFormDialog.vue'
import TransferFormDialog from '@/components/TransferFormDialog.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
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
import { exportToCSV } from '@/lib/export'
import { useAuthStore } from '@/stores/auth'
import { useCatalogStore } from '@/stores/catalog'
import { useInventoryStore } from '@/stores/inventory'
import { useWarehousesStore } from '@/stores/warehouses'

const auth = useAuthStore()
const inventoryStore = useInventoryStore()
const catalogStore = useCatalogStore()
const warehouseStore = useWarehousesStore()

const selectedWarehouseId = ref<string>()
const locationFilter = ref<string>('all')
const productFilter = ref<string>('all')

const selectedLocationId = computed(() =>
  locationFilter.value === 'all' ? undefined : locationFilter.value,
)
const selectedProductId = computed(() =>
  productFilter.value === 'all' ? undefined : productFilter.value,
)

const receiveVisible = ref(false)
const pickVisible = ref(false)
const transferVisible = ref(false)
const adjustVisible = ref(false)

interface EnrichedBalance {
  id: string
  location_id: string
  product_id: string
  lot_id: string | null
  quantity: string
  reserved_quantity: string
  available_quantity: string
  location_code: string
  product_name: string
  product_sku: string
  lot_number: string | null
  expiration_date: string | null
}

const lotsByProduct = ref<Map<string, import('@/api/inventory').Lot[]>>(new Map())

const enrichedBalances = computed<EnrichedBalance[]>(() => {
  return inventoryStore.balances.map((b) => {
    const location = warehouseStore.locations.find((l) => l.id === b.location_id)
    const product = catalogStore.products.find((p) => p.id === b.product_id)
    const lot = b.lot_id
      ? (lotsByProduct.value.get(b.product_id) ?? []).find((l) => l.id === b.lot_id)
      : undefined
    return {
      id: b.id,
      location_id: b.location_id,
      product_id: b.product_id,
      lot_id: b.lot_id,
      quantity: b.quantity,
      reserved_quantity: b.reserved_quantity,
      available_quantity: b.available_quantity,
      location_code: location?.code ?? b.location_id.substring(0, 8),
      product_name: product?.name ?? 'Product',
      product_sku: product?.sku ?? '—',
      lot_number: lot?.lot_number ?? null,
      expiration_date: lot?.expiration_date ?? null,
    }
  })
})

async function loadLotsForVisibleBalances() {
  const productIds = new Set(inventoryStore.balances.filter((b) => b.lot_id).map((b) => b.product_id))
  for (const productId of productIds) {
    if (!lotsByProduct.value.has(productId)) {
      const lots = await inventoryStore.fetchLots(productId)
      lotsByProduct.value.set(productId, lots)
    }
  }
}

function exportInventoryCSV() {
  exportToCSV('inventory_stock_balances', enrichedBalances.value, [
    { key: 'location_code', label: 'Location' },
    { key: 'product_sku', label: 'SKU' },
    { key: 'product_name', label: 'Product Name' },
    { key: 'lot_number', label: 'Lot Number' },
    { key: 'expiration_date', label: 'Expiration Date' },
    { key: 'quantity', label: 'Available Quantity' },
    { key: 'reserved_quantity', label: 'Reserved Quantity' },
  ])
}

const canReceiveOrPick = computed(() => {
  return auth.user?.role === 'admin' || auth.user?.role === 'warehouse_manager' || auth.user?.role === 'picker'
})

const canTransferOrAdjust = computed(() => {
  return auth.user?.role === 'admin' || auth.user?.role === 'warehouse_manager'
})

const currentWarehouseLocations = computed(() => {
  if (!selectedWarehouseId.value) return warehouseStore.locations
  return warehouseStore.locations.filter((l) => l.warehouse_id === selectedWarehouseId.value)
})

onMounted(async () => {
  await Promise.all([
    warehouseStore.fetchWarehouses(),
    catalogStore.fetchProducts(),
  ])

  if (warehouseStore.warehouses.length > 0) {
    selectedWarehouseId.value = warehouseStore.warehouses[0].id
    await warehouseStore.fetchLocations(selectedWarehouseId.value)
  }

  await fetchBalances()
})

watch(selectedWarehouseId, async (newWhId) => {
  locationFilter.value = 'all'
  if (newWhId) {
    await warehouseStore.fetchLocations(newWhId)
  }
  await fetchBalances()
})

watch([selectedLocationId, selectedProductId], () => {
  void fetchBalances()
})

async function fetchBalances() {
  await inventoryStore.fetchBalances({
    warehouse_id: selectedWarehouseId.value,
    location_id: selectedLocationId.value,
    product_id: selectedProductId.value,
  })
  await loadLotsForVisibleBalances()
}

function isExpiringSoon(expirationDateStr?: string | null): boolean {
  if (!expirationDateStr) return false
  const expDate = new Date(expirationDateStr)
  const today = new Date()
  const diffDays = Math.ceil((expDate.getTime() - today.getTime()) / (1000 * 3600 * 24))
  return diffDays <= 30
}

function isLowStock(qtyStr: string): boolean {
  const q = parseFloat(qtyStr)
  return q < 10
}
</script>

<template>
  <div class="grid gap-6">
    <div class="flex flex-wrap items-end justify-between gap-4">
      <div class="grid gap-1">
        <h1 class="text-2xl font-semibold tracking-tight">Inventory Stock Control</h1>
        <p class="text-sm text-muted-foreground">
          Real-time balances, lot FEFO expiry tracking, and movement actions.
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button variant="outline" @click="exportInventoryCSV">
          <FileSpreadsheet class="size-4" />
          Export CSV
        </Button>
        <Button v-if="canReceiveOrPick" @click="receiveVisible = true">
          <Download class="size-4" />
          Receive
        </Button>
        <Button v-if="canReceiveOrPick" variant="secondary" @click="pickVisible = true">
          <Upload class="size-4" />
          FEFO Pick
        </Button>
        <Button v-if="canTransferOrAdjust" variant="outline" @click="transferVisible = true">
          <ArrowLeftRight class="size-4" />
          Transfer
        </Button>
        <Button v-if="canTransferOrAdjust" variant="outline" @click="adjustVisible = true">
          <SlidersHorizontal class="size-4" />
          Adjust
        </Button>
      </div>
    </div>

    <Card>
      <CardContent class="grid gap-4">
        <div class="flex flex-wrap items-center gap-3">
          <Select v-model="selectedWarehouseId">
            <SelectTrigger class="w-[220px]">
              <SelectValue placeholder="Select Warehouse" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="wh in warehouseStore.warehouses" :key="wh.id" :value="wh.id">
                {{ wh.name }}
              </SelectItem>
            </SelectContent>
          </Select>

          <Select v-model="locationFilter">
            <SelectTrigger class="w-[190px]">
              <SelectValue placeholder="All Locations" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Locations</SelectItem>
              <SelectItem v-for="loc in currentWarehouseLocations" :key="loc.id" :value="loc.id">
                {{ loc.code }}
              </SelectItem>
            </SelectContent>
          </Select>

          <Select v-model="productFilter">
            <SelectTrigger class="w-[220px]">
              <SelectValue placeholder="All Products" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Products</SelectItem>
              <SelectItem v-for="product in catalogStore.products" :key="product.id" :value="product.id">
                {{ product.name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <p
          v-if="inventoryStore.error"
          role="alert"
          class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ inventoryStore.error }}
        </p>

        <div class="overflow-x-auto rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Location</TableHead>
                <TableHead>Product</TableHead>
                <TableHead>Lot / Expiration</TableHead>
                <TableHead>Available Qty</TableHead>
                <TableHead>Reserved Qty</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="inventoryStore.loading">
                <TableCell :colspan="5">
                  <div class="grid gap-2 py-2">
                    <Skeleton class="h-6 w-full" />
                    <Skeleton class="h-6 w-full" />
                    <Skeleton class="h-6 w-2/3" />
                  </div>
                </TableCell>
              </TableRow>

              <TableRow v-else-if="enrichedBalances.length === 0">
                <TableCell :colspan="5" class="py-10 text-center text-sm text-muted-foreground">
                  No stock balances found for the selected location or filters.
                </TableCell>
              </TableRow>

              <TableRow v-for="row in enrichedBalances" v-else :key="row.id">
                <TableCell>
                  <span class="inline-flex items-center gap-1 rounded border bg-muted px-2 py-0.5 font-mono text-xs font-semibold">
                    <MapPin class="size-3" />
                    {{ row.location_code }}
                  </span>
                </TableCell>
                <TableCell>
                  <div class="grid leading-tight">
                    <strong class="text-sm">{{ row.product_name }}</strong>
                    <small class="font-mono text-xs text-muted-foreground">SKU: {{ row.product_sku }}</small>
                  </div>
                </TableCell>
                <TableCell>
                  <div v-if="row.lot_number" class="grid gap-0.5">
                    <span class="font-mono text-xs font-semibold">Lot: {{ row.lot_number }}</span>
                    <span
                      v-if="row.expiration_date"
                      class="inline-flex items-center gap-1 text-[11px]"
                      :class="isExpiringSoon(row.expiration_date) ? 'font-semibold text-amber-600' : 'text-muted-foreground'"
                    >
                      Exp: {{ row.expiration_date }}
                      <Badge v-if="isExpiringSoon(row.expiration_date)" variant="outline" class="gap-1 border-amber-500/30 bg-amber-500/10 text-amber-700">
                        <AlertTriangle class="size-3" />
                        Expiring
                      </Badge>
                    </span>
                  </div>
                  <span v-else class="text-xs text-muted-foreground">Non-lot item</span>
                </TableCell>
                <TableCell>
                  <div class="flex items-center gap-2">
                    <span class="text-base font-semibold">{{ row.quantity }}</span>
                    <Badge v-if="isLowStock(row.quantity)" variant="outline" class="border-amber-500/30 bg-amber-500/10 text-amber-700">
                      Low Stock
                    </Badge>
                  </div>
                </TableCell>
                <TableCell class="text-sm text-muted-foreground">
                  {{ row.reserved_quantity || '0.000' }}
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>

    <ReceiveFormDialog
      v-model:visible="receiveVisible"
      :locations="currentWarehouseLocations"
      :products="catalogStore.products"
      @submitted="fetchBalances"
    />

    <PickFormDialog
      v-model:visible="pickVisible"
      :locations="currentWarehouseLocations"
      :products="catalogStore.products"
      @submitted="fetchBalances"
    />

    <TransferFormDialog
      v-model:visible="transferVisible"
      :locations="currentWarehouseLocations"
      :products="catalogStore.products"
      @submitted="fetchBalances"
    />

    <AdjustFormDialog
      v-model:visible="adjustVisible"
      :locations="currentWarehouseLocations"
      :products="catalogStore.products"
      @submitted="fetchBalances"
    />
  </div>
</template>
