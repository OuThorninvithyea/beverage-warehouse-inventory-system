<script setup lang="ts">
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputNumber from 'primevue/inputnumber'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'
import Select from 'primevue/select'
import Textarea from 'primevue/textarea'
import { computed, ref, watch } from 'vue'

import type { Product } from '@/api/catalog'
import type { Lot, PickInput } from '@/api/inventory'
import type { Location } from '@/api/warehouses'
import BarcodeScannerModal from '@/components/BarcodeScannerModal.vue'
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
const lotId = ref<string | null>(null)
const quantity = ref<number | null>(1)
const reference = ref('')
const notes = ref('')

const errorMessage = ref('')
const scannerVisible = ref(false)
const availableLots = ref<Lot[]>([])

const selectedProduct = computed(() => {
  return props.products.find((p) => p.id === productId.value)
})

watch(
  () => productId.value,
  async (newProdId) => {
    lotId.value = null
    availableLots.value = []
    if (newProdId) {
      availableLots.value = await inventoryStore.fetchLots(newProdId)
    }
  },
)

watch(
  () => props.visible,
  (isVis) => {
    if (isVis) {
      errorMessage.value = ''
      locationId.value = props.locations[0]?.id || ''
      productId.value = ''
      lotId.value = null
      quantity.value = 1
      reference.value = ''
      notes.value = ''
      availableLots.value = []
    }
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

async function submitPick() {
  errorMessage.value = ''
  if (!locationId.value) {
    errorMessage.value = 'Source location is required'
    return
  }
  if (!productId.value) {
    errorMessage.value = 'Product is required'
    return
  }
  if (!quantity.value || quantity.value <= 0) {
    errorMessage.value = 'Pick quantity must be greater than 0'
    return
  }

  const payload: PickInput = {
    location_id: locationId.value,
    product_id: productId.value,
    quantity: quantity.value.toString(),
    lot_id: lotId.value || null,
    reference: reference.value.trim() || null,
    notes: notes.value.trim() || null,
  }

  try {
    await inventoryStore.doPick(payload)
    emit('submitted')
    closeDialog()
  } catch (err: any) {
    errorMessage.value = err.message || 'FEFO Pick failed'
  }
}

function closeDialog() {
  emit('update:visible', false)
}
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    header="FEFO Stock Pick"
    :style="{ width: '90vw', maxWidth: '620px' }"
    @update:visible="closeDialog"
  >
    <form class="grid gap-4 py-2" @submit.prevent="submitPick">
      <Message v-if="errorMessage" severity="error">
        {{ errorMessage }}
      </Message>

      <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
        <div class="grid gap-1">
          <label for="pick-location" class="text-xs font-semibold text-brand-muted">Source Location *</label>
          <Select
            id="pick-location"
            v-model="locationId"
            :options="locations"
            option-label="code"
            option-value="id"
            placeholder="Select location"
            required
          />
        </div>

        <div class="grid gap-1">
          <label for="pick-product" class="text-xs font-semibold text-brand-muted">Product *</label>
          <div class="flex gap-2">
            <Select
              id="pick-product"
              v-model="productId"
              :options="products"
              option-label="name"
              option-value="id"
              placeholder="Select product"
              class="w-full"
              filter
            />
            <Button
              type="button"
              icon="pi pi-camera"
              severity="secondary"
              title="Scan barcode"
              @click="scannerVisible = true"
            />
          </div>
        </div>
      </div>

      <div v-if="selectedProduct" class="rounded-[8px] bg-brand-surface p-3 text-xs flex justify-between items-center">
        <div>
          <span class="font-bold text-brand-navy">{{ selectedProduct.name }}</span>
          <div class="text-brand-muted">SKU: {{ selectedProduct.sku }} | Unit: {{ selectedProduct.unit }}</div>
        </div>
        <span
          v-if="selectedProduct.is_lot_tracked"
          class="rounded-full bg-blue-100 px-2 py-0.5 text-[10px] font-bold text-blue-800"
        >
          FEFO Auto Allocation Active
        </span>
      </div>

      <div v-if="availableLots.length > 0" class="rounded-[8px] border border-blue-200 bg-blue-50/50 p-3 text-xs">
        <div class="font-bold text-blue-900 mb-1 flex items-center gap-1">
          <i class="pi pi-clock text-blue-600" />
          FEFO Lots (Earliest Expiration First)
        </div>
        <div class="grid gap-1 max-h-[100px] overflow-y-auto pr-1">
          <div
            v-for="l in availableLots"
            :key="l.id"
            class="flex justify-between items-center p-1 rounded bg-white border border-blue-100"
          >
            <span>Lot: <strong>{{ l.lot_number }}</strong></span>
            <span class="font-semibold text-amber-700">Expires: {{ l.expiration_date || 'N/A' }}</span>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
        <div class="grid gap-1">
          <label for="pick-qty" class="text-xs font-semibold text-brand-muted">Pick Quantity *</label>
          <InputNumber
            id="pick-qty"
            v-model="quantity"
            :min="0.001"
            :min-fraction-digits="0"
            :max-fraction-digits="3"
            placeholder="e.g. 10"
            required
          />
        </div>

        <div class="grid gap-1">
          <label for="pick-lot" class="text-xs font-semibold text-brand-muted">Specific Lot (Optional - Auto FEFO if blank)</label>
          <Select
            id="pick-lot"
            v-model="lotId"
            :options="availableLots"
            option-label="lot_number"
            option-value="id"
            placeholder="Auto FEFO (Recommended)"
            show-clear
          />
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1 border-t border-brand-border pt-3">
        <div class="grid gap-1">
          <label for="pick-ref" class="text-xs font-semibold text-brand-muted">Sales Order / Pick Ref</label>
          <InputText
            id="pick-ref"
            v-model="reference"
            placeholder="e.g. SO-2201"
          />
        </div>

        <div class="grid gap-1">
          <label for="pick-notes" class="text-xs font-semibold text-brand-muted">Notes</label>
          <Textarea
            id="pick-notes"
            v-model="notes"
            rows="1"
            placeholder="Optional picking notes"
          />
        </div>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button label="Cancel" severity="secondary" outlined @click="closeDialog" />
        <Button
          label="Execute Pick"
          icon="pi pi-upload"
          severity="warn"
          :loading="inventoryStore.loading"
          @click="submitPick"
        />
      </div>
    </template>
  </Dialog>

  <BarcodeScannerModal
    v-model:visible="scannerVisible"
    @select="onBarcodeScanned"
  />
</template>
