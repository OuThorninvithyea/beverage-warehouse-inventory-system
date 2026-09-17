<script setup lang="ts">
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputNumber from 'primevue/inputnumber'
import { computed, ref } from 'vue'

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
  <Dialog
    :visible="visible"
    modal
    header="Print Barcode & Shelf Labels"
    :style="{ width: '90vw', maxWidth: '580px' }"
    @update:visible="closeDialog"
  >
    <div class="grid gap-4 py-2">
      <div class="flex items-center justify-between gap-4 rounded-[8px] bg-brand-surface p-3 border border-brand-border">
        <div>
          <strong class="block text-sm text-brand-navy">{{ itemTitle }}</strong>
          <small class="text-brand-muted">{{ itemSubtitle }}</small>
        </div>

        <div class="flex items-center gap-2">
          <label for="copies" class="text-xs font-bold text-brand-muted">Copies:</label>
          <InputNumber
            id="copies"
            v-model="labelCopies"
            :min="1"
            :max="24"
            class="w-[80px]"
          />
        </div>
      </div>

      <!-- Printable Area -->
      <div id="printable-labels-container" class="grid grid-cols-2 gap-3 max-h-[300px] overflow-y-auto p-2 border border-dashed border-slate-300 rounded-lg">
        <div
          v-for="n in labelCopies"
          :key="n"
          class="flex flex-col items-center justify-center p-3 border-2 border-slate-800 rounded bg-white text-center shadow-sm"
        >
          <span class="text-[10px] font-extrabold uppercase text-slate-700 tracking-tight line-clamp-1">{{ itemTitle }}</span>
          
          <!-- Barcode visual simulation -->
          <div class="my-1.5 flex h-10 items-center justify-center gap-[2px]">
            <div
              v-for="(bar, i) in 32"
              :key="i"
              class="h-full bg-slate-900"
              :style="{ width: i % 3 === 0 ? '3px' : '1.5px' }"
            />
          </div>

          <span class="font-mono text-xs font-extrabold text-slate-900 tracking-widest">{{ itemCode }}</span>
          <span class="text-[9px] text-slate-500 font-medium">BWIMS Distributor Tag</span>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button label="Close" severity="secondary" outlined @click="closeDialog" />
        <Button
          label="Print Labels"
          icon="pi pi-print"
          severity="success"
          @click="triggerPrint"
        />
      </div>
    </template>
  </Dialog>
</template>
