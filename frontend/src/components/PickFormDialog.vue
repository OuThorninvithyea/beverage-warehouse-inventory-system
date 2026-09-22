<script setup lang="ts">
import { Camera, Clock, Upload } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'

import type { Product } from '@/api/catalog'
import { listBalances, type Lot, type PickInput } from '@/api/inventory'
import type { Location } from '@/api/warehouses'
import BarcodeScannerModal from '@/components/BarcodeScannerModal.vue'
import InventoryOperationSummary, { type SummaryRow } from '@/components/InventoryOperationSummary.vue'
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
import { useCatalogStore } from '@/stores/catalog'
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
const catalogStore = useCatalogStore()

const locationId = ref('')
const productId = ref('')
const lotId = ref<string>('auto')
const quantity = ref<string>('1')
const reference = ref('')
const notes = ref('')

const errorMessage = ref('')
const scannerVisible = ref(false)
const availableLots = ref<Lot[]>([])
const step = ref<'details' | 'review' | 'success'>('details')
const pendingPayload = ref<PickInput | null>(null)
const successRows = ref<SummaryRow[]>([])

const selectedProduct = computed(() => props.products.find((p) => p.id === productId.value))
const selectedLocation = computed(() => props.locations.find((location) => location.id === locationId.value))
const selectedLot = computed(() => availableLots.value.find((lot) => lot.id === lotId.value))
const reviewRows = computed<SummaryRow[]>(() => [
  { label: 'Product', value: `${selectedProduct.value?.name ?? '—'} (${selectedProduct.value?.sku ?? '—'})`, strong: true },
  { label: 'Source location', value: selectedLocation.value?.code ?? '—', strong: true },
  { label: 'Pick quantity', value: `${quantity.value} ${selectedProduct.value?.unit ?? 'units'}`, strong: true },
  { label: 'Allocation', value: lotId.value === 'auto' ? 'Automatic FEFO' : selectedLot.value?.lot_number ?? '—' },
  { label: 'Reference', value: reference.value.trim() || 'Not provided' },
  { label: 'Notes', value: notes.value.trim() || 'None' },
])

watch(
  () => productId.value,
  async (newProdId) => {
    lotId.value = 'auto'
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
    lotId.value = 'auto'
    quantity.value = '1'
    reference.value = ''
    notes.value = ''
    availableLots.value = []
  },
)

async function onBarcodeScanned(code: string) {
  const found = await catalogStore.lookupBarcode(code)
  if (found) {
    productId.value = found.id
  } else {
    errorMessage.value = `No product found for barcode "${code}"`
  }
}

function buildPayload(): PickInput | null {
  errorMessage.value = ''
  const qty = Number(quantity.value)

  if (!locationId.value) {
    errorMessage.value = 'Source location is required'
    return null
  }
  if (!productId.value) {
    errorMessage.value = 'Product is required'
    return null
  }
  if (!Number.isFinite(qty) || qty <= 0) {
    errorMessage.value = 'Pick quantity must be greater than 0'
    return null
  }

  return {
    location_id: locationId.value,
    product_id: productId.value,
    quantity: qty.toString(),
    lot_id: lotId.value === 'auto' ? null : lotId.value,
    reference: reference.value.trim() || null,
    notes: notes.value.trim() || null,
  }
}

function reviewPick() {
  const payload = buildPayload()
  if (!payload) return
  pendingPayload.value = payload
  step.value = 'review'
}

