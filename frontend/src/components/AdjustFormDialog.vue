<script setup lang="ts">
import { Check } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'

import type { Product } from '@/api/catalog'
import { listLots, type AdjustInput } from '@/api/inventory'
import type { Location } from '@/api/warehouses'
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
import { formatQuantity, useLocationStock, type StockedLot } from '@/lib/location-stock'
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
const lotId = ref<string>('none')
const direction = ref<'increase' | 'decrease'>('increase')
const quantity = ref<string>('1')
const notes = ref('')

const errorMessage = ref('')
const availableLots = ref<StockedLot[]>([])

// Decreasing removes stock that must already be on this shelf, so it offers
// only what the location holds. Increasing records stock that was found, so
// any active product qualifies, as with a receive.
const stock = useLocationStock(locationId)
const activeLocations = computed(() => props.locations.filter((loc) => loc.is_active))
const isDecrease = computed(() => direction.value === 'decrease')
const locationCode = computed(
  () => props.locations.find((loc) => loc.id === locationId.value)?.code ?? 'this location',
)

const productOptions = computed(() =>
  isDecrease.value
    ? stock.stockedProducts.value
    : stock.allProducts.value.map((product) => ({ product, available: stock.availableFor(product.id) })),
)
const selectedProduct = computed<Product | undefined>(
  () => productOptions.value.find((row) => row.product.id === productId.value)?.product,
)
const maxAvailable = computed(() =>
  stock.availableFor(productId.value, lotId.value === 'none' ? null : lotId.value),
)

async function loadLots() {
  lotId.value = 'none'
  availableLots.value = []
  const id = productId.value
  if (!id) return
  availableLots.value = isDecrease.value
    ? await stock.lotsFor(id)
    : (await listLots(id)).map((lot) => ({ lot, available: stock.availableFor(id, lot.id) }))
}

// Switching to a decrease, or to another shelf, can invalidate the product.
watch([locationId, direction], () => {
  if (isDecrease.value && stock.availableFor(productId.value) <= 0) productId.value = ''
  void loadLots()
})
watch(() => productId.value, () => void loadLots())

watch(
  () => props.visible,
  async (isVis) => {
    if (!isVis) return
    errorMessage.value = ''
    productId.value = ''
    lotId.value = 'none'
    direction.value = 'increase'
    quantity.value = '1'
    notes.value = ''
    availableLots.value = []
    await stock.loadCatalog()
    const first = activeLocations.value[0]?.id || ''
    if (locationId.value === first) await stock.reload()
    else locationId.value = first
  },
)

async function submitAdjust() {
  errorMessage.value = ''
  const qty = Number(quantity.value)

  if (!locationId.value) {
    errorMessage.value = 'Location is required'
    return
  }
  if (!productId.value) {
    errorMessage.value = 'Product is required'
    return
  }
  if (!Number.isFinite(qty) || qty <= 0) {
    errorMessage.value = 'Adjustment quantity must be greater than 0'
    return
  }
  if (isDecrease.value && qty > maxAvailable.value) {
    errorMessage.value = `Only ${formatQuantity(maxAvailable.value)} available in ${locationCode.value}`
    return
  }
  if (!notes.value.trim()) {
    errorMessage.value = 'Reason note is required for cycle count adjustments'
    return
  }

  const payload: AdjustInput = {
    location_id: locationId.value,
    product_id: productId.value,
    direction: direction.value,
    quantity: qty.toString(),
    lot_id: lotId.value === 'none' ? null : lotId.value,
    notes: notes.value.trim(),
  }

  try {
    await inventoryStore.doAdjust(payload)
    emit('submitted')
    closeDialog()
  } catch (err: unknown) {
    errorMessage.value = err instanceof Error ? err.message : 'Stock adjustment failed'
  }
}

function closeDialog() {
  emit('update:visible', false)
}
</script>

<template>
  <Dialog :open="visible" @update:open="(value: boolean) => emit('update:visible', value)">
    <DialogContent class="sm:max-w-xl">
      <DialogHeader>
        <DialogTitle>Cycle Count Stock Adjustment</DialogTitle>
        <DialogDescription>Correct counted stock with a mandatory audit reason.</DialogDescription>
      </DialogHeader>

      <form class="grid gap-4" @submit.prevent="submitAdjust">
        <p
          v-if="errorMessage"
          class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ errorMessage }}
        </p>

        <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="adj-location">Location *</Label>
            <Select v-model="locationId">
              <SelectTrigger id="adj-location" class="w-full">
                <SelectValue placeholder="Select location" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="loc in activeLocations" :key="loc.id" :value="loc.id">
                  {{ loc.code }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="grid gap-2">
            <Label for="adj-direction">Adjustment Direction *</Label>
            <Select v-model="direction">
              <SelectTrigger id="adj-direction" class="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="increase">Increase (+ Stock Found)</SelectItem>
                <SelectItem value="decrease">Decrease (- Stock Missing/Damaged)</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="adj-product">Product *</Label>
            <Select v-model="productId" :disabled="stock.loading.value || productOptions.length === 0">
              <SelectTrigger id="adj-product" class="w-full">
                <SelectValue
                  :placeholder="stock.loading.value ? 'Loading stock…' : productOptions.length === 0 ? 'No stock here to decrease' : 'Select product'"
                />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="row in productOptions" :key="row.product.id" :value="row.product.id">
                  {{ row.product.name }}<template v-if="row.available > 0"> · {{ formatQuantity(row.available) }} here</template>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="grid gap-2">
            <Label for="adj-qty">Quantity *</Label>
            <Input
              id="adj-qty"
              v-model="quantity"
              type="number"
              min="0.001"
              :max="isDecrease ? maxAvailable || undefined : undefined"
              step="0.001"
              placeholder="e.g. 2"
              required
            />
            <small v-if="isDecrease && productId" class="text-xs text-muted-foreground">
              Up to {{ formatQuantity(maxAvailable) }} can be removed
            </small>
          </div>
        </div>

        <p
          v-if="isDecrease && locationId && !stock.loading.value && productOptions.length === 0"
          class="rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-sm text-amber-700"
        >
          {{ locationCode }} has no available stock to decrease. Choose another location.
        </p>

        <div v-if="selectedProduct?.is_lot_tracked" class="grid gap-2">
          <Label for="adj-lot">Lot (Optional)</Label>
          <Select v-model="lotId">
            <SelectTrigger id="adj-lot" class="w-full">
              <SelectValue placeholder="Select lot" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="none">No specific lot</SelectItem>
              <SelectItem v-for="row in availableLots" :key="row.lot.id" :value="row.lot.id">
                {{ row.lot.lot_number }}<template v-if="row.available > 0"> · {{ formatQuantity(row.available) }} here</template>
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div class="grid gap-2 border-t pt-4">
          <Label for="adj-notes">Mandatory Audit Reason Note *</Label>
          <Textarea
            id="adj-notes"
            v-model="notes"
            rows="2"
            placeholder="e.g. Damaged during forklift transit / Periodic physical count correction"
            required
          />
        </div>
      </form>

      <DialogFooter>
        <Button variant="outline" @click="closeDialog">Cancel</Button>
        <Button
          :disabled="inventoryStore.loading"
          :variant="direction === 'increase' ? 'default' : 'destructive'"
          @click="submitAdjust"
        >
          <Check class="size-4" />
          Commit Adjustment
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
