<script setup lang="ts">
import { Download, PackagePlus } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'

import { listCategories, type Category, type Product } from '@/api/catalog'
import type { ReceiveInput } from '@/api/inventory'
import type { Location } from '@/api/warehouses'
import { Badge } from '@/components/ui/badge'
import LocationSelect from '@/components/LocationSelect.vue'
import ProductFormDialog from '@/components/ProductFormDialog.vue'
import ScanBar from '@/components/ScanBar.vue'
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
import { useLocationLabels } from '@/lib/location-label'
import { useLocationStock } from '@/lib/location-stock'
import { resolveScan, type ScanFeedback } from '@/lib/scan'
import { useAuthStore } from '@/stores/auth'
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
const auth = useAuthStore()
const labels = useLocationLabels()

const locationId = ref('')
const productId = ref('')
const quantity = ref<string>('1')
const unitCost = ref<string>('0')
const lotNumber = ref('')
const expirationDate = ref('')
const reference = ref('')
const notes = ref('')

const errorMessage = ref('')
const scanFeedback = ref<ScanFeedback | null>(null)
// A carton the catalog does not know yet, offered for creation.
const unknownBarcode = ref('')
const productFormVisible = ref(false)
const categories = ref<Category[]>([])

// Only managers and admins may add products; a picker is told who to ask.
const canCreateProducts = computed(() =>
  ['admin', 'warehouse_manager'].includes(auth.user?.role ?? ''),
)

// Receiving fills empty shelves, so every active product qualifies. The
// catalog store's list is paginated to 20, so load the full catalog instead.
// No location stock is needed here, so the composable is given no location.
const stock = useLocationStock(ref(''))
const activeLocations = computed(() => props.locations.filter((loc) => loc.is_active))
const selectedProduct = computed(() =>
  stock.allProducts.value.find((p) => p.id === productId.value),
)

watch(
  () => props.visible,
  async (isVis) => {
    if (!isVis) return
    errorMessage.value = ''
    scanFeedback.value = null
    unknownBarcode.value = ''
    void stock.loadCatalog()
    locationId.value = activeLocations.value[0]?.id || ''
    productId.value = ''
    quantity.value = '1'
    unitCost.value = '0'
    lotNumber.value = ''
    expirationDate.value = ''
    reference.value = ''
    notes.value = ''
  },
)

// Scan the shelf, then the carton. An unknown carton can be created here and
// received straight away.
async function onScan(code: string) {
  unknownBarcode.value = ''
  try {
    const result = await resolveScan(code, props.locations)
    if (result.kind === 'location') {
      const { location } = result
      if (!location.is_active) {
        scanFeedback.value = { tone: 'warning', text: `${labels.label(location)} is inactive.` }
        return
      }
      locationId.value = location.id
      scanFeedback.value = { tone: 'success', text: `Receiving into ${labels.label(location)}. Now scan the carton.` }
      return
    }
    if (result.kind === 'product') {
      productId.value = result.product.id
      scanFeedback.value = { tone: 'success', text: `${result.product.name} selected.` }
      return
    }
    unknownBarcode.value = result.code
    scanFeedback.value = {
      tone: 'warning',
      text: canCreateProducts.value
        ? `${result.code} is not in the catalog yet.`
        : `${result.code} is not in the catalog yet. Ask a manager to add it.`,
    }
  } catch (err: unknown) {
    scanFeedback.value = { tone: 'warning', text: err instanceof Error ? err.message : 'Scan lookup failed' }
  }
}

/** Every active category, so the new product can be filed under any of them. */
async function loadCategories() {
  const rows: Category[] = []
  let after: string | undefined
  do {
    const page = await listCategories({ is_active: true, limit: 100, after })
    rows.push(...page.items)
    after = page.page.has_more ? (page.page.next_cursor ?? undefined) : undefined
  } while (after)
  categories.value = rows.sort((a, b) => a.name.localeCompare(b.name))
}

async function openProductForm() {
  if (categories.value.length === 0) await loadCategories().catch(() => undefined)
  productFormVisible.value = true
}

async function onProductCreated(product: Product) {
  await stock.loadCatalog()
  productId.value = product.id
  unknownBarcode.value = ''
  scanFeedback.value = { tone: 'success', text: `${product.name} added to the catalog and selected.` }
}

