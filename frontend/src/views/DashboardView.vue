<script setup lang="ts">
import {
  ArrowLeftRight,
  ArrowRight,
  Download,
  Upload,
} from 'lucide-vue-next'
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'

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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import type { Lot, StockMovement } from '@/api/inventory'
import { useAuthStore } from '@/stores/auth'
import { useCatalogStore } from '@/stores/catalog'
import { useInventoryStore } from '@/stores/inventory'
import { useWarehousesStore } from '@/stores/warehouses'
import lowStockSticker from '@/assets/sticker-low-stock.png'
import nearExpirySticker from '@/assets/sticker-near-expiry.png'
import totalSkusSticker from '@/assets/sticker-total-skus.png'

const auth = useAuthStore()
const catalogStore = useCatalogStore()
const inventoryStore = useInventoryStore()
const warehouseStore = useWarehousesStore()

const receiveVisible = ref(false)
const pickVisible = ref(false)
const transferVisible = ref(false)
const lotsByProduct = ref<Map<string, Lot[]>>(new Map())

const canReceiveOrPick = computed(() =>
  ['admin', 'warehouse_manager', 'picker'].includes(auth.user?.role ?? ''),
)
const canTransfer = computed(() => ['admin', 'warehouse_manager'].includes(auth.user?.role ?? ''))

const totalSkus = computed(() => catalogStore.products.length)
const lowStockCount = computed(() =>
  inventoryStore.balances.filter((balance) => Number(balance.available_quantity) < 10).length,
)

const nearExpiryLots = computed(() => {
  const today = new Date()
  let count = 0
  for (const lots of lotsByProduct.value.values()) {
    for (const lot of lots) {
      if (!lot.expiration_date || Number(lot.available_quantity ?? 0) <= 0) continue
      const days = Math.ceil(
        (new Date(lot.expiration_date).getTime() - today.getTime()) / (1000 * 60 * 60 * 24),
      )
      if (days >= 0 && days <= 30) count += 1
    }
  }
  return count
})

const recentMovements = computed(() => inventoryStore.movements.slice(0, 5))

function productName(productId: string): string {
  return catalogStore.products.find((product) => product.id === productId)?.name ?? 'Unknown product'
}

function productSku(productId: string): string {
  return catalogStore.products.find((product) => product.id === productId)?.sku ?? '—'
}

function movementClass(type: StockMovement['movement_type']): string {
  if (type === 'receive') return 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700'
  if (type === 'pick') return 'border-amber-500/30 bg-amber-500/10 text-amber-700'
  if (type === 'transfer') return 'border-sky-500/30 bg-sky-500/10 text-sky-700'
  return 'border-slate-500/30 bg-slate-500/10 text-slate-700'
}

