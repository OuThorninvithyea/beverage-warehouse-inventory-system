<script setup lang="ts">
import { ArrowLeftRight } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'

import type { Product } from '@/api/catalog'
import type { Lot, TransferInput } from '@/api/inventory'
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

const productId = ref('')
const fromLocationId = ref('')
const toLocationId = ref('')
const lotId = ref<string>('')
const quantity = ref<string>('1')
const reference = ref('')
const notes = ref('')

const errorMessage = ref('')
const availableLots = ref<Lot[]>([])
const step = ref<'details' | 'review' | 'success'>('details')
const pendingPayload = ref<TransferInput | null>(null)
const successRows = ref<SummaryRow[]>([])

const selectedProduct = computed(() => props.products.find((p) => p.id === productId.value))
const sourceLocation = computed(() => props.locations.find((location) => location.id === fromLocationId.value))
const destinationLocation = computed(() => props.locations.find((location) => location.id === toLocationId.value))
const selectedLot = computed(() => availableLots.value.find((lot) => lot.id === lotId.value))
const reviewRows = computed<SummaryRow[]>(() => [
  { label: 'Product', value: `${selectedProduct.value?.name ?? '—'} (${selectedProduct.value?.sku ?? '—'})`, strong: true },
  { label: 'From location', value: sourceLocation.value?.code ?? '—', strong: true },
  { label: 'To location', value: destinationLocation.value?.code ?? '—', strong: true },
  { label: 'Quantity', value: `${quantity.value} ${selectedProduct.value?.unit ?? 'units'}`, strong: true },
  { label: 'Lot', value: selectedLot.value?.lot_number ?? 'Non-lot inventory', mono: Boolean(selectedLot.value) },
  { label: 'Reference', value: reference.value.trim() || 'Not provided' },
  { label: 'Notes', value: notes.value.trim() || 'None' },
])

const destinationLocations = computed(() =>
  props.locations.filter((loc) => loc.id !== fromLocationId.value),
)

watch(
  () => productId.value,
  async (newProdId) => {
    lotId.value = ''
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
    productId.value = ''
    fromLocationId.value = props.locations[0]?.id || ''
    toLocationId.value = props.locations[1]?.id || ''
    lotId.value = ''
    quantity.value = '1'
    reference.value = ''
    notes.value = ''
    availableLots.value = []
  },
)

function buildPayload(): TransferInput | null {
  errorMessage.value = ''
  const qty = Number(quantity.value)

  if (!productId.value) {
    errorMessage.value = 'Product is required'
    return null
  }
  if (!fromLocationId.value) {
    errorMessage.value = 'Source location is required'
    return null
  }
  if (!toLocationId.value) {
    errorMessage.value = 'Destination location is required'
    return null
  }
  if (fromLocationId.value === toLocationId.value) {
    errorMessage.value = 'Source and Destination locations must be different'
    return null
  }
  if (!Number.isFinite(qty) || qty <= 0) {
    errorMessage.value = 'Transfer quantity must be greater than 0'
    return null
  }
  if (selectedProduct.value?.is_lot_tracked && !lotId.value) {
    errorMessage.value = 'Lot selection is required for lot-tracked products'
    return null
  }

  return {
    product_id: productId.value,
    from_location_id: fromLocationId.value,
    to_location_id: toLocationId.value,
    quantity: qty.toString(),
    lot_id: lotId.value || null,
    reference: reference.value.trim() || null,
    notes: notes.value.trim() || null,
  }
}

function reviewTransfer() {
  const payload = buildPayload()
  if (!payload) return
  pendingPayload.value = payload
  step.value = 'review'
}

