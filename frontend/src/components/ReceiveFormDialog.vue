<script setup lang="ts">
import Button from 'primevue/button'
import DatePicker from 'primevue/datepicker'
import Dialog from 'primevue/dialog'
import InputNumber from 'primevue/inputnumber'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'
import Select from 'primevue/select'
import Textarea from 'primevue/textarea'
import { computed, ref, watch } from 'vue'

import type { Product } from '@/api/catalog'
import type { ReceiveInput } from '@/api/inventory'
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
const quantity = ref<number | null>(1)
const unitCost = ref<number | null>(0)
const lotNumber = ref('')
const expirationDate = ref<Date | null>(null)
const reference = ref('')
const notes = ref('')

const errorMessage = ref('')
const scannerVisible = ref(false)

const selectedProduct = computed(() => {
  return props.products.find((p) => p.id === productId.value)
})

watch(
  () => props.visible,
  (isVis) => {
    if (isVis) {
      errorMessage.value = ''
      locationId.value = props.locations[0]?.id || ''
      productId.value = ''
      quantity.value = 1
      unitCost.value = 0
      lotNumber.value = ''
      expirationDate.value = null
      reference.value = ''
      notes.value = ''
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

async function submitReceive() {
  errorMessage.value = ''
  if (!locationId.value) {
    errorMessage.value = 'Target location is required'
    return
  }
  if (!productId.value) {
    errorMessage.value = 'Product is required'
    return
  }
  if (!quantity.value || quantity.value <= 0) {
    errorMessage.value = 'Quantity must be greater than 0'
    return
  }
  if (unitCost.value === null || unitCost.value < 0) {
    errorMessage.value = 'Unit cost must be 0 or greater'
    return
  }
  if (selectedProduct.value?.is_lot_tracked && !lotNumber.value.trim()) {
    errorMessage.value = 'Lot number is required for lot-tracked products'
    return
  }

  let expDateStr: string | null = null
  if (expirationDate.value) {
    expDateStr = expirationDate.value.toISOString().split('T')[0]
  }

  const payload: ReceiveInput = {
    location_id: locationId.value,
    product_id: productId.value,
    quantity: quantity.value.toString(),
    unit_cost: unitCost.value.toString(),
    lot_number: lotNumber.value.trim() || null,
    expiration_date: expDateStr,
    reference: reference.value.trim() || null,
    notes: notes.value.trim() || null,
  }

  try {
    await inventoryStore.doReceive(payload)
    emit('submitted')
    closeDialog()
  } catch (err: any) {
    errorMessage.value = err.message || 'Failed to receive stock'
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
    header="Receive Inventory"
    :style="{ width: '90vw', maxWidth: '620px' }"
    @update:visible="closeDialog"
  >
    <form class="grid gap-4 py-2" @submit.prevent="submitReceive">
      <Message v-if="errorMessage" severity="error">
        {{ errorMessage }}
      </Message>

      <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
        <div class="grid gap-1">
          <label for="rcv-location" class="text-xs font-semibold text-brand-muted">Target Location *</label>
          <Select
            id="rcv-location"
            v-model="locationId"
            :options="locations"
            option-label="code"
            option-value="id"
            placeholder="Select location"
            required
          />
        </div>

        <div class="grid gap-1">
          <label for="rcv-product" class="text-xs font-semibold text-brand-muted">Product *</label>
          <div class="flex gap-2">
            <Select
              id="rcv-product"
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
          class="rounded-full bg-amber-100 px-2 py-0.5 text-[10px] font-bold text-amber-800"
        >
          Lot Tracked
        </span>
      </div>

      <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
        <div class="grid gap-1">
          <label for="rcv-qty" class="text-xs font-semibold text-brand-muted">Quantity *</label>
          <InputNumber
            id="rcv-qty"
            v-model="quantity"
            :min="0.001"
            :min-fraction-digits="0"
            :max-fraction-digits="3"
            placeholder="e.g. 50"
            required
          />
        </div>

        <div class="grid gap-1">
          <label for="rcv-cost" class="text-xs font-semibold text-brand-muted">Unit Cost ($) *</label>
          <InputNumber
            id="rcv-cost"
            v-model="unitCost"
            mode="currency"
            currency="USD"
            locale="en-US"
            :min="0"
            :min-fraction-digits="2"
            placeholder="$0.00"
            required
          />
        </div>
      </div>

      <div v-if="selectedProduct?.is_lot_tracked" class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1 border-t border-brand-border pt-3">
        <div class="grid gap-1">
          <label for="rcv-lot" class="text-xs font-semibold text-brand-muted">Lot Number *</label>
          <InputText
            id="rcv-lot"
            v-model="lotNumber"
            placeholder="e.g. LOT-2026-08-A"
            required
          />
        </div>

        <div class="grid gap-1">
          <label for="rcv-exp" class="text-xs font-semibold text-brand-muted">Expiration Date</label>
          <DatePicker
            id="rcv-exp"
            v-model="expirationDate"
            date-format="yy-mm-dd"
            placeholder="YYYY-MM-DD"
            show-icon
          />
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1 border-t border-brand-border pt-3">
        <div class="grid gap-1">
          <label for="rcv-ref" class="text-xs font-semibold text-brand-muted">Reference (PO / Invoice)</label>
          <InputText
            id="rcv-ref"
            v-model="reference"
            placeholder="e.g. PO-10042"
          />
        </div>

        <div class="grid gap-1">
          <label for="rcv-notes" class="text-xs font-semibold text-brand-muted">Notes</label>
          <Textarea
            id="rcv-notes"
            v-model="notes"
            rows="1"
            placeholder="Optional receiving notes"
          />
        </div>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button label="Cancel" severity="secondary" outlined @click="closeDialog" />
        <Button
          label="Receive Stock"
          icon="pi pi-download"
          :loading="inventoryStore.loading"
          @click="submitReceive"
        />
      </div>
    </template>
  </Dialog>

  <BarcodeScannerModal
    v-model:visible="scannerVisible"
    @select="onBarcodeScanned"
  />
</template>