function formatDate(value: string): string {
  return new Date(value).toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

async function refreshAfterMovement() {
  await Promise.all([inventoryStore.fetchBalances(), inventoryStore.fetchMovements()])
}

onMounted(async () => {
  await Promise.all([
    warehouseStore.fetchWarehouses(),
    catalogStore.fetchProducts(),
    inventoryStore.fetchBalances(),
    inventoryStore.fetchMovements(),
  ])
  await warehouseStore.fetchAllLocations()

  const productIds = new Set(inventoryStore.balances.map((balance) => balance.product_id))
  await Promise.all(
    [...productIds].map(async (productId) => {
      lotsByProduct.value.set(productId, await inventoryStore.fetchLots(productId))
    }),
  )
})
</script>

<template>
  <div class="grid gap-6">
    <div class="flex flex-wrap items-end justify-between gap-4">
      <div class="grid gap-1">
        <p class="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Operations overview</p>
        <h1 class="text-2xl font-semibold tracking-tight">Good morning, {{ auth.user?.full_name }}</h1>
        <p class="text-sm text-muted-foreground">
          Keep stock moving safely across {{ warehouseStore.warehouses.length || 'your' }} warehouse(s).
        </p>
      </div>
      <Badge variant="outline" class="border-emerald-500/30 bg-emerald-500/10 text-emerald-700">System live</Badge>
    </div>

    <div class="grid gap-4 md:grid-cols-3">
      <Card>
        <CardContent class="flex items-center justify-between">
          <div class="grid gap-1">
            <span class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Total SKUs</span>
            <strong class="text-3xl font-semibold">{{ totalSkus }}</strong>
            <span class="text-xs text-muted-foreground">Active catalog items</span>
          </div>
          <img :src="totalSkusSticker" alt="" class="size-[100px] shrink-0 object-contain drop-shadow-sm" />
        </CardContent>
      </Card>
      <Card>
        <CardContent class="flex items-center justify-between">
          <div class="grid gap-1">
            <span class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Low Stock</span>
            <strong class="text-3xl font-semibold">{{ lowStockCount }}</strong>
            <span class="text-xs text-muted-foreground">Balances below 10 units</span>
          </div>
          <img :src="lowStockSticker" alt="" class="size-[100px] shrink-0 object-contain drop-shadow-sm" />
        </CardContent>
      </Card>
      <Card>
        <CardContent class="flex items-center justify-between">
          <div class="grid gap-1">
            <span class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Near-Expiry Lots</span>
            <strong class="text-3xl font-semibold">{{ nearExpiryLots }}</strong>
            <span class="text-xs text-muted-foreground">Expiring within 30 days</span>
          </div>
          <img :src="nearExpirySticker" alt="" class="size-[100px] shrink-0 object-contain drop-shadow-sm" />
        </CardContent>
      </Card>
    </div>

    <Card>
      <CardHeader>
        <CardTitle class="text-base">Inventory operations</CardTitle>
        <CardDescription>Start the next warehouse task without leaving the dashboard.</CardDescription>
      </CardHeader>
      <CardContent class="grid gap-3 sm:grid-cols-3">
        <Button v-if="canReceiveOrPick" class="h-auto justify-start gap-3 p-4" @click="receiveVisible = true">
          <Download class="size-5" /><span class="grid justify-items-start gap-0.5"><strong>Receive Stock</strong><small class="font-normal opacity-80">Record inbound stock</small></span>
        </Button>
        <Button v-if="canReceiveOrPick" variant="secondary" class="h-auto justify-start gap-3 p-4" @click="pickVisible = true">
          <Upload class="size-5" /><span class="grid justify-items-start gap-0.5"><strong>Pick Order</strong><small class="font-normal opacity-80">Pick by FEFO</small></span>
        </Button>
        <Button v-if="canTransfer" variant="outline" class="h-auto justify-start gap-3 p-4" @click="transferVisible = true">
          <ArrowLeftRight class="size-5" /><span class="grid justify-items-start gap-0.5"><strong>Transfer Stock</strong><small class="font-normal opacity-80">Move between locations</small></span>
        </Button>
        <p v-if="!canReceiveOrPick && !canTransfer" class="text-sm text-muted-foreground">Viewer access is read-only. Transactional actions are disabled.</p>
      </CardContent>
    </Card>

    <Card>
      <CardHeader class="flex-row items-center justify-between space-y-0">
        <div><CardTitle class="text-base">Recent movements</CardTitle><CardDescription>Latest receives, picks, transfers, and adjustments.</CardDescription></div>
        <Button variant="ghost" size="sm" as-child><RouterLink to="/movements">View history <ArrowRight class="size-4" /></RouterLink></Button>
      </CardHeader>
      <CardContent>
        <div class="overflow-x-auto rounded-lg border">
          <Table>
            <TableHeader><TableRow><TableHead>Date</TableHead><TableHead>Type</TableHead><TableHead>Product</TableHead><TableHead>Quantity</TableHead><TableHead>Reference</TableHead></TableRow></TableHeader>
            <TableBody>
              <TableRow v-if="recentMovements.length === 0"><TableCell colspan="5" class="py-10 text-center text-sm text-muted-foreground">No movements recorded yet.</TableCell></TableRow>
              <TableRow v-for="movement in recentMovements" v-else :key="movement.id">
                <TableCell class="whitespace-nowrap text-xs text-muted-foreground">{{ formatDate(movement.created_at) }}</TableCell>
                <TableCell><Badge variant="outline" :class="['uppercase', movementClass(movement.movement_type)]">{{ movement.movement_type }}</Badge></TableCell>
                <TableCell><div class="grid"><strong class="text-sm">{{ productName(movement.product_id) }}</strong><small class="font-mono text-xs text-muted-foreground">{{ productSku(movement.product_id) }}</small></div></TableCell>
                <TableCell class="font-semibold">{{ movement.quantity }}</TableCell>
                <TableCell class="text-xs text-muted-foreground">{{ movement.reference || '—' }}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>

    <ReceiveFormDialog v-model:visible="receiveVisible" :locations="warehouseStore.locations" :products="catalogStore.products" @submitted="refreshAfterMovement" />
    <PickFormDialog v-model:visible="pickVisible" :locations="warehouseStore.locations" :products="catalogStore.products" @submitted="refreshAfterMovement" />
    <TransferFormDialog v-model:visible="transferVisible" :locations="warehouseStore.locations" :products="catalogStore.products" @submitted="refreshAfterMovement" />
  </div>
</template>
