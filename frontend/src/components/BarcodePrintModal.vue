<script setup lang="ts">
import { Printer, TriangleAlert } from 'lucide-vue-next'
import { computed, ref } from 'vue'

import { buildBarcodeSymbol } from '@/lib/barcode-render'
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

import type { Product } from '@/api/catalog'
import type { Location } from '@/api/warehouses'

const props = defineProps<{
  visible: boolean
  product?: Product | null
  location?: Location | null
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
}>()

const labelCopies = ref(4)

const itemTitle = computed(() => {
  if (props.product) return props.product.name
  if (props.location) return `Location: ${props.location.code}`
  return 'Barcode Label'
})

const itemCode = computed(() => {
  if (props.product) return props.product.barcode || props.product.sku
  if (props.location) return props.location.barcode || props.location.code
  return 'BWIMS-ITEM'
})

const itemSubtitle = computed(() => {
  if (props.product) return `SKU: ${props.product.sku} | Unit: ${props.product.unit}`
  if (props.location) return `Zone: ${props.location.zone || 'General'} | Aisle: ${props.location.aisle || 'Main'}`
  return ''
})

// The symbol is generated from the digits, so a printed label is actually
// scannable. A value with a bad check digit has no legitimate symbol, so the
// label shows a warning instead of bars a scanner would reject.
const symbol = computed(() => buildBarcodeSymbol(itemCode.value, 2, 44))

function triggerPrint() {
  window.print()
}

function closeDialog() {
  emit('update:visible', false)
}
</script>

<template>
  <Dialog :open="visible" @update:open="(value: boolean) => emit('update:visible', value)">
    <DialogContent class="sm:max-w-xl">
      <DialogHeader>
        <DialogTitle>Print Barcode &amp; Shelf Labels</DialogTitle>
        <DialogDescription>Preview and print label sheets for this item.</DialogDescription>
      </DialogHeader>

      <div class="grid gap-4">
        <div class="flex items-center justify-between gap-4 rounded-lg border bg-muted/40 p-3">
          <div class="grid gap-0.5">
            <strong class="text-sm">{{ itemTitle }}</strong>
            <small class="text-xs text-muted-foreground">{{ itemSubtitle }}</small>
          </div>

          <div class="flex items-center gap-2">
            <Label for="copies" class="text-xs">Copies</Label>
            <Input id="copies" v-model.number="labelCopies" type="number" min="1" max="24" class="w-20" />
          </div>
        </div>

        <div
          id="printable-labels-container"
          class="grid max-h-[300px] grid-cols-2 gap-3 overflow-y-auto rounded-lg border border-dashed p-2"
        >
          <div
            v-for="n in labelCopies"
            :key="n"
            class="flex flex-col items-center justify-center rounded border-2 border-slate-800 bg-white p-3 text-center"
          >
            <span class="line-clamp-1 text-[10px] font-extrabold uppercase tracking-tight text-slate-700">
              {{ itemTitle }}
            </span>

            <svg
              v-if="symbol"
              class="my-1.5"
              :width="symbol.width"
              :height="symbol.height"
              :viewBox="`0 0 ${symbol.width} ${symbol.height}`"
              role="img"
              :aria-label="`Barcode ${symbol.value}`"
            >
              <rect :width="symbol.width" :height="symbol.height" fill="#ffffff" />
              <rect
                v-for="(bar, barIndex) in symbol.bars"
                :key="barIndex"
                :x="bar.x"
                y="0"
                :width="bar.width"
                :height="symbol.height"
                fill="#0f172a"
              />
            </svg>

            <span
              v-else
              class="my-1.5 flex items-center gap-1 text-[9px] font-semibold text-amber-700"
            >
              <TriangleAlert class="size-3" /> Not a scannable EAN-13 / UPC-A value
            </span>

            <span class="font-mono text-xs font-extrabold tracking-widest text-slate-900">{{ itemCode }}</span>
            <span class="text-[9px] font-medium text-slate-500">BWIMS Distributor Tag</span>
          </div>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="closeDialog">Close</Button>
        <Button @click="triggerPrint">
          <Printer class="size-4" />
          Print Labels
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
