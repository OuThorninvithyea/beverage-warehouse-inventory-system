<script setup lang="ts">
import { Loader2 } from 'lucide-vue-next'
import { computed, reactive, ref, watch } from 'vue'

import { ApiClientError } from '@/api/client'
import type { Warehouse, WarehouseInput } from '@/api/warehouses'
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
  <Dialog :open="visible" @update:open="(value: boolean) => emit('update:visible', value)">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>{{ isEdit ? 'Edit warehouse' : 'Add warehouse' }}</DialogTitle>
        <DialogDescription>Facility code, name, and address details.</DialogDescription>
      </DialogHeader>

      <div class="grid gap-4">
        <div class="grid gap-2">
          <Label for="warehouse-code">Code</Label>
          <Input id="warehouse-code" v-model="form.code" :aria-invalid="codeError !== null" />
          <p v-if="codeError" class="text-xs text-destructive">{{ codeError }}</p>
        </div>

        <div class="grid gap-2">
          <Label for="warehouse-name">Name</Label>
          <Input id="warehouse-name" v-model="form.name" :aria-invalid="nameError !== null" />
          <p v-if="nameError" class="text-xs text-destructive">{{ nameError }}</p>
        </div>

        <div class="grid gap-2">
          <Label for="warehouse-address">Address</Label>
          <Input id="warehouse-address" v-model="form.address" />
        </div>

        <div class="flex items-center gap-3">
          <Switch id="warehouse-active" v-model="form.is_active" />
          <Label for="warehouse-active">Active</Label>
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
