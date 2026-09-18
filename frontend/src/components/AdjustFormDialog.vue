<script setup lang="ts">
import { Check } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'

import type { Product } from '@/api/catalog'
import type { AdjustInput, Lot } from '@/api/inventory'
import type { Location } from '@/api/warehouses'
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
const lotId = ref<string>('none')
const direction = ref<'increase' | 'decrease'>('increase')
const quantity = ref<string>('1')
const notes = ref('')

const errorMessage = ref('')
const availableLots = ref<Lot[]>([])

const selectedProduct = computed(() => props.products.find((p) => p.id === productId.value))

watch(
  () => productId.value,
  async (newProdId) => {
    lotId.value = 'none'
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
    locationId.value = props.locations[0]?.id || ''
    productId.value = ''
    lotId.value = 'none'
    direction.value = 'increase'
    quantity.value = '1'
    notes.value = ''
    availableLots.value = []
  },
)

async function submitAdjust() {
  errorMessage.value = ''
  const qty = Number(quantity.value)

  if (!locationId.value) {
    errorMessage.value = 'Location is required'
    return
  }
  if (!productId.value) {
    errorMessage.value = 'Product is required'
    return
  }
  if (!Number.isFinite(qty) || qty <= 0) {
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
    quantity: qty.toString(),
    lot_id: lotId.value === 'none' ? null : lotId.value,
    notes: notes.value.trim(),
  }

  try {
    await inventoryStore.doAdjust(payload)
    emit('submitted')
    closeDialog()
  } catch (err: unknown) {
    errorMessage.value = err instanceof Error ? err.message : 'Stock adjustment failed'
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
        <DialogTitle>Cycle Count Stock Adjustment</DialogTitle>
        <DialogDescription>Correct counted stock with a mandatory audit reason.</DialogDescription>
      </DialogHeader>

      <form class="grid gap-4" @submit.prevent="submitAdjust">
        <p
          v-if="errorMessage"
          class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ errorMessage }}
        </p>

        <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="adj-location">Location *</Label>
            <Select v-model="locationId">
              <SelectTrigger id="adj-location" class="w-full">
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
            <Label for="adj-product">Product *</Label>
            <Select v-model="productId">
              <SelectTrigger id="adj-product" class="w-full">
                <SelectValue placeholder="Select product" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="product in products" :key="product.id" :value="product.id">
                  {{ product.name }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="adj-direction">Adjustment Direction *</Label>
            <Select v-model="direction">
              <SelectTrigger id="adj-direction" class="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="increase">Increase (+ Stock Found)</SelectItem>
                <SelectItem value="decrease">Decrease (- Stock Missing/Damaged)</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="grid gap-2">
            <Label for="adj-qty">Quantity *</Label>
            <Input id="adj-qty" v-model="quantity" type="number" min="0.001" step="0.001" placeholder="e.g. 2" required />
          </div>
        </div>

        <div v-if="selectedProduct?.is_lot_tracked" class="grid gap-2">
          <Label for="adj-lot">Lot (Optional)</Label>
          <Select v-model="lotId">
            <SelectTrigger id="adj-lot" class="w-full">
              <SelectValue placeholder="Select lot" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="none">No specific lot</SelectItem>
              <SelectItem v-for="lot in availableLots" :key="lot.id" :value="lot.id">
                {{ lot.lot_number }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div class="grid gap-2 border-t pt-4">
          <Label for="adj-notes">Mandatory Audit Reason Note *</Label>
          <Textarea
            id="adj-notes"
            v-model="notes"
            rows="2"
            placeholder="e.g. Damaged during forklift transit / Periodic physical count correction"
            required
          />
        </div>
      </form>

      <DialogFooter>
        <Button variant="outline" @click="closeDialog">Cancel</Button>
        <Button
          :disabled="inventoryStore.loading"
          :variant="direction === 'increase' ? 'default' : 'destructive'"
          @click="submitAdjust"
        >
          <Check class="size-4" />
          Commit Adjustment
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
