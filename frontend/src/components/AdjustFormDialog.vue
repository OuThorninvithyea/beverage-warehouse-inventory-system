<script setup lang="ts">
import { Check } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'

import type { Product } from '@/api/catalog'
import type { AdjustInput, Lot } from '@/api/inventory'
import type { Location } from '@/api/warehouses'
import InventoryOperationSummary, { type SummaryRow } from '@/components/InventoryOperationSummary.vue'
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
const availableLots = ref<Lot[]>([])
const step = ref<'details' | 'review' | 'success'>('details')
const pendingPayload = ref<AdjustInput | null>(null)
const successRows = ref<SummaryRow[]>([])

const selectedProduct = computed(() => props.products.find((p) => p.id === productId.value))
const selectedLocation = computed(() => props.locations.find((location) => location.id === locationId.value))
const selectedLot = computed(() => availableLots.value.find((lot) => lot.id === lotId.value))
const reviewRows = computed<SummaryRow[]>(() => [
  { label: 'Product', value: `${selectedProduct.value?.name ?? '—'} (${selectedProduct.value?.sku ?? '—'})`, strong: true },
  { label: 'Location', value: selectedLocation.value?.code ?? '—', strong: true },
  { label: 'Direction', value: direction.value === 'increase' ? 'Increase stock' : 'Decrease stock', strong: true },
  { label: 'Quantity', value: `${quantity.value} ${selectedProduct.value?.unit ?? 'units'}`, strong: true },
  { label: 'Lot', value: selectedLot.value?.lot_number ?? 'No specific lot', mono: Boolean(selectedLot.value) },
  { label: 'Audit reason', value: notes.value.trim() || '—' },
])

watch(
  () => productId.value,
  async (newProdId) => {
    lotId.value = 'none'
    availableLots.value = []
    if (newProdId) {
      availableLots.value = await inventoryStore.fetchLots(newProdId)
    }
  },
)

watch(
  () => props.visible,
  (isVis) => {
    if (!isVis) return
    errorMessage.value = ''
    step.value = 'details'
    pendingPayload.value = null
    successRows.value = []
    locationId.value = props.locations[0]?.id || ''
    productId.value = ''
    lotId.value = 'none'
    direction.value = 'increase'
    quantity.value = '1'
    notes.value = ''
    availableLots.value = []
  },
)

function buildPayload(): AdjustInput | null {
  errorMessage.value = ''
  const qty = Number(quantity.value)

  if (!locationId.value) {
    errorMessage.value = 'Location is required'
    return null
  }
  if (!productId.value) {
    errorMessage.value = 'Product is required'
    return null
  }
  if (!Number.isFinite(qty) || qty <= 0) {
    errorMessage.value = 'Adjustment quantity must be greater than 0'
    return null
  }
  if (!notes.value.trim()) {
    errorMessage.value = 'Reason note is required for cycle count adjustments'
    return null
  }

  return {
    location_id: locationId.value,
    product_id: productId.value,
    direction: direction.value,
    quantity: qty.toString(),
    lot_id: lotId.value === 'none' ? null : lotId.value,
    notes: notes.value.trim(),
  }
}

function reviewAdjust() {
  const payload = buildPayload()
  if (!payload) return
  pendingPayload.value = payload
  step.value = 'review'
}

async function submitAdjust() {
  if (!pendingPayload.value) return
  errorMessage.value = ''

  try {
    const result = await inventoryStore.doAdjust(pendingPayload.value)
    successRows.value = [
      { label: 'Product', value: selectedProduct.value?.name ?? '—', strong: true },
      { label: 'Location', value: selectedLocation.value?.code ?? '—' },
      { label: 'Adjustment', value: `${pendingPayload.value.direction === 'increase' ? '+' : '−'}${pendingPayload.value.quantity} ${selectedProduct.value?.unit ?? 'units'}`, strong: true },
      { label: 'Affected lot', value: selectedLot.value?.lot_number ?? 'No specific lot', mono: Boolean(selectedLot.value) },
      { label: 'New available balance', value: result.balance.available_quantity, strong: true },
      { label: 'Audit reason', value: pendingPayload.value.notes },
      { label: 'Movement ID', value: result.movement.id, mono: true },
    ]
    emit('submitted')
    step.value = 'success'
  } catch (err: unknown) {
    errorMessage.value = err instanceof Error ? err.message : 'Stock adjustment failed'
  }
}

