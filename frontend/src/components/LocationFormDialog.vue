<script setup lang="ts">
import { Loader2 } from 'lucide-vue-next'
import { computed, reactive, ref, watch } from 'vue'

import { ApiClientError } from '@/api/client'
import type { Location, LocationInput } from '@/api/warehouses'
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
import { Switch } from '@/components/ui/switch'
import { validateBarcode } from '@/lib/barcode'
import { useWarehousesStore } from '@/stores/warehouses'

const zoneOptions = [
  { value: 'AMBIENT', label: 'Ambient' },
  { value: 'CHILLED', label: 'Chilled' },
  { value: 'FROZEN', label: 'Frozen' },
  { value: 'RECEIVING', label: 'Receiving' },
  { value: 'QUARANTINE', label: 'Quarantine' },
]

const props = defineProps<{
  visible: boolean
  warehouseId: string
  location: Location | null
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  saved: []
}>()

const store = useWarehousesStore()
const isEdit = computed(() => props.location !== null)

const form = reactive({
  code: '',
  zone: '',
  aisle: '',
  rack: '',
  shelf: '',
  barcode: '',
  is_pickable: true,
  is_active: true,
})

const codeError = ref<string | null>(null)
const barcodeError = ref<string | null>(null)
const generalError = ref<string | null>(null)
const submitting = ref(false)

const barcodeHint = computed(() => {
  if (form.barcode.trim() === '') {
    return null
  }
  const result = validateBarcode(form.barcode)
  return result.valid ? `Valid ${result.format}` : result.message
})

watch(
  () => props.visible,
  (visible) => {
    if (!visible) {
      return
    }
    codeError.value = null
    barcodeError.value = null
    generalError.value = null
    form.code = props.location?.code ?? ''
    form.zone = props.location?.zone ?? ''
    form.aisle = props.location?.aisle ?? ''
    form.rack = props.location?.rack ?? ''
    form.shelf = props.location?.shelf ?? ''
    form.barcode = props.location?.barcode ?? ''
    form.is_pickable = props.location?.is_pickable ?? true
    form.is_active = props.location?.is_active ?? true
  },
  { immediate: true },
)

function validate(): boolean {
  codeError.value = form.code.trim() === '' ? 'Code is required.' : null
  return codeError.value === null
}

async function submit() {
  generalError.value = null
  barcodeError.value = null
  if (!validate()) {
    return
  }
  submitting.value = true
  try {
    const input: LocationInput = {
      code: form.code,
      zone: form.zone || null,
      aisle: form.aisle || null,
      rack: form.rack || null,
      shelf: form.shelf || null,
      barcode: form.barcode || null,
      is_pickable: form.is_pickable,
      is_active: form.is_active,
    }
    if (isEdit.value && props.location) {
      await store.updateLocation(props.warehouseId, props.location.id, input)
    } else {
      await store.createLocation(props.warehouseId, input)
    }
    emit('saved')
    emit('update:visible', false)
  } catch (err) {
    if (err instanceof ApiClientError && err.code === 'LOCATION_CODE_CONFLICT') {
      codeError.value = 'This code is already in use in this warehouse.'
    } else if (err instanceof ApiClientError && err.code === 'LOCATION_BARCODE_CONFLICT') {
      barcodeError.value = 'This barcode is already in use.'
    } else if (err instanceof ApiClientError) {
      generalError.value = err.message
    } else {
      generalError.value = 'Something went wrong. Try again.'
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Dialog :open="visible" @update:open="(value: boolean) => emit('update:visible', value)">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>{{ isEdit ? 'Edit location' : 'Add location' }}</DialogTitle>
        <DialogDescription>Zone, aisle, rack, shelf, and barcode details.</DialogDescription>
      </DialogHeader>

      <div class="grid gap-4">
        <div class="grid gap-2">
          <Label for="location-code">Code</Label>
          <Input id="location-code" v-model="form.code" :aria-invalid="codeError !== null" />
          <p v-if="codeError" class="text-xs text-destructive">{{ codeError }}</p>
        </div>

        <div class="grid grid-cols-2 gap-4">
          <div class="grid gap-2">
            <Label for="location-zone">Temperature / workflow zone</Label>
            <Select v-model="form.zone">
              <SelectTrigger id="location-zone" class="w-full"><SelectValue placeholder="Select zone" /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="zone in zoneOptions" :key="zone.value" :value="zone.value">{{ zone.label }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="grid gap-2">
            <Label for="location-aisle">Aisle</Label>
            <Input id="location-aisle" v-model="form.aisle" />
          </div>
          <div class="grid gap-2">
            <Label for="location-rack">Rack</Label>
            <Input id="location-rack" v-model="form.rack" />
          </div>
          <div class="grid gap-2">
            <Label for="location-shelf">Shelf</Label>
            <Input id="location-shelf" v-model="form.shelf" />
          </div>
        </div>

        <div class="grid gap-2">
          <Label for="location-barcode">Barcode</Label>
          <Input id="location-barcode" v-model="form.barcode" :aria-invalid="barcodeError !== null" />
          <p v-if="barcodeError" class="text-xs text-destructive">{{ barcodeError }}</p>
          <small v-else-if="barcodeHint" data-testid="barcode-hint" class="text-xs text-muted-foreground">
            {{ barcodeHint }}
          </small>
        </div>

        <div class="flex flex-wrap items-center gap-6">
          <div class="flex items-center gap-3">
            <Switch id="location-pickable" v-model="form.is_pickable" />
            <Label for="location-pickable">Pickable</Label>
          </div>
          <div class="flex items-center gap-3">
            <Switch id="location-active" v-model="form.is_active" />
            <Label for="location-active">Active</Label>
          </div>
        </div>

        <p v-if="generalError" class="text-sm text-destructive">{{ generalError }}</p>
      </div>

      <DialogFooter>
        <Button variant="outline" data-testid="cancel" @click="emit('update:visible', false)">
          Cancel
        </Button>
        <Button :disabled="submitting" data-testid="submit" @click="submit">
          <Loader2 v-if="submitting" class="size-4 animate-spin" />
          Save
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
