<script setup lang="ts">
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'
import Select from 'primevue/select'
import ToggleSwitch from 'primevue/toggleswitch'
import { computed, ref, watch } from 'vue'

import type { Category, Product, ProductInput } from '@/api/catalog'
import BarcodeScannerModal from '@/components/BarcodeScannerModal.vue'
import { validateBarcode } from '@/lib/barcode'
import { useCatalogStore } from '@/stores/catalog'

const props = defineProps<{
  visible: boolean
  product?: Product | null
  categories: Category[]
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'saved', product: Product): void
}>()

const catalogStore = useCatalogStore()

const sku = ref('')
const name = ref('')
const unit = ref('case')
const barcode = ref('')
const categoryId = ref<string | null>(null)
const isLotTracked = ref(true)
const isActive = ref(true)

const errorMessage = ref('')
const scannerVisible = ref(false)

const isEditMode = computed(() => Boolean(props.product?.id))
const barcodeValidation = computed(() => {
  if (!barcode.value.trim()) return null
  return validateBarcode(barcode.value)
})

watch(
  () => props.visible,
  (isVis) => {
    if (isVis) {
      errorMessage.value = ''
      if (props.product) {
        sku.value = props.product.sku
        name.value = props.product.name
        unit.value = props.product.unit
        barcode.value = props.product.barcode || ''
        categoryId.value = props.product.category_id
        isLotTracked.value = props.product.is_lot_tracked
        isActive.value = props.product.is_active
      } else {
        sku.value = ''
        name.value = ''
        unit.value = 'case'
        barcode.value = ''
        categoryId.value = null
        isLotTracked.value = true
        isActive.value = true
      }
    }
  },
)

function onBarcodeScanned(scannedCode: string) {
  barcode.value = scannedCode
}

async function saveProduct() {
  errorMessage.value = ''
  if (!sku.value.trim()) {
    errorMessage.value = 'SKU is required'
    return
  }
  if (!name.value.trim()) {
    errorMessage.value = 'Product name is required'
    return
  }
  if (!unit.value.trim()) {
    errorMessage.value = 'Unit of measure is required'
    return
  }
  if (barcode.value.trim() && barcodeValidation.value && !barcodeValidation.value.valid) {
    errorMessage.value = barcodeValidation.value.message
    return
  }

  const payload: ProductInput = {
    sku: sku.value.trim().toUpperCase(),
    name: name.value.trim(),
    unit: unit.value.trim().toLowerCase(),
    barcode: barcode.value.trim() || null,
    category_id: categoryId.value || null,
    is_lot_tracked: isLotTracked.value,
    is_active: isActive.value,
  }

  try {
    let saved: Product
    if (isEditMode.value && props.product?.id) {
      saved = await catalogStore.editProduct(props.product.id, payload)
    } else {
      saved = await catalogStore.addProduct(payload)
    }
    emit('saved', saved)
    closeDialog()
  } catch (err: any) {
    errorMessage.value = err.message || 'Failed to save product'
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
    :header="isEditMode ? 'Edit Product' : 'Add New Product'"
    :style="{ width: '90vw', maxWidth: '580px' }"
    @update:visible="closeDialog"
  >
    <form class="grid gap-4 py-2" @submit.prevent="saveProduct">
      <Message v-if="errorMessage" severity="error">
        {{ errorMessage }}
      </Message>

      <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
        <div class="grid gap-1">
          <label for="sku" class="text-xs font-semibold text-brand-muted">SKU *</label>
          <InputText
            id="sku"
            v-model="sku"
            placeholder="e.g. COKE-330-CAN"
            required
            class="uppercase"
          />
        </div>

        <div class="grid gap-1">
          <label for="unit" class="text-xs font-semibold text-brand-muted">Unit *</label>
          <Select
            id="unit"
            v-model="unit"
            :options="['case', 'bottle', 'can', 'pack', 'pallet', 'keg']"
            placeholder="Select unit"
          />
        </div>
      </div>

      <div class="grid gap-1">
        <label for="name" class="text-xs font-semibold text-brand-muted">Product Name *</label>
        <InputText
          id="name"
          v-model="name"
          placeholder="e.g. Coca-Cola 330ml Can"
          required
        />
      </div>

      <div class="grid gap-1">
        <label for="category" class="text-xs font-semibold text-brand-muted">Category</label>
        <Select
          id="category"
          v-model="categoryId"
          :options="categories"
          option-label="name"
          option-value="id"
          placeholder="Select category (Optional)"
          show-clear
        />
      </div>

      <div class="grid gap-1">
        <label for="barcode" class="text-xs font-semibold text-brand-muted">Barcode (EAN-13 / UPC-A)</label>
        <div class="flex gap-2">
          <InputText
            id="barcode"
            v-model="barcode"
            placeholder="e.g. 4006381333931"
            class="w-full"
          />
          <Button
            type="button"
            icon="pi pi-camera"
            severity="secondary"
            title="Scan barcode"
            @click="scannerVisible = true"
          />
        </div>
        <small v-if="barcodeValidation" :class="barcodeValidation.valid ? 'text-green-600' : 'text-amber-600'">
          {{ barcodeValidation.message }}
        </small>
      </div>

      <div class="flex items-center justify-between rounded-[8px] border border-brand-border bg-brand-surface p-3">
        <div>
          <strong class="block text-sm">Lot & Expiration Tracking</strong>
          <small class="text-brand-muted">Track expiration dates and FEFO lot picking for this item</small>
        </div>
        <ToggleSwitch v-model="isLotTracked" />
      </div>

      <div v-if="isEditMode" class="flex items-center justify-between rounded-[8px] border border-brand-border bg-brand-surface p-3">
        <div>
          <strong class="block text-sm">Active Status</strong>
          <small class="text-brand-muted">Active products can be received and picked in inventory</small>
        </div>
        <ToggleSwitch v-model="isActive" />
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button label="Cancel" severity="secondary" outlined @click="closeDialog" />
        <Button
          :label="isEditMode ? 'Update Product' : 'Create Product'"
          icon="pi pi-check"
          :loading="catalogStore.loading"
          @click="saveProduct"
        />
      </div>
    </template>
  </Dialog>

  <BarcodeScannerModal
    v-model:visible="scannerVisible"
    @select="onBarcodeScanned"
  />
</template>
