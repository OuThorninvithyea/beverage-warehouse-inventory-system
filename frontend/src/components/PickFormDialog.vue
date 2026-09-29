<script setup lang="ts">
import { Clock, Upload } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'

import type { Product } from '@/api/catalog'
import type { PickInput } from '@/api/inventory'
import type { Location } from '@/api/warehouses'
import { Badge } from '@/components/ui/badge'
import LocationSelect from '@/components/LocationSelect.vue'
import ScanBar from '@/components/ScanBar.vue'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { useLocationLabels } from '@/lib/location-label'
import { formatQuantity, useLocationStock, type StockedLot } from '@/lib/location-stock'
import { resolveScan, type ScanFeedback } from '@/lib/scan'
import { useInventoryStore } from '@/stores/inventory'

const props = defineProps<{
  visible: boolean
  locations: Location[]
  products: Product[]
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'submitted'): void
}>()

const inventoryStore = useInventoryStore()

const locationId = ref('')
const productId = ref('')
const lotId = ref<string>('auto')
const quantity = ref<string>('1')
const reference = ref('')
const notes = ref('')

const errorMessage = ref('')
const scanFeedback = ref<ScanFeedback | null>(null)
const labels = useLocationLabels()
const availableLots = ref<StockedLot[]>([])

// Only what is actually on the chosen shelf can be picked, so the product and
// lot lists come from that location's stock rather than the whole catalog.
const stock = useLocationStock(locationId)
const activeLocations = computed(() => props.locations.filter((loc) => loc.is_active))
const locationCode = computed(
  () => labels.label(props.locations.find((loc) => loc.id === locationId.value)) || 'this location',
)

const selectedProduct = computed<Product | undefined>(
  () => stock.stockedProducts.value.find((row) => row.product.id === productId.value)?.product,
)
const maxAvailable = computed(() =>
  stock.availableFor(productId.value, lotId.value === 'auto' ? null : lotId.value),
)

// A product chosen for one location may not exist on the next shelf.
watch(locationId, () => {
  productId.value = ''
})

watch(
  () => productId.value,
  async (newProdId) => {
    lotId.value = 'auto'
    availableLots.value = []
    if (newProdId) {
      availableLots.value = await stock.lotsFor(newProdId)
    }
  },
)

watch(
  () => props.visible,
  async (isVis) => {
    if (!isVis) return
    errorMessage.value = ''
    scanFeedback.value = null
    productId.value = ''
    lotId.value = 'auto'
    quantity.value = '1'
    reference.value = ''
    notes.value = ''
    availableLots.value = []
    await stock.loadCatalog()
    const first = activeLocations.value[0]?.id || ''
    if (locationId.value === first) await stock.reload()
    else locationId.value = first
  },
)

// Scan the shelf first, then the carton.
async function onScan(code: string) {
  try {
    const result = await resolveScan(code, props.locations)
    if (result.kind === 'location') {
      const { location } = result
      if (!location.is_active) {
        scanFeedback.value = { tone: 'warning', text: `${labels.label(location)} is inactive.` }
        return
      }
      locationId.value = location.id
      scanFeedback.value = { tone: 'success', text: `Picking from ${labels.label(location)}. Now scan the carton.` }
      return
    }
    if (result.kind === 'unknown') {
      scanFeedback.value = { tone: 'warning', text: `${result.code} is not a shelf or product in the system.` }
      return
    }
    const { product } = result
    if (stock.loading.value) await stock.reload()
    if (stock.availableFor(product.id) <= 0) {
      scanFeedback.value = { tone: 'warning', text: `${locationCode.value} has no ${product.name} available.` }
      return
    }
    productId.value = product.id
    scanFeedback.value = {
      tone: 'success',
      text: `${product.name} · ${formatQuantity(stock.availableFor(product.id))} available here.`,
    }
  } catch (err: unknown) {
    scanFeedback.value = { tone: 'warning', text: err instanceof Error ? err.message : 'Scan lookup failed' }
  }
}

async function submitPick() {
  errorMessage.value = ''
  const qty = Number(quantity.value)

  if (!locationId.value) {
    errorMessage.value = 'Source location is required'
    return
  }
  if (!productId.value) {
    errorMessage.value = 'Product is required'
    return
  }
  if (!Number.isFinite(qty) || qty <= 0) {
    errorMessage.value = 'Pick quantity must be greater than 0'
    return
  }
  if (qty > maxAvailable.value) {
    errorMessage.value = `Only ${formatQuantity(maxAvailable.value)} available in ${locationCode.value}`
    return
  }

  const payload: PickInput = {
    location_id: locationId.value,
    product_id: productId.value,
    quantity: qty.toString(),
    lot_id: lotId.value === 'auto' ? null : lotId.value,
    reference: reference.value.trim() || null,
    notes: notes.value.trim() || null,
  }

  try {
    await inventoryStore.doPick(payload)
    emit('submitted')
    closeDialog()
  } catch (err: unknown) {
    errorMessage.value = err instanceof Error ? err.message : 'FEFO Pick failed'
  }
}