function closeDialog() {
  if (inventoryStore.loading) return
  emit('update:visible', false)
}

function handleOpenChange(value: boolean) {
  if (!value && inventoryStore.loading) return
  emit('update:visible', value)
}
</script>

<template>
  <Dialog :open="visible" @update:open="handleOpenChange">
    <DialogContent class="sm:max-w-xl">
      <DialogHeader>
        <DialogTitle>{{ step === 'details' ? 'Cycle Count Stock Adjustment' : step === 'review' ? 'Review Adjustment' : 'Stock Adjusted' }}</DialogTitle>
        <DialogDescription>
          {{ step === 'details' ? 'Enter the counted stock correction and audit reason.' : step === 'review' ? 'Step 2 of 2 — verify the adjustment before committing it.' : 'The balance and permanent audit record were updated.' }}
        </DialogDescription>
      </DialogHeader>

      <form v-if="step === 'details'" class="grid gap-4" @submit.prevent="reviewAdjust">
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
                <SelectItem v-for="loc in locations" :key="loc.id" :value="loc.id">
                  {{ loc.code }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="grid gap-2">
            <Label for="adj-product">Product *</Label>
            <Select v-model="productId">
              <SelectTrigger id="adj-product" class="w-full">
                <SelectValue placeholder="Select product" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="product in products" :key="product.id" :value="product.id">
                  {{ product.name }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
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

          <div class="grid gap-2">
            <Label for="adj-qty">Quantity *</Label>
            <Input id="adj-qty" v-model="quantity" type="number" min="0.001" step="0.001" placeholder="e.g. 2" required />
          </div>
        </div>

        <div v-if="selectedProduct?.is_lot_tracked" class="grid gap-2">
          <Label for="adj-lot">Lot (Optional)</Label>
          <Select v-model="lotId">
            <SelectTrigger id="adj-lot" class="w-full">
              <SelectValue placeholder="Select lot" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="none">No specific lot</SelectItem>
              <SelectItem v-for="lot in availableLots" :key="lot.id" :value="lot.id">
                {{ lot.lot_number }}
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

      <div v-else-if="step === 'review'" class="grid gap-4">
        <p v-if="errorMessage" role="alert" class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {{ errorMessage }}
        </p>
        <InventoryOperationSummary :rows="reviewRows" />
      </div>

      <InventoryOperationSummary v-else :rows="successRows" mode="success" message="The cycle-count correction was recorded successfully." />

      <DialogFooter>
        <Button v-if="step === 'details'" type="button" variant="outline" @click="closeDialog">Cancel</Button>
        <Button
          v-if="step === 'details'"
          type="button"
          :variant="direction === 'increase' ? 'default' : 'destructive'"
          @click="reviewAdjust"
        >
          Review Adjustment
        </Button>
        <Button v-if="step === 'review'" type="button" variant="outline" :disabled="inventoryStore.loading" @click="step = 'details'">Back</Button>
        <Button
          v-if="step === 'review'"
          type="button"
          :disabled="inventoryStore.loading"
          :variant="direction === 'increase' ? 'default' : 'destructive'"
          @click="submitAdjust"
        >
          <Check class="size-4" />
          {{ inventoryStore.loading ? 'Committing…' : 'Confirm Adjustment' }}
        </Button>
        <Button v-if="step === 'success'" type="button" @click="closeDialog">Done</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
