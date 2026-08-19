<script setup lang="ts">
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'
import ToggleSwitch from 'primevue/toggleswitch'
import { computed, reactive, ref, watch } from 'vue'

import { ApiClientError } from '@/api/client'
import type { Warehouse, WarehouseInput } from '@/api/warehouses'
import { useWarehousesStore } from '@/stores/warehouses'

const props = defineProps<{
  visible: boolean
  warehouse: Warehouse | null
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  saved: []
}>()

const store = useWarehousesStore()
const isEdit = computed(() => props.warehouse !== null)

const form = reactive({
  code: '',
  name: '',
  address: '',
  is_active: true,
})

const codeError = ref<string | null>(null)
const nameError = ref<string | null>(null)
const generalError = ref<string | null>(null)
const submitting = ref(false)

watch(
  () => props.visible,
  (visible) => {
    if (!visible) {
      return
    }
    codeError.value = null
    nameError.value = null
    generalError.value = null
    form.code = props.warehouse?.code ?? ''
    form.name = props.warehouse?.name ?? ''
    form.address = props.warehouse?.address ?? ''
    form.is_active = props.warehouse?.is_active ?? true
  },
  { immediate: true },
)

function validate(): boolean {
  codeError.value = form.code.trim() === '' ? 'Code is required.' : null
  nameError.value = form.name.trim() === '' ? 'Name is required.' : null
  return codeError.value === null && nameError.value === null
}

async function submit() {
  generalError.value = null
  if (!validate()) {
    return
  }
  submitting.value = true
  try {
    const input: WarehouseInput = {
      code: form.code,
      name: form.name,
      address: form.address || null,
      is_active: form.is_active,
    }
    if (isEdit.value && props.warehouse) {
      await store.updateWarehouse(props.warehouse.id, input)
    } else {
      await store.createWarehouse(input)
    }
    emit('saved')
    emit('update:visible', false)
  } catch (err) {
    if (err instanceof ApiClientError && err.code === 'WAREHOUSE_CODE_CONFLICT') {
      codeError.value = 'This code is already in use.'
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
    :header="isEdit ? 'Edit warehouse' : 'Add warehouse'"
    @update:visible="(value: boolean) => emit('update:visible', value)"
  >
    <div class="form-field">
      <label for="warehouse-code">Code</label>
      <InputText id="warehouse-code" v-model="form.code" :invalid="codeError !== null" />
      <Message v-if="codeError" severity="error" size="small" variant="simple">{{ codeError }}</Message>
    </div>
    <div class="form-field">
      <label for="warehouse-name">Name</label>
      <InputText id="warehouse-name" v-model="form.name" :invalid="nameError !== null" />
      <Message v-if="nameError" severity="error" size="small" variant="simple">{{ nameError }}</Message>
    </div>
    <div class="form-field">
      <label for="warehouse-address">Address</label>
      <InputText id="warehouse-address" v-model="form.address" />
    </div>
    <div class="form-field form-field--inline">
      <label for="warehouse-active">Active</label>
      <ToggleSwitch id="warehouse-active" v-model="form.is_active" />
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