async function submitTransfer() {
  if (!pendingPayload.value) return
  errorMessage.value = ''

  try {
    const result = await inventoryStore.doTransfer(pendingPayload.value)
    successRows.value = [
      { label: 'Product', value: selectedProduct.value?.name ?? '—', strong: true },
      { label: 'Route', value: `${sourceLocation.value?.code ?? '—'} → ${destinationLocation.value?.code ?? '—'}`, strong: true },
      { label: 'Quantity transferred', value: `${pendingPayload.value.quantity} ${selectedProduct.value?.unit ?? 'units'}`, strong: true },
      { label: 'Affected lot', value: selectedLot.value?.lot_number ?? 'Non-lot inventory', mono: Boolean(selectedLot.value) },
      { label: 'Source balance', value: result.source_balance.available_quantity },
      { label: 'Destination balance', value: result.destination_balance.available_quantity },
      { label: 'Reference', value: result.movement.reference || 'Not provided' },
      { label: 'Movement ID', value: result.movement.id, mono: true },
    ]
    emit('submitted')
    step.value = 'success'
  } catch (err: unknown) {
    errorMessage.value = err instanceof Error ? err.message : 'Stock transfer failed'
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
    <DialogContent class="sm:max-w-2xl">
      <DialogHeader>
        <DialogTitle>{{ step === 'details' ? 'Transfer Stock Between Locations' : step === 'review' ? 'Review Transfer' : 'Stock Transferred' }}</DialogTitle>
        <DialogDescription>
          {{ step === 'details' ? 'Enter the stock movement details.' : step === 'review' ? 'Step 2 of 2 — verify both locations before confirming.' : 'Both location balances and the audit trail were updated.' }}
        </DialogDescription>
      </DialogHeader>

      <form v-if="step === 'details'" class="grid gap-4" @submit.prevent="reviewTransfer">
        <p
          v-if="errorMessage"
          class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ errorMessage }}
        </p>

        <div class="grid gap-2">
          <Label for="trf-product">Product *</Label>
          <Select v-model="productId">
            <SelectTrigger id="trf-product" class="w-full">
              <SelectValue placeholder="Select product to transfer" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="product in products" :key="product.id" :value="product.id">
                {{ product.name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="trf-from">From Location (Source) *</Label>
            <Select v-model="fromLocationId">
              <SelectTrigger id="trf-from" class="w-full">
                <SelectValue placeholder="Select source location" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="loc in locations" :key="loc.id" :value="loc.id">
                  {{ loc.code }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="grid gap-2">
            <Label for="trf-to">To Location (Destination) *</Label>
            <Select v-model="toLocationId">
              <SelectTrigger id="trf-to" class="w-full">
                <SelectValue placeholder="Select destination location" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="loc in destinationLocations" :key="loc.id" :value="loc.id">
                  {{ loc.code }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="trf-qty">Transfer Quantity *</Label>
            <Input id="trf-qty" v-model="quantity" type="number" min="0.001" step="0.001" placeholder="e.g. 5" required />
          </div>

          <div v-if="selectedProduct?.is_lot_tracked" class="grid gap-2">
            <Label for="trf-lot">Lot Number *</Label>
            <Select v-model="lotId">
              <SelectTrigger id="trf-lot" class="w-full">
                <SelectValue placeholder="Select lot" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="lot in availableLots" :key="lot.id" :value="lot.id">
                  {{ lot.lot_number }}
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

      <div v-else-if="step === 'review'" class="grid gap-4">
        <p v-if="errorMessage" role="alert" class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {{ errorMessage }}
        </p>
        <InventoryOperationSummary :rows="reviewRows" />
      </div>

      <InventoryOperationSummary v-else :rows="successRows" mode="success" message="Stock is now available at the destination location." />

      <DialogFooter>
        <Button v-if="step === 'details'" type="button" variant="outline" @click="closeDialog">Cancel</Button>
        <Button v-if="step === 'details'" type="button" @click="reviewTransfer">Review Transfer</Button>
        <Button v-if="step === 'review'" type="button" variant="outline" :disabled="inventoryStore.loading" @click="step = 'details'">Back</Button>
        <Button v-if="step === 'review'" type="button" :disabled="inventoryStore.loading" @click="submitTransfer">
          <ArrowLeftRight class="size-4" />
          {{ inventoryStore.loading ? 'Transferring…' : 'Confirm Transfer' }}
        </Button>
        <Button v-if="step === 'success'" type="button" @click="closeDialog">Done</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
