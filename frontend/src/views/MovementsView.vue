<script setup lang="ts">
import {
  ArrowLeftRight,
  ArrowRight,
  Download,
  FileSpreadsheet,
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
import { useUsersStore } from '@/stores/users'
import { useWarehousesStore } from '@/stores/warehouses'

const auth = useAuthStore()
const inventoryStore = useInventoryStore()
const catalogStore = useCatalogStore()
const warehouseStore = useWarehousesStore()
const usersStore = useUsersStore()

type MovementType = 'receive' | 'pick' | 'transfer' | 'adjust'

const typeFilter = ref<'all' | MovementType>('all')
const productFilter = ref<string>('all')

const movementTypeFilter = computed(() =>
  typeFilter.value === 'all' ? undefined : (typeFilter.value as MovementType),
)
const selectedProductId = computed(() =>
  productFilter.value === 'all' ? undefined : productFilter.value,
)

const receiveVisible = ref(false)
const pickVisible = ref(false)
const transferVisible = ref(false)
const adjustVisible = ref(false)

interface EnrichedMovement {
  id: string
  type: string
  created_at: string
  quantity: string
  reference: string | null
  product_name: string
  product_sku: string
  from_location_code: string | null
  to_location_code: string | null
  performer_name: string
}

const enrichedMovements = computed<EnrichedMovement[]>(() => {
  return inventoryStore.movements.map((m) => {
    const product = catalogStore.products.find((p) => p.id === m.product_id)
    const fromLocation = warehouseStore.locations.find((l) => l.id === m.from_location_id)
    const toLocation = warehouseStore.locations.find((l) => l.id === m.to_location_id)
    const performer = m.performed_by ? usersStore.users.find((u) => u.id === m.performed_by) : undefined
    return {
      id: m.id,
      type: m.movement_type,
      created_at: m.created_at,
      quantity: m.quantity,
      reference: m.reference,
      product_name: product?.name ?? 'Product',
      product_sku: product?.sku ?? '—',
      from_location_code: fromLocation?.code ?? null,
      to_location_code: toLocation?.code ?? null,
      performer_name: performer?.full_name ?? (m.performed_by ? 'Unknown Operator' : 'System Operator'),
    }
  })
})

function exportMovementsCSV() {
  exportToCSV('stock_movement_audit_log', enrichedMovements.value, [
    { key: 'created_at', label: 'Timestamp' },
    { key: 'type', label: 'Type' },
    { key: 'product_sku', label: 'SKU' },
    { key: 'product_name', label: 'Product Name' },
    { key: 'quantity', label: 'Quantity' },
    { key: 'from_location_code', label: 'From Location' },
    { key: 'to_location_code', label: 'To Location' },
    { key: 'reference', label: 'Reference' },
    { key: 'performer_name', label: 'Operator' },
  ])
}

const canReceiveOrPick = computed(() => {
  return auth.user?.role === 'admin' || auth.user?.role === 'warehouse_manager' || auth.user?.role === 'picker'
})

const canTransferOrAdjust = computed(() => {
  return auth.user?.role === 'admin' || auth.user?.role === 'warehouse_manager'
})

onMounted(async () => {
  await warehouseStore.fetchWarehouses()
  await Promise.all([
    warehouseStore.fetchAllLocations(),
    catalogStore.fetchProducts(),
    inventoryStore.fetchMovements(),
  ])
  if (auth.user?.role === 'admin') {
    void usersStore.fetchUsers()
  }
})

watch([movementTypeFilter, selectedProductId], () => {
  void inventoryStore.fetchMovements({
    movement_type: movementTypeFilter.value,
    product_id: selectedProductId.value,
  })
})

function getMovementClass(type: string): string {
  switch (type) {
    case 'receive':
      return 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700'
    case 'pick':
      return 'border-amber-500/30 bg-amber-500/10 text-amber-700'
    case 'transfer':
      return 'border-sky-500/30 bg-sky-500/10 text-sky-700'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '—'
  const date = new Date(dateStr)
  return date.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}
</script>

<template>
  <div class="grid gap-6">
    <div class="flex flex-wrap items-end justify-between gap-4">
      <div class="grid gap-1">
        <h1 class="text-2xl font-semibold tracking-tight">Stock Movement Audit Log</h1>
        <p class="text-sm text-muted-foreground">
          Immutable history of receives, FEFO picks, transfers, and adjustments.
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button variant="outline" @click="exportMovementsCSV">
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
          <Select v-model="typeFilter">
            <SelectTrigger class="w-[210px]">
              <SelectValue placeholder="All Movement Types" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Movement Types</SelectItem>
              <SelectItem value="receive">Receive</SelectItem>
              <SelectItem value="pick">Pick (FEFO)</SelectItem>
              <SelectItem value="transfer">Transfer</SelectItem>
              <SelectItem value="adjust">Adjustment</SelectItem>
            </SelectContent>
          </Select>

          <Select v-model="productFilter">
            <SelectTrigger class="w-[240px]">
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
                <TableHead>Timestamp</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Product</TableHead>
                <TableHead>Quantity</TableHead>
                <TableHead>Locations</TableHead>
                <TableHead>Reference</TableHead>
                <TableHead>Operator</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="inventoryStore.loading">
                <TableCell :colspan="7">
                  <div class="grid gap-2 py-2">
                    <Skeleton class="h-6 w-full" />
                    <Skeleton class="h-6 w-full" />
                    <Skeleton class="h-6 w-2/3" />
                  </div>
                </TableCell>
              </TableRow>

              <TableRow v-else-if="enrichedMovements.length === 0">
                <TableCell :colspan="7" class="py-10 text-center text-sm text-muted-foreground">
                  No movement audit records found.
                </TableCell>
              </TableRow>

              <TableRow v-for="movement in enrichedMovements" v-else :key="movement.id">
                <TableCell class="text-xs font-medium text-muted-foreground">
                  {{ formatDate(movement.created_at) }}
                </TableCell>
                <TableCell>
                  <Badge variant="outline" :class="['font-mono text-[10px] uppercase', getMovementClass(movement.type)]">
                    {{ movement.type }}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div class="grid leading-tight">
                    <strong class="text-sm">{{ movement.product_name }}</strong>
                    <small class="font-mono text-xs text-muted-foreground">SKU: {{ movement.product_sku }}</small>
                  </div>
                </TableCell>
                <TableCell class="font-semibold">{{ movement.quantity }}</TableCell>
                <TableCell>
                  <div class="flex items-center gap-1 font-mono text-xs">
                    <span
                      v-if="movement.from_location_code"
                      class="rounded border border-destructive/20 bg-destructive/10 px-1 text-destructive"
                    >
                      {{ movement.from_location_code }}
                    </span>
                    <ArrowRight
                      v-if="movement.from_location_code && movement.to_location_code"
                      class="size-3 text-muted-foreground"
                    />
                    <span
                      v-if="movement.to_location_code"
                      class="rounded border border-emerald-500/20 bg-emerald-500/10 px-1 text-emerald-700"
                    >
                      {{ movement.to_location_code }}
                    </span>
                  </div>
                </TableCell>
                <TableCell>
                  <span v-if="movement.reference" class="font-mono text-xs font-semibold">
                    {{ movement.reference }}
                  </span>
                  <span v-else class="text-xs text-muted-foreground">&mdash;</span>
                </TableCell>
                <TableCell class="text-xs text-muted-foreground">
                  {{ movement.performer_name }}
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>

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
  </div>
</template>
