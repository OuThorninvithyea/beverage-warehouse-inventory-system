<script setup lang="ts">
import { ArrowLeftRight, ArrowRight } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'

import type { Product } from '@/api/catalog'
import type { TransferInput } from '@/api/inventory'
import type { Location } from '@/api/warehouses'
import LocationSelect from '@/components/LocationSelect.vue'
import ScanBar from '@/components/ScanBar.vue'
import { Badge } from '@/components/ui/badge'
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

const productId = ref('')
const fromLocationId = ref('')
const toLocationId = ref('')
const lotId = ref<string>('')
const quantity = ref<string>('1')
const reference = ref('')
const notes = ref('')

const errorMessage = ref('')
const scanFeedback = ref<ScanFeedback | null>(null)
// The first shelf scanned is the source, the next one the destination.
const nextShelf = ref<'from' | 'to'>('from')
const availableLots = ref<StockedLot[]>([])

// A transfer moves stock that is already on the source shelf, so the product
// and lot lists come from that location's stock rather than the catalog.
const stock = useLocationStock(fromLocationId)
const activeLocations = computed(() => props.locations.filter((loc) => loc.is_active))
const labels = useLocationLabels()
const sourceLocation = computed(() => props.locations.find((loc) => loc.id === fromLocationId.value))
const destinationLocation = computed(() =>
  props.locations.find((loc) => loc.id === toLocationId.value),
)
const sourceCode = computed(() => labels.label(sourceLocation.value) || 'the source location')
// Stock leaving for another branch is worth calling out: it changes which
// warehouse owns the stock and its cost.
const crossesWarehouses = computed(
  () =>
    !!sourceLocation.value &&
    !!destinationLocation.value &&
    sourceLocation.value.warehouse_id !== destinationLocation.value.warehouse_id,
)

const selectedProduct = computed<Product | undefined>(
  () => stock.stockedProducts.value.find((row) => row.product.id === productId.value)?.product,
)
const maxAvailable = computed(() => stock.availableFor(productId.value, lotId.value || null))

const destinationLocations = computed(() =>
  activeLocations.value.filter((loc) => loc.id !== fromLocationId.value),
)

// Whatever was chosen for the old source may not exist on the new one.
watch(fromLocationId, () => {
  productId.value = ''
  if (toLocationId.value === fromLocationId.value) {
    toLocationId.value = destinationLocations.value[0]?.id || ''
  }
})

watch(
  () => productId.value,
  async (newProdId) => {
    lotId.value = ''
    availableLots.value = []
    if (newProdId) {
      availableLots.value = await stock.lotsFor(newProdId)
      // With a single lot on the shelf there is nothing to choose.
      if (availableLots.value.length === 1) lotId.value = availableLots.value[0].lot.id
    }
  },
)

watch(
  () => props.visible,
  async (isVis) => {
    if (!isVis) return
    errorMessage.value = ''
    scanFeedback.value = null
    nextShelf.value = 'from'
    productId.value = ''
    lotId.value = ''
    quantity.value = '1'
    reference.value = ''
    notes.value = ''
    availableLots.value = []
    await stock.loadCatalog()
    const first = activeLocations.value[0]?.id || ''
    if (fromLocationId.value === first) await stock.reload()
    else fromLocationId.value = first
    toLocationId.value = destinationLocations.value[0]?.id || ''
  },
)

async function onScan(code: string) {
  try {
    const result = await resolveScan(code, props.locations)
    if (result.kind === 'location') {
      const { location } = result
      const name = labels.label(location)
      if (!location.is_active) {
        scanFeedback.value = { tone: 'warning', text: `${name} is inactive.` }
        return
      }
      if (nextShelf.value === 'from') {
        fromLocationId.value = location.id
        nextShelf.value = 'to'
        scanFeedback.value = { tone: 'success', text: `From ${name}. Now scan the destination shelf.` }
        return
      }
      if (location.id === fromLocationId.value) {
        scanFeedback.value = { tone: 'warning', text: `${name} is already the source. Scan a different shelf.` }
        return
      }
      toLocationId.value = location.id
      nextShelf.value = 'from'
      scanFeedback.value = { tone: 'success', text: `To ${name}. Now scan the carton.` }
      return
    }
    if (result.kind === 'unknown') {
      scanFeedback.value = { tone: 'warning', text: `${result.code} is not a shelf or product in the system.` }
      return
    }
    const { product } = result
    if (stock.loading.value) await stock.reload()
    if (stock.availableFor(product.id) <= 0) {
      scanFeedback.value = { tone: 'warning', text: `${sourceCode.value} has no ${product.name} available.` }
      return
    }
    productId.value = product.id
    scanFeedback.value = {
      tone: 'success',
      text: `${product.name} · ${formatQuantity(stock.availableFor(product.id))} available to move.`,
    }
  } catch (err: unknown) {
    scanFeedback.value = { tone: 'warning', text: err instanceof Error ? err.message : 'Scan lookup failed' }
  }
}

