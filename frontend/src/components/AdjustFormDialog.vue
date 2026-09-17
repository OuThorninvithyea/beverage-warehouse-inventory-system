<script setup lang="ts">
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputNumber from 'primevue/inputnumber'
import Message from 'primevue/message'
import Select from 'primevue/select'
import Textarea from 'primevue/textarea'
import { computed, ref, watch } from 'vue'

import type { Product } from '@/api/catalog'
import type { AdjustInput, Lot } from '@/api/inventory'
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

const locationId = ref('')
const productId = ref('')
const lotId = ref<string | null>(null)
const direction = ref<'increase' | 'decrease'>('increase')
const quantity = ref<number | null>(1)
const notes = ref('')

const errorMessage = ref('')
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
      direction.value = 'increase'
      quantity.value = 1
      notes.value = ''
      availableLots.value = []
    }
  },
)

async function submitAdjust() {
  errorMessage.value = ''
  if (!locationId.value) {
    errorMessage.value = 'Location is required'
    return
  }
  if (!productId.value) {
    errorMessage.value = 'Product is required'
    return
  }
  if (!quantity.value || quantity.value <= 0) {
    errorMessage.value = 'Adjustment quantity must be greater than 0'
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
    quantity: quantity.value.toString(),
    lot_id: lotId.value || null,
    notes: notes.value.trim(),
  }

  try {
    await inventoryStore.doAdjust(payload)
    emit('submitted')
    closeDialog()
  } catch (err: any) {
    errorMessage.value = err.message || 'Stock adjustment failed'
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
    header="Cycle Count Stock Adjustment"
    :style="{ width: '90vw', maxWidth: '580px' }"
    @update:visible="closeDialog"
  >
    <form class="grid gap-4 py-2" @submit.prevent="submitAdjust">
      <Message v-if="errorMessage" severity="error">
        {{ errorMessage }}
      </Message>

      <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
        <div class="grid gap-1">
          <label for="adj-location" class="text-xs font-semibold text-brand-muted">Location *</label>
          <Select
            id="adj-location"
            v-model="locationId"
            :options="locations"
            option-label="code"
            option-value="id"
            placeholder="Select location"
            required
          />
        </div>

        <div class="grid gap-1">
          <label for="adj-product" class="text-xs font-semibold text-brand-muted">Product *</label>
          <Select
            id="adj-product"
            v-model="productId"
            :options="products"
            option-label="name"
            option-value="id"
            placeholder="Select product"
            class="w-full"
            filter
          />
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
        <div class="grid gap-1">
          <label for="adj-direction" class="text-xs font-semibold text-brand-muted">Adjustment Direction *</label>
          <Select
            id="adj-direction"
            v-model="direction"
            :options="[
              { label: 'Increase (+ Stock Found)', value: 'increase' },
              { label: 'Decrease (- Stock Missing/Damaged)', value: 'decrease' },
            ]"
            option-label="label"
            option-value="value"
          />
        </div>

        <div class="grid gap-1">
          <label for="adj-qty" class="text-xs font-semibold text-brand-muted">Quantity *</label>
          <InputNumber
            id="adj-qty"
            v-model="quantity"
            :min="0.001"
            :min-fraction-digits="0"
            :max-fraction-digits="3"
            placeholder="e.g. 2"
            required
          />
        </div>
      </div>

      <div v-if="selectedProduct?.is_lot_tracked" class="grid gap-1">
        <label for="adj-lot" class="text-xs font-semibold text-brand-muted">Lot (Optional)</label>
        <Select
          id="adj-lot"
          v-model="lotId"
          :options="availableLots"
          option-label="lot_number"
          option-value="id"
          placeholder="Select lot"
          show-clear
        />
      </div>

      <div class="grid gap-1 border-t border-brand-border pt-3">
        <label for="adj-notes" class="text-xs font-semibold text-brand-muted">Mandatory Audit Reason Note *</label>
        <Textarea
          id="adj-notes"
          v-model="notes"
          rows="2"
          placeholder="e.g. Damaged during forklift transit / Periodic physical count correction"
          required
        />
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button label="Cancel" severity="secondary" outlined @click="closeDialog" />
        <Button
          label="Commit Adjustment"
          icon="pi pi-check"
          :severity="direction === 'increase' ? 'success' : 'danger'"
          :loading="inventoryStore.loading"
          @click="submitAdjust"
        />
      </div>
    </template>
  </Dialog>
</template>
