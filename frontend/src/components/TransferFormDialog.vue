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
import type { Lot, TransferInput } from '@/api/inventory'
import type { Location } from '@/api/warehouses'
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
const lotId = ref<string | null>(null)
const quantity = ref<number | null>(1)
const reference = ref('')
const notes = ref('')

const errorMessage = ref('')
const availableLots = ref<Lot[]>([])

const selectedProduct = computed(() => {
  return props.products.find((p) => p.id === productId.value)
})

const destinationLocations = computed(() => {
  return props.locations.filter((loc) => loc.id !== fromLocationId.value)
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
      productId.value = ''
      fromLocationId.value = props.locations[0]?.id || ''
      toLocationId.value = props.locations[1]?.id || ''
      lotId.value = null
      quantity.value = 1
      reference.value = ''
      notes.value = ''
      availableLots.value = []
    }
  },
)

async function submitTransfer() {
  errorMessage.value = ''
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
  if (!quantity.value || quantity.value <= 0) {
    errorMessage.value = 'Transfer quantity must be greater than 0'
    return
  }
  if (selectedProduct.value?.is_lot_tracked && !lotId.value) {
    errorMessage.value = 'Lot selection is required for lot-tracked products'
    return
  }

  const payload: TransferInput = {
    product_id: productId.value,
    from_location_id: fromLocationId.value,
    to_location_id: toLocationId.value,
    quantity: quantity.value.toString(),
    lot_id: lotId.value || null,
    reference: reference.value.trim() || null,
    notes: notes.value.trim() || null,
  }

  try {
    await inventoryStore.doTransfer(payload)
    emit('submitted')
    closeDialog()
  } catch (err: any) {
    errorMessage.value = err.message || 'Stock transfer failed'
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
    header="Transfer Stock Between Locations"
    :style="{ width: '90vw', maxWidth: '620px' }"
    @update:visible="closeDialog"
  >
    <form class="grid gap-4 py-2" @submit.prevent="submitTransfer">
      <Message v-if="errorMessage" severity="error">
        {{ errorMessage }}
      </Message>

      <div class="grid gap-1">
        <label for="trf-product" class="text-xs font-semibold text-brand-muted">Product *</label>
        <Select
          id="trf-product"
          v-model="productId"
          :options="products"
          option-label="name"
          option-value="id"
          placeholder="Select product to transfer"
          class="w-full"
          filter
        />
      </div>

      <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
        <div class="grid gap-1">
          <label for="trf-from" class="text-xs font-semibold text-brand-muted">From Location (Source) *</label>
          <Select
            id="trf-from"
            v-model="fromLocationId"
            :options="locations"
            option-label="code"
            option-value="id"
            placeholder="Select source location"
            required
          />
        </div>

        <div class="grid gap-1">
          <label for="trf-to" class="text-xs font-semibold text-brand-muted">To Location (Destination) *</label>
          <Select
            id="trf-to"
            v-model="toLocationId"
            :options="destinationLocations"
            option-label="code"
            option-value="id"
            placeholder="Select destination location"
            required
          />
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
        <div class="grid gap-1">
          <label for="trf-qty" class="text-xs font-semibold text-brand-muted">Transfer Quantity *</label>
          <InputNumber
            id="trf-qty"
            v-model="quantity"
            :min="0.001"
            :min-fraction-digits="0"
            :max-fraction-digits="3"
            placeholder="e.g. 5"
            required
          />
        </div>

        <div v-if="selectedProduct?.is_lot_tracked" class="grid gap-1">
          <label for="trf-lot" class="text-xs font-semibold text-brand-muted">Lot Number *</label>
          <Select
            id="trf-lot"
            v-model="lotId"
            :options="availableLots"
            option-label="lot_number"
            option-value="id"
            placeholder="Select lot"
            required
          />
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1 border-t border-brand-border pt-3">
        <div class="grid gap-1">
          <label for="trf-ref" class="text-xs font-semibold text-brand-muted">Transfer Reference</label>
          <InputText
            id="trf-ref"
            v-model="reference"
            placeholder="e.g. TR-8801"
          />
        </div>

        <div class="grid gap-1">
          <label for="trf-notes" class="text-xs font-semibold text-brand-muted">Notes</label>
          <Textarea
            id="trf-notes"
            v-model="notes"
            rows="1"
            placeholder="Optional transfer notes"
          />
        </div>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button label="Cancel" severity="secondary" outlined @click="closeDialog" />
        <Button
          label="Execute Transfer"
          icon="pi pi-arrows-h"
          :loading="inventoryStore.loading"
          @click="submitTransfer"
        />
      </div>
    </template>
  </Dialog>
</template>
