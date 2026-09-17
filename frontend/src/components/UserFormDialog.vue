<script setup lang="ts">
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'
import Password from 'primevue/password'
import Select from 'primevue/select'
import ToggleSwitch from 'primevue/toggleswitch'
import { computed, ref, watch } from 'vue'

import type { CreateUserInput, UpdateUserInput, User, UserRole } from '@/api/users'
import type { Warehouse } from '@/api/warehouses'
import { useUsersStore } from '@/stores/users'

const props = defineProps<{
  visible: boolean
  user?: User | null
  warehouses: Warehouse[]
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'saved', user: User): void
}>()

const usersStore = useUsersStore()

const email = ref('')
const fullName = ref('')
const role = ref<UserRole>('picker')
const warehouseId = ref<string | null>(null)
const password = ref('')
const isActive = ref(true)

const errorMessage = ref('')

const isEditMode = computed(() => Boolean(props.user?.id))

const roleOptions = [
  { label: 'Admin (Full access)', value: 'admin' },
  { label: 'Warehouse Manager', value: 'warehouse_manager' },
  { label: 'Picker (Receive & Pick)', value: 'picker' },
  { label: 'Viewer (Read-only)', value: 'viewer' },
]

watch(
  () => props.visible,
  (isVis) => {
    if (isVis) {
      errorMessage.value = ''
      if (props.user) {
        email.value = props.user.email
        fullName.value = props.user.full_name
        role.value = props.user.role
        warehouseId.value = props.user.warehouse_id
        password.value = ''
        isActive.value = props.user.is_active
      } else {
        email.value = ''
        fullName.value = ''
        role.value = 'picker'
        warehouseId.value = props.warehouses[0]?.id || null
        password.value = ''
        isActive.value = true
      }
    }
  },
)

async function saveUser() {
  errorMessage.value = ''
  if (!fullName.value.trim()) {
    errorMessage.value = 'Full name is required'
    return
  }
  if (!isEditMode.value) {
    if (!email.value.trim()) {
      errorMessage.value = 'Email address is required'
      return
    }
    if (!password.value || password.value.length < 12) {
      errorMessage.value = 'Password must be at least 12 characters long'
      return
    }
  }

  try {
    let saved: User
    if (isEditMode.value && props.user?.id) {
      const updatePayload: UpdateUserInput = {
        full_name: fullName.value.trim(),
        role: role.value,
        warehouse_id: warehouseId.value || null,
        is_active: isActive.value,
      }
      saved = await usersStore.editUser(props.user.id, updatePayload)
    } else {
      const createPayload: CreateUserInput = {
        email: email.value.trim().toLowerCase(),
        full_name: fullName.value.trim(),
        role: role.value,
        warehouse_id: warehouseId.value || null,
        password: password.value,
      }
      saved = await usersStore.addUser(createPayload)
    }
    emit('saved', saved)
    closeDialog()
  } catch (err: any) {
    errorMessage.value = err.message || 'Failed to save user'
  }
}

function closeDialog() {
  emit('update:visible', false)
}
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    :header="isEditMode ? 'Edit User Account' : 'Create User Account'"
    :style="{ width: '90vw', maxWidth: '540px' }"
    @update:visible="closeDialog"
  >
    <form class="grid gap-4 py-2" @submit.prevent="saveUser">
      <Message v-if="errorMessage" severity="error">
        {{ errorMessage }}
      </Message>

      <div class="grid gap-1">
        <label for="usr-name" class="text-xs font-semibold text-brand-muted">Full Name *</label>
        <InputText
          id="usr-name"
          v-model="fullName"
          placeholder="e.g. John Smith"
          required
        />
      </div>

      <div class="grid gap-1">
        <label for="usr-email" class="text-xs font-semibold text-brand-muted">Email Address *</label>
        <InputText
          id="usr-email"
          v-model="email"
          type="email"
          placeholder="e.g. jsmith@bwims.local"
          :disabled="isEditMode"
          required
        />
      </div>

      <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
        <div class="grid gap-1">
          <label for="usr-role" class="text-xs font-semibold text-brand-muted">Role *</label>
          <Select
            id="usr-role"
            v-model="role"
            :options="roleOptions"
            option-label="label"
            option-value="value"
          />
        </div>

        <div class="grid gap-1">
          <label for="usr-wh" class="text-xs font-semibold text-brand-muted">Assigned Warehouse</label>
          <Select
            id="usr-wh"
            v-model="warehouseId"
            :options="warehouses"
            option-label="name"
            option-value="id"
            placeholder="All Warehouses (Global)"
            show-clear
          />
        </div>
      </div>

      <div v-if="!isEditMode" class="grid gap-1 border-t border-brand-border pt-3">
        <label for="usr-pw" class="text-xs font-semibold text-brand-muted">Initial Password (min 12 chars) *</label>
        <Password
          id="usr-pw"
          v-model="password"
          toggle-mask
          placeholder="••••••••••••"
          input-class="w-full"
          class="w-full"
          required
        />
      </div>

      <div v-if="isEditMode" class="flex items-center justify-between rounded-[8px] border border-brand-border bg-brand-surface p-3">
        <div>
          <strong class="block text-sm">Account Active Status</strong>
          <small class="text-brand-muted">Deactivated users cannot sign in to BWIMS</small>
        </div>
        <ToggleSwitch v-model="isActive" />
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button label="Cancel" severity="secondary" outlined @click="closeDialog" />
        <Button
          :label="isEditMode ? 'Update User' : 'Create User'"
          icon="pi pi-check"
          :loading="usersStore.loading"
          @click="saveUser"
        />
      </div>
    </template>
  </Dialog>
</template>