async function submitPick() {
  if (!pendingPayload.value) return
  errorMessage.value = ''

  try {
    const result = await inventoryStore.doPick(pendingPayload.value)
    let remainingQuantity = 'Open Inventory to view'
    try {
      const balances = await listBalances({
        location_id: pendingPayload.value.location_id,
        product_id: pendingPayload.value.product_id,
        limit: 100,
      })
      remainingQuantity = balances.items
        .reduce((total, balance) => total + Number(balance.available_quantity), 0)
        .toFixed(3)
    } catch {
      // The pick succeeded; a follow-up balance lookup must not turn it into an error.
    }

    const affectedLots = result.movements
      .map((movement) => availableLots.value.find((lot) => lot.id === movement.lot_id)?.lot_number ?? movement.lot_id)
      .filter((lot): lot is string => Boolean(lot))

    successRows.value = [
      { label: 'Product', value: selectedProduct.value?.name ?? '—', strong: true },
      { label: 'Source location', value: selectedLocation.value?.code ?? '—' },
      { label: 'Quantity picked', value: `${result.total_quantity} ${selectedProduct.value?.unit ?? 'units'}`, strong: true },
      { label: 'Affected lots', value: affectedLots.length ? affectedLots.join(', ') : 'Non-lot inventory', mono: affectedLots.length > 0 },
      { label: 'Remaining available', value: remainingQuantity, strong: true },
      { label: 'Reference', value: result.movements[0]?.reference || 'Not provided' },
      { label: 'Movement records', value: String(result.movements.length) },
    ]
    emit('submitted')
    step.value = 'success'
  } catch (err: unknown) {
    errorMessage.value = err instanceof Error ? err.message : 'FEFO Pick failed'
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
        <DialogTitle>{{ step === 'details' ? 'FEFO Stock Pick' : step === 'review' ? 'Review FEFO Pick' : 'Stock Picked' }}</DialogTitle>
        <DialogDescription>
          {{ step === 'details' ? 'Enter outbound stock details.' : step === 'review' ? 'Step 2 of 2 — verify the FEFO allocation before confirming.' : 'The selected stock was picked and the audit trail was updated.' }}
        </DialogDescription>
      </DialogHeader>

      <form v-if="step === 'details'" class="grid gap-4" @submit.prevent="reviewPick">
        <p
          v-if="errorMessage"
          class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ errorMessage }}
        </p>

        <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="pick-location">Source Location *</Label>
            <Select v-model="locationId">
              <SelectTrigger id="pick-location" class="w-full">
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
            <Label for="pick-product">Product *</Label>
            <div class="flex gap-2">
              <Select v-model="productId">
                <SelectTrigger id="pick-product" class="flex-1">
                  <SelectValue placeholder="Select product" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="product in products" :key="product.id" :value="product.id">
                    {{ product.name }}
                  </SelectItem>
                </SelectContent>
              </Select>
              <Button type="button" variant="outline" size="icon" title="Scan barcode" @click="scannerVisible = true">
                <Camera class="size-4" />
              </Button>
            </div>
          </div>
        </div>

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
              v-for="lot in availableLots"
              :key="lot.id"
              class="flex items-center justify-between rounded border bg-background p-1.5"
            >
              <span>Lot: <strong>{{ lot.lot_number }}</strong></span>
              <span class="font-medium text-amber-700">Expires: {{ lot.expiration_date || 'N/A' }}</span>
            </div>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="pick-qty">Pick Quantity *</Label>
            <Input id="pick-qty" v-model="quantity" type="number" min="0.001" step="0.001" placeholder="e.g. 10" required />
          </div>

          <div class="grid gap-2">
            <Label for="pick-lot">Specific Lot (auto FEFO if unset)</Label>
            <Select v-model="lotId">
              <SelectTrigger id="pick-lot" class="w-full">
                <SelectValue placeholder="Auto FEFO (Recommended)" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="auto">Auto FEFO (Recommended)</SelectItem>
                <SelectItem v-for="lot in availableLots" :key="lot.id" :value="lot.id">
                  {{ lot.lot_number }}
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

      <div v-else-if="step === 'review'" class="grid gap-4">
        <p v-if="errorMessage" role="alert" class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {{ errorMessage }}
        </p>
        <InventoryOperationSummary :rows="reviewRows" />
      </div>

      <InventoryOperationSummary v-else :rows="successRows" mode="success" message="The FEFO pick was committed successfully." />

      <DialogFooter>
        <Button v-if="step === 'details'" type="button" variant="outline" @click="closeDialog">Cancel</Button>
        <Button v-if="step === 'details'" type="button" @click="reviewPick">Review Pick</Button>
        <Button v-if="step === 'review'" type="button" variant="outline" :disabled="inventoryStore.loading" @click="step = 'details'">Back</Button>
        <Button v-if="step === 'review'" type="button" :disabled="inventoryStore.loading" @click="submitPick">
          <Upload class="size-4" />
          {{ inventoryStore.loading ? 'Picking…' : 'Confirm Pick' }}
        </Button>
        <Button v-if="step === 'success'" type="button" @click="closeDialog">Done</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <BarcodeScannerModal v-model:visible="scannerVisible" @select="onBarcodeScanned" />
</template>
