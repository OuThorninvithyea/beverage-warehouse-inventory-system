<script setup lang="ts">
import { AlertTriangle, CalendarClock, PackageX } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'

import { Badge } from '@/components/ui/badge'
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
import { useInventoryStore } from '@/stores/inventory'
import { useWarehousesStore } from '@/stores/warehouses'

const inventoryStore = useInventoryStore()
const warehousesStore = useWarehousesStore()

const windowDays = ref('30')
const warehouseId = ref('all')

const windowOptions = [
  { value: '0', label: 'Expired only' },
  { value: '7', label: 'Next 7 days' },
  { value: '30', label: 'Next 30 days' },
  { value: '90', label: 'Next 90 days' },
  { value: '365', label: 'Next 12 months' },
]

const alerts = computed(() => inventoryStore.expiryAlerts)
const expiredCount = computed(
  () => alerts.value.filter((alert) => alert.status === 'expired').length,
)
const expiringCount = computed(
  () => alerts.value.filter((alert) => alert.status === 'expiring').length,
)
const expiredQuantity = computed(() =>
  alerts.value
    .filter((alert) => alert.status === 'expired')
    .reduce((total, alert) => total + Number(alert.quantity), 0),
)

async function load() {
  await inventoryStore.fetchExpiryAlerts({
    within_days: Number(windowDays.value),
    warehouse_id: warehouseId.value === 'all' ? undefined : warehouseId.value,
  })
}

onMounted(async () => {
  await warehousesStore.fetchWarehouses()
  await load()
})

watch([windowDays, warehouseId], load)

function describeRemaining(days: number) {
  if (days < 0) {
    return `${Math.abs(days)} ${Math.abs(days) === 1 ? 'day' : 'days'} ago`
  }
  if (days === 0) {
    return 'Today'
  }
  return `in ${days} ${days === 1 ? 'day' : 'days'}`
}
</script>

<template>
  <section class="grid gap-6">
    <div class="grid gap-2">
      <h1 class="text-3xl font-semibold tracking-tight">Expiry alerts</h1>
      <p class="text-sm text-muted-foreground">
        Lots that have expired or are approaching expiry, with the location
        holding them. Expired stock stays pickable and is flagged in the audit
        trail rather than blocked.
      </p>
    </div>

    <div class="grid grid-cols-3 gap-4 max-[800px]:grid-cols-1">
      <Card>
        <CardContent class="flex items-center gap-4 pt-6">
          <div class="rounded-xl bg-destructive/10 p-3 text-destructive">
            <PackageX class="size-5" />
          </div>
          <div>
            <p class="text-xs uppercase tracking-wide text-muted-foreground">Expired lots</p>
            <p class="text-2xl font-semibold">{{ expiredCount }}</p>
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="flex items-center gap-4 pt-6">
          <div class="rounded-xl bg-amber-500/10 p-3 text-amber-600">
            <CalendarClock class="size-5" />
          </div>
          <div>
            <p class="text-xs uppercase tracking-wide text-muted-foreground">Approaching expiry</p>
            <p class="text-2xl font-semibold">{{ expiringCount }}</p>
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardContent class="flex items-center gap-4 pt-6">
          <div class="rounded-xl bg-muted p-3 text-muted-foreground">
            <AlertTriangle class="size-5" />
          </div>
          <div>
            <p class="text-xs uppercase tracking-wide text-muted-foreground">Expired quantity</p>
            <p class="text-2xl font-semibold">{{ expiredQuantity }}</p>
          </div>
        </CardContent>
      </Card>
    </div>

    <Card>
      <CardContent class="grid gap-4 pt-6">
        <div class="flex flex-wrap items-center gap-3">
          <Select v-model="windowDays">
            <SelectTrigger class="w-[190px]">
              <SelectValue placeholder="Time window" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="option in windowOptions" :key="option.value" :value="option.value">
                {{ option.label }}
              </SelectItem>
            </SelectContent>
          </Select>

          <Select v-model="warehouseId">
            <SelectTrigger class="w-[220px]">
              <SelectValue placeholder="All warehouses" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All warehouses</SelectItem>
              <SelectItem
                v-for="warehouse in warehousesStore.warehouses"
                :key="warehouse.id"
                :value="warehouse.id"
              >
                {{ warehouse.code }} — {{ warehouse.name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div v-if="inventoryStore.loading" class="grid gap-2">
          <Skeleton v-for="row in 5" :key="row" class="h-10 w-full" />
        </div>

        <p
          v-else-if="inventoryStore.error"
          class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ inventoryStore.error }}
        </p>

        <p v-else-if="alerts.length === 0" class="py-8 text-center text-sm text-muted-foreground">
          Nothing expires in this window. Stock is in date.
        </p>

        <Table v-else>
          <TableHeader>
            <TableRow>
              <TableHead>Status</TableHead>
              <TableHead>Product</TableHead>
              <TableHead>Lot</TableHead>
              <TableHead>Expires</TableHead>
              <TableHead>Location</TableHead>
              <TableHead class="text-right">On hand</TableHead>
              <TableHead class="text-right">Available</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="alert in alerts" :key="`${alert.lot_id}-${alert.location_id}`">
              <TableCell>
                <Badge :variant="alert.status === 'expired' ? 'destructive' : 'secondary'">
                  {{ alert.status === 'expired' ? 'Expired' : 'Expiring' }}
                </Badge>
              </TableCell>
              <TableCell>
                <div class="font-medium">{{ alert.product_name }}</div>
                <div class="text-xs text-muted-foreground">{{ alert.sku }}</div>
              </TableCell>
              <TableCell class="font-mono text-xs">{{ alert.lot_number }}</TableCell>
              <TableCell>
                <div>{{ alert.expiration_date }}</div>
                <div
                  class="text-xs"
                  :class="alert.days_remaining < 0 ? 'text-destructive' : 'text-muted-foreground'"
                >
                  {{ describeRemaining(alert.days_remaining) }}
                </div>
              </TableCell>
              <TableCell>
                <div>{{ alert.location_code }}</div>
                <div class="text-xs text-muted-foreground">{{ alert.warehouse_code }}</div>
              </TableCell>
              <TableCell class="text-right tabular-nums">{{ alert.quantity }}</TableCell>
              <TableCell class="text-right tabular-nums">{{ alert.available_quantity }}</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  </section>
</template>