async function submitReceive() {
  errorMessage.value = ''
  const qty = Number(quantity.value)
  const cost = Number(unitCost.value)

  if (!locationId.value) {
    errorMessage.value = 'Target location is required'
    return
  }
  if (!productId.value) {
    errorMessage.value = 'Product is required'
    return
  }
  if (!Number.isFinite(qty) || qty <= 0) {
    errorMessage.value = 'Quantity must be greater than 0'
    return
  }
  if (!Number.isFinite(cost) || cost < 0) {
    errorMessage.value = 'Unit cost must be 0 or greater'
    return
  }
  if (selectedProduct.value?.is_lot_tracked && !lotNumber.value.trim()) {
    errorMessage.value = 'Lot number is required for lot-tracked products'
    return
  }
  if (selectedProduct.value?.is_lot_tracked && !expirationDate.value) {
    errorMessage.value = 'Expiration date is required for lot-tracked products'
    return
  }

  const payload: ReceiveInput = {
    location_id: locationId.value,
    product_id: productId.value,
    quantity: qty.toString(),
    unit_cost: cost.toString(),
    lot_number: lotNumber.value.trim() || null,
    expiration_date: expirationDate.value || null,
    reference: reference.value.trim() || null,
    notes: notes.value.trim() || null,
  }

  try {
    await inventoryStore.doReceive(payload)
    emit('submitted')
    closeDialog()
  } catch (err: unknown) {
    errorMessage.value = err instanceof Error ? err.message : 'Failed to receive stock'
  }
}

function closeDialog() {
  emit('update:visible', false)
}
</script>

<template>
  <Dialog :open="visible" @update:open="(value: boolean) => emit('update:visible', value)">
    <DialogContent class="sm:max-w-2xl">
      <DialogHeader>
        <DialogTitle>Receive Inventory</DialogTitle>
        <DialogDescription>Inbound stock with mandatory lot and expiry tracking for applicable products.</DialogDescription>
      </DialogHeader>

      <form class="grid gap-4" @submit.prevent="submitReceive">
        <p
          v-if="errorMessage"
          class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ errorMessage }}
        </p>

        <ScanBar :feedback="scanFeedback" hint="Scan the shelf label, then the carton." @scan="onScan">
          <template #action>
            <Button
              v-if="unknownBarcode && canCreateProducts"
              type="button"
              size="sm"
              variant="secondary"
              class="ml-auto h-7"
              @click="openProductForm"
            >
              <PackagePlus class="size-4" />
              Create product
            </Button>
          </template>
        </ScanBar>

        <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="rcv-location">Target Location *</Label>
            <LocationSelect
              id="rcv-location"
              v-model="locationId"
              :locations="activeLocations"
              placeholder="Select location"
            />
          </div>

          <div class="grid gap-2">
            <Label for="rcv-product">Product *</Label>
            <Select v-model="productId">
              <SelectTrigger id="rcv-product" class="w-full">
                <SelectValue placeholder="Select product" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="product in stock.allProducts.value" :key="product.id" :value="product.id">
                  {{ product.name }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div v-if="selectedProduct" class="flex items-center justify-between rounded-lg border bg-muted/40 p-3 text-xs">
          <div class="grid gap-0.5">
            <span class="text-sm font-semibold">{{ selectedProduct.name }}</span>
            <span class="text-muted-foreground">
              SKU: {{ selectedProduct.sku }} | Unit: {{ selectedProduct.unit }}
            </span>
          </div>
          <Badge v-if="selectedProduct.is_lot_tracked" variant="outline" class="border-amber-500/30 bg-amber-500/10 text-amber-700">
            Lot Tracked
          </Badge>
        </div>

        <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="rcv-qty">Quantity *</Label>
            <Input id="rcv-qty" v-model="quantity" type="number" min="0.001" step="0.001" placeholder="e.g. 50" required />
          </div>

          <div class="grid gap-2">
            <Label for="rcv-cost">Unit Cost ($) *</Label>
            <Input id="rcv-cost" v-model="unitCost" type="number" min="0" step="0.01" placeholder="0.00" required />
          </div>
        </div>

        <div v-if="selectedProduct?.is_lot_tracked" class="grid grid-cols-2 gap-4 border-t pt-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="rcv-lot">Lot Number *</Label>
            <Input id="rcv-lot" v-model="lotNumber" placeholder="e.g. LOT-2026-08-A" required />
          </div>

          <div class="grid gap-2">
            <Label for="rcv-exp">Expiration Date<span v-if="selectedProduct?.is_lot_tracked"> *</span></Label>
            <Input id="rcv-exp" v-model="expirationDate" type="date" />
          </div>
        </div>

        <div class="grid grid-cols-2 gap-4 border-t pt-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="rcv-ref">Reference (PO / Invoice)</Label>
            <Input id="rcv-ref" v-model="reference" placeholder="e.g. PO-10042" />
          </div>

          <div class="grid gap-2">
            <Label for="rcv-notes">Notes</Label>
            <Textarea id="rcv-notes" v-model="notes" rows="1" placeholder="Optional receiving notes" />
          </div>
        </div>
      </form>

      <DialogFooter>
        <Button variant="outline" @click="closeDialog">Cancel</Button>
        <Button :disabled="inventoryStore.loading" @click="submitReceive">
          <Download class="size-4" />
          Receive Stock
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <ProductFormDialog
    v-model:visible="productFormVisible"
    :categories="categories"
    :prefill-barcode="unknownBarcode"
    @saved="onProductCreated"
  />
</template>