async function submitTransfer() {
  errorMessage.value = ''
  const qty = Number(quantity.value)

  if (!productId.value) {
    errorMessage.value = 'Product is required'
    return
  }
  if (!fromLocationId.value) {
    errorMessage.value = 'Source location is required'
    return
  }
  if (!toLocationId.value) {
    errorMessage.value = 'Destination location is required'
    return
  }
  if (fromLocationId.value === toLocationId.value) {
    errorMessage.value = 'Source and Destination locations must be different'
    return
  }
  if (!Number.isFinite(qty) || qty <= 0) {
    errorMessage.value = 'Transfer quantity must be greater than 0'
    return
  }
  if (selectedProduct.value?.is_lot_tracked && !lotId.value) {
    errorMessage.value = 'Lot selection is required for lot-tracked products'
    return
  }
  if (qty > maxAvailable.value) {
    errorMessage.value = `Only ${formatQuantity(maxAvailable.value)} available in ${sourceCode.value}`
    return
  }

  const payload: TransferInput = {
    product_id: productId.value,
    from_location_id: fromLocationId.value,
    to_location_id: toLocationId.value,
    quantity: qty.toString(),
    lot_id: lotId.value || null,
    reference: reference.value.trim() || null,
    notes: notes.value.trim() || null,
  }

  try {
    await inventoryStore.doTransfer(payload)
    emit('submitted')
    closeDialog()
  } catch (err: unknown) {
    errorMessage.value = err instanceof Error ? err.message : 'Stock transfer failed'
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
        <DialogTitle>Transfer Stock Between Locations</DialogTitle>
        <DialogDescription>Move stock while preserving lot and audit history.</DialogDescription>
      </DialogHeader>

      <form class="grid gap-4" @submit.prevent="submitTransfer">
        <p
          v-if="errorMessage"
          class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ errorMessage }}
        </p>

        <ScanBar
          :feedback="scanFeedback"
          hint="Scan the source shelf, the destination shelf, then the carton."
          @scan="onScan"
        />

        <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="trf-from">From Location (Source) *</Label>
            <LocationSelect
              id="trf-from"
              v-model="fromLocationId"
              :locations="activeLocations"
              placeholder="Select source location"
            />
          </div>

          <div class="grid gap-2">
            <Label for="trf-to">To Location (Destination) *</Label>
            <LocationSelect
              id="trf-to"
              v-model="toLocationId"
              :locations="destinationLocations"
              placeholder="Select destination location"
            />
          </div>
        </div>

        <div
          v-if="sourceLocation && destinationLocation"
          class="flex flex-wrap items-center gap-x-2 gap-y-1 rounded-md border bg-muted/40 px-3 py-2 text-sm"
        >
          <span class="font-medium">{{ labels.label(sourceLocation) }}</span>
          <ArrowRight class="size-4 text-muted-foreground" />
          <span class="font-medium">{{ labels.label(destinationLocation) }}</span>
          <Badge v-if="crossesWarehouses" variant="secondary" class="ml-auto">Between warehouses</Badge>
          <Badge v-else variant="outline" class="ml-auto">Same warehouse</Badge>
        </div>

        <div class="grid gap-2">
          <Label for="trf-product">Product *</Label>
          <Select v-model="productId" :disabled="stock.loading.value || stock.stockedProducts.value.length === 0">
            <SelectTrigger id="trf-product" class="w-full">
              <SelectValue
                :placeholder="stock.loading.value ? 'Loading stock…' : stock.stockedProducts.value.length === 0 ? 'No stock at the source' : 'Select product to transfer'"
              />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="row in stock.stockedProducts.value" :key="row.product.id" :value="row.product.id">
                {{ row.product.name }} · {{ formatQuantity(row.available) }} available
              </SelectItem>
            </SelectContent>
          </Select>
          <p
            v-if="fromLocationId && !stock.loading.value && stock.stockedProducts.value.length === 0"
            class="rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-sm text-amber-700"
          >
            {{ sourceCode }} has no available stock. Choose another source location.
          </p>
        </div>

        <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="trf-qty">Transfer Quantity *</Label>
            <Input
              id="trf-qty"
              v-model="quantity"
              type="number"
              min="0.001"
              :max="maxAvailable || undefined"
              step="0.001"
              placeholder="e.g. 5"
              required
            />
            <small v-if="productId" class="text-xs text-muted-foreground">
              Up to {{ formatQuantity(maxAvailable) }} available
            </small>
          </div>

          <div v-if="selectedProduct?.is_lot_tracked" class="grid gap-2">
            <Label for="trf-lot">Lot Number *</Label>
            <Select v-model="lotId">
              <SelectTrigger id="trf-lot" class="w-full">
                <SelectValue placeholder="Select lot" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="row in availableLots" :key="row.lot.id" :value="row.lot.id">
                  {{ row.lot.lot_number }} · {{ formatQuantity(row.available) }} · exp {{ row.lot.expiration_date || 'N/A' }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-4 border-t pt-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="trf-ref">Transfer Reference</Label>
            <Input id="trf-ref" v-model="reference" placeholder="e.g. TR-8801" />
          </div>

          <div class="grid gap-2">
            <Label for="trf-notes">Notes</Label>
            <Textarea id="trf-notes" v-model="notes" rows="1" placeholder="Optional transfer notes" />
          </div>
        </div>
      </form>

      <DialogFooter>
        <Button variant="outline" @click="closeDialog">Cancel</Button>
        <Button :disabled="inventoryStore.loading" @click="submitTransfer">
          <ArrowLeftRight class="size-4" />
          Execute Transfer
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
