<script setup lang="ts">
import { Camera, Check } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'

import type { Category, Product, ProductInput } from '@/api/catalog'
import BarcodeScannerModal from '@/components/BarcodeScannerModal.vue'
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
import { Switch } from '@/components/ui/switch'
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
const categoryId = ref<string>('none')
const isLotTracked = ref(true)
const isActive = ref(true)

const errorMessage = ref('')
const scannerVisible = ref(false)

const unitOptions = ['case', 'bottle', 'can', 'pack', 'pallet', 'keg']

const isEditMode = computed(() => Boolean(props.product?.id))
const barcodeValidation = computed(() => {
  if (!barcode.value.trim()) return null
  return validateBarcode(barcode.value)
})

watch(
  () => props.visible,
  (isVis) => {
    if (!isVis) return
    errorMessage.value = ''
    if (props.product) {
      sku.value = props.product.sku
      name.value = props.product.name
      unit.value = props.product.unit
      barcode.value = props.product.barcode || ''
      categoryId.value = props.product.category_id ?? 'none'
      isLotTracked.value = props.product.is_lot_tracked
      isActive.value = props.product.is_active
    } else {
      sku.value = ''
      name.value = ''
      unit.value = 'case'
      barcode.value = ''
      categoryId.value = 'none'
      isLotTracked.value = true
      isActive.value = true
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
    category_id: categoryId.value === 'none' ? null : categoryId.value,
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
  } catch (err: unknown) {
    errorMessage.value = err instanceof Error ? err.message : 'Failed to save product'
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
        <DialogTitle>{{ isEditMode ? 'Edit Product' : 'Add New Product' }}</DialogTitle>
        <DialogDescription>SKU, barcode, unit, and FEFO lot tracking.</DialogDescription>
      </DialogHeader>

      <form class="grid gap-4" @submit.prevent="saveProduct">
        <p
          v-if="errorMessage"
          class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ errorMessage }}
        </p>

        <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="sku">SKU *</Label>
            <Input id="sku" v-model="sku" placeholder="e.g. COKE-330-CAN" class="uppercase" required />
          </div>

          <div class="grid gap-2">
            <Label for="unit">Unit *</Label>
            <Select v-model="unit">
              <SelectTrigger id="unit" class="w-full">
                <SelectValue placeholder="Select unit" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="option in unitOptions" :key="option" :value="option" class="capitalize">
                  {{ option }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div class="grid gap-2">
          <Label for="name">Product Name *</Label>
          <Input id="name" v-model="name" placeholder="e.g. Coca-Cola 330ml Can" required />
        </div>

        <div class="grid gap-2">
          <Label for="category">Category</Label>
          <Select v-model="categoryId">
            <SelectTrigger id="category" class="w-full">
              <SelectValue placeholder="Select category (Optional)" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="none">No category</SelectItem>
              <SelectItem v-for="cat in categories" :key="cat.id" :value="cat.id">
                {{ cat.name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div class="grid gap-2">
          <Label for="barcode">Barcode (EAN-13 / UPC-A)</Label>
          <div class="flex gap-2">
            <Input id="barcode" v-model="barcode" placeholder="e.g. 4006381333931" class="flex-1" />
            <Button type="button" variant="outline" size="icon" title="Scan barcode" @click="scannerVisible = true">
              <Camera class="size-4" />
            </Button>
          </div>
          <small
            v-if="barcodeValidation"
            class="text-xs"
            :class="barcodeValidation.valid ? 'text-emerald-600' : 'text-amber-600'"
          >
            {{ barcodeValidation.message }}
          </small>
        </div>

        <div class="flex items-center justify-between rounded-lg border bg-muted/40 p-3">
          <div class="grid gap-0.5">
            <strong class="text-sm">Lot &amp; Expiration Tracking</strong>
            <small class="text-xs text-muted-foreground">
              Track expiration dates and FEFO lot picking for this item
            </small>
          </div>
          <Switch v-model="isLotTracked" />
        </div>

        <div v-if="isEditMode" class="flex items-center justify-between rounded-lg border bg-muted/40 p-3">
          <div class="grid gap-0.5">
            <strong class="text-sm">Active Status</strong>
            <small class="text-xs text-muted-foreground">
              Active products can be received and picked in inventory
            </small>
          </div>
          <Switch v-model="isActive" />
        </div>
      </form>

      <DialogFooter>
        <Button variant="outline" @click="closeDialog">Cancel</Button>
        <Button :disabled="catalogStore.loading" @click="saveProduct">
          <Check class="size-4" />
          {{ isEditMode ? 'Update Product' : 'Create Product' }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <BarcodeScannerModal v-model:visible="scannerVisible" @select="onBarcodeScanned" />
</template>
