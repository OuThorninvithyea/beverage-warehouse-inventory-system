<script setup lang="ts">
import { Camera, Check, LoaderCircle } from 'lucide-vue-next'
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
import { lookupExternalProduct, suggestProduct } from '@/lib/product-lookup'
import { resolveScan } from '@/lib/scan'
import { useCatalogStore } from '@/stores/catalog'

const props = defineProps<{
  visible: boolean
  product?: Product | null
  categories: Category[]
  /** A new product's scanned barcode: filled in and looked up on open. */
  prefillBarcode?: string
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

type LookupState =
  | { status: 'idle' }
  | { status: 'loading' }
  | { status: 'found'; brand: string | null }
  | { status: 'not-found' }
  | { status: 'duplicate'; product: Product }
const lookup = ref<LookupState>({ status: 'idle' })
// A slower lookup for an earlier barcode must not fill the form.
let lookupRequest = 0

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
    lookup.value = { status: 'idle' }
    if (!props.product && props.prefillBarcode) void fillFromBarcode(props.prefillBarcode)
  },
)

/**
 * For a new product: refuse a barcode the catalog already has, otherwise
 * suggest name, SKU and category from Open Food Facts. Only empty fields are
 * filled, so nothing the user typed is overwritten.
 */
async function fillFromBarcode(code: string) {
  barcode.value = code.trim()
  if (isEditMode.value || !barcode.value) return
  const request = ++lookupRequest
  lookup.value = { status: 'loading' }

  try {
    const existing = await resolveScan(barcode.value, [])
    if (request !== lookupRequest) return
    if (existing.kind === 'product') {
      lookup.value = { status: 'duplicate', product: existing.product }
      return
    }
  } catch {
    // The server enforces unique barcodes on save; carry on with the lookup.
  }

  const found = await lookupExternalProduct(barcode.value)
  if (request !== lookupRequest) return
  if (!found) {
    lookup.value = { status: 'not-found' }
    return
  }
  const suggestion = suggestProduct(found, props.categories)
  if (!name.value.trim()) name.value = suggestion.name
  if (!sku.value.trim()) sku.value = suggestion.sku
  if (categoryId.value === 'none' && suggestion.categoryId) categoryId.value = suggestion.categoryId
  lookup.value = { status: 'found', brand: suggestion.brand }
}

function onBarcodeScanned(scannedCode: string) {
  void fillFromBarcode(scannedCode)
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
  if (lookup.value.status === 'duplicate' && lookup.value.product.barcode === barcode.value.trim()) {
    errorMessage.value = `This barcode already belongs to ${lookup.value.product.name}`
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
          <small v-if="lookup.status === 'loading'" class="flex items-center gap-1 text-xs text-muted-foreground">
            <LoaderCircle class="size-3 animate-spin" /> Looking up this barcode…
          </small>
          <small v-else-if="lookup.status === 'found'" class="text-xs text-emerald-600">
            Details filled from Open Food Facts<template v-if="lookup.brand"> ({{ lookup.brand }})</template>.
            Check them before saving.
          </small>
          <small v-else-if="lookup.status === 'not-found'" class="text-xs text-muted-foreground">
            Not found in Open Food Facts. Enter the details by hand.
          </small>
          <small v-else-if="lookup.status === 'duplicate'" class="text-xs text-destructive">
            Already in the catalog as {{ lookup.product.name }} ({{ lookup.product.sku }}).
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
