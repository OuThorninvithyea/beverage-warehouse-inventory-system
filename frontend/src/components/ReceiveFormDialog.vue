<script setup lang="ts">
import { Camera, Download } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'

import type { Product } from '@/api/catalog'
import type { ReceiveInput } from '@/api/inventory'
import type { Location } from '@/api/warehouses'
import BarcodeScannerModal from '@/components/BarcodeScannerModal.vue'
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
const quantity = ref<string>('1')
const unitCost = ref<string>('0')
const lotNumber = ref('')
const expirationDate = ref('')
const reference = ref('')
const notes = ref('')

const errorMessage = ref('')
const scannerVisible = ref(false)

const selectedProduct = computed(() => props.products.find((p) => p.id === productId.value))

watch(
  () => props.visible,
  (isVis) => {
    if (!isVis) return
    errorMessage.value = ''
    locationId.value = props.locations[0]?.id || ''
    productId.value = ''
    quantity.value = '1'
    unitCost.value = '0'
    lotNumber.value = ''
    expirationDate.value = ''
    reference.value = ''
    notes.value = ''
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
        <DialogDescription>Inbound stock with optional lot and expiration tracking.</DialogDescription>
      </DialogHeader>

      <form class="grid gap-4" @submit.prevent="submitReceive">
        <p
          v-if="errorMessage"
          class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ errorMessage }}
        </p>

        <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="rcv-location">Target Location *</Label>
            <Select v-model="locationId">
              <SelectTrigger id="rcv-location" class="w-full">
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
            <Label for="rcv-product">Product *</Label>
            <div class="flex gap-2">
              <Select v-model="productId">
                <SelectTrigger id="rcv-product" class="flex-1">
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
            <Label for="rcv-exp">Expiration Date</Label>
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

  <BarcodeScannerModal v-model:visible="scannerVisible" @select="onBarcodeScanned" />
</template>