function closeDialog() {
  emit('update:visible', false)
}
</script>

<template>
  <Dialog :open="visible" @update:open="(value: boolean) => emit('update:visible', value)">
    <DialogContent class="sm:max-w-2xl">
      <DialogHeader>
        <DialogTitle>FEFO Stock Pick</DialogTitle>
        <DialogDescription>Outbound picking with first-expiry-first-out allocation.</DialogDescription>
      </DialogHeader>

      <form class="grid gap-4" @submit.prevent="submitPick">
        <p
          v-if="errorMessage"
          class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ errorMessage }}
        </p>

        <ScanBar :feedback="scanFeedback" hint="Scan the shelf label, then the carton." @scan="onScan" />

        <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="pick-location">Source Location *</Label>
            <LocationSelect
              id="pick-location"
              v-model="locationId"
              :locations="activeLocations"
              placeholder="Select location"
            />
          </div>

          <div class="grid gap-2">
            <Label for="pick-product">Product *</Label>
            <Select v-model="productId" :disabled="stock.loading.value || stock.stockedProducts.value.length === 0">
              <SelectTrigger id="pick-product" class="w-full">
                <SelectValue
                  :placeholder="stock.loading.value ? 'Loading stock…' : stock.stockedProducts.value.length === 0 ? 'No stock here' : 'Select product'"
                />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="row in stock.stockedProducts.value" :key="row.product.id" :value="row.product.id">
                  {{ row.product.name }} · {{ formatQuantity(row.available) }} available
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <p
          v-if="locationId && !stock.loading.value && stock.stockedProducts.value.length === 0"
          class="rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-sm text-amber-700"
        >
          {{ locationCode }} has no available stock. Choose another location.
        </p>

        <div v-if="selectedProduct" class="flex items-center justify-between rounded-lg border bg-muted/40 p-3 text-xs">
          <div class="grid gap-0.5">
            <span class="text-sm font-semibold">{{ selectedProduct.name }}</span>
            <span class="text-muted-foreground">
              SKU: {{ selectedProduct.sku }} | Unit: {{ selectedProduct.unit }}
            </span>
          </div>
          <Badge v-if="selectedProduct.is_lot_tracked" variant="outline" class="border-sky-500/30 bg-sky-500/10 text-sky-700">
            FEFO Auto Allocation
          </Badge>
        </div>

        <div v-if="availableLots.length > 0" class="rounded-lg border bg-sky-500/5 p-3 text-xs">
          <div class="mb-2 flex items-center gap-1 font-semibold text-sky-700">
            <Clock class="size-3.5" />
            FEFO Lots (Earliest Expiration First)
          </div>
          <div class="grid max-h-[110px] gap-1 overflow-y-auto pr-1">
            <div
              v-for="row in availableLots"
              :key="row.lot.id"
              class="flex items-center justify-between rounded border bg-background p-1.5"
            >
              <span>Lot: <strong>{{ row.lot.lot_number }}</strong> · {{ formatQuantity(row.available) }} here</span>
              <span class="font-medium text-amber-700">Expires: {{ row.lot.expiration_date || 'N/A' }}</span>
            </div>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="pick-qty">Pick Quantity *</Label>
            <Input
              id="pick-qty"
              v-model="quantity"
              type="number"
              min="0.001"
              :max="maxAvailable || undefined"
              step="0.001"
              placeholder="e.g. 10"
              required
            />
            <small v-if="productId" class="text-xs text-muted-foreground">
              Up to {{ formatQuantity(maxAvailable) }} available
            </small>
          </div>

          <div class="grid gap-2">
            <Label for="pick-lot">Specific Lot (auto FEFO if unset)</Label>
            <Select v-model="lotId">
              <SelectTrigger id="pick-lot" class="w-full">
                <SelectValue placeholder="Auto FEFO (Recommended)" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="auto">Auto FEFO (Recommended)</SelectItem>
                <SelectItem v-for="row in availableLots" :key="row.lot.id" :value="row.lot.id">
                  {{ row.lot.lot_number }} · {{ formatQuantity(row.available) }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-4 border-t pt-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="pick-ref">Sales Order / Pick Ref</Label>
            <Input id="pick-ref" v-model="reference" placeholder="e.g. SO-2201" />
          </div>

          <div class="grid gap-2">
            <Label for="pick-notes">Notes</Label>
            <Textarea id="pick-notes" v-model="notes" rows="1" placeholder="Optional picking notes" />
          </div>
        </div>
      </form>

      <DialogFooter>
        <Button variant="outline" @click="closeDialog">Cancel</Button>
        <Button :disabled="inventoryStore.loading" @click="submitPick">
          <Upload class="size-4" />
          Execute Pick
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

</template>
