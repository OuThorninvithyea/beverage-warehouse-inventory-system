<script setup lang="ts">
import { Printer } from 'lucide-vue-next'
import { computed, ref } from 'vue'

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

            <div class="my-1.5 flex h-10 items-center justify-center gap-[2px]">
              <div
                v-for="(bar, i) in 32"
                :key="i"
                class="h-full bg-slate-900"
                :style="{ width: i % 3 === 0 ? '3px' : '1.5px' }"
              />
            </div>

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
