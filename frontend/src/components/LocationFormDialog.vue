<script setup lang="ts">
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'
import ToggleSwitch from 'primevue/toggleswitch'
import { computed, reactive, ref, watch } from 'vue'

import { ApiClientError } from '@/api/client'
import type { Location, LocationInput } from '@/api/warehouses'
import { validateBarcode } from '@/lib/barcode'
import { useWarehousesStore } from '@/stores/warehouses'

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
  <Dialog
    :visible="visible"
    modal
    :header="isEdit ? 'Edit location' : 'Add location'"
    @update:visible="(value: boolean) => emit('update:visible', value)"
  >
    <div class="form-field">
      <label for="location-code">Code</label>
      <InputText id="location-code" v-model="form.code" :invalid="codeError !== null" />
      <Message v-if="codeError" severity="error" size="small" variant="simple">{{ codeError }}</Message>
    </div>
    <div class="form-field">
      <label for="location-zone">Zone</label>
      <InputText id="location-zone" v-model="form.zone" />
    </div>
    <div class="form-field">
      <label for="location-aisle">Aisle</label>
      <InputText id="location-aisle" v-model="form.aisle" />
    </div>
    <div class="form-field">
      <label for="location-rack">Rack</label>
      <InputText id="location-rack" v-model="form.rack" />
    </div>
    <div class="form-field">
      <label for="location-shelf">Shelf</label>
      <InputText id="location-shelf" v-model="form.shelf" />
    </div>
    <div class="form-field">
      <label for="location-barcode">Barcode</label>
      <InputText id="location-barcode" v-model="form.barcode" :invalid="barcodeError !== null" />
      <Message v-if="barcodeError" severity="error" size="small" variant="simple">{{ barcodeError }}</Message>
      <small v-else-if="barcodeHint" data-testid="barcode-hint">{{ barcodeHint }}</small>
    </div>
    <div class="form-field form-field--inline">
      <label for="location-pickable">Pickable</label>
      <ToggleSwitch id="location-pickable" v-model="form.is_pickable" />
    </div>
    <div class="form-field form-field--inline">
      <label for="location-active">Active</label>
      <ToggleSwitch id="location-active" v-model="form.is_active" />
    </div>
    <Message v-if="generalError" severity="error" size="small">{{ generalError }}</Message>
    <template #footer>
      <Button label="Cancel" severity="secondary" data-testid="cancel" @click="emit('update:visible', false)" />
      <Button label="Save" :loading="submitting" data-testid="submit" @click="submit" />
    </template>
  </Dialog>
</template>

<style scoped>
.form-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-bottom: 16px;
}
.form-field--inline {
  flex-direction: row;
  align-items: center;
  gap: 12px;
}
</style>
