<script setup lang="ts">
import { Check, Eye, EyeOff } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'

import type { CreateUserInput, UpdateUserInput, User, UserRole } from '@/api/users'
import type { Warehouse } from '@/api/warehouses'
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
import { Switch } from '@/components/ui/switch'
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
const warehouseId = ref<string>('global')
const password = ref('')
const isActive = ref(true)
const showPassword = ref(false)

const errorMessage = ref('')

const isEditMode = computed(() => Boolean(props.user?.id))

watch(
  () => props.visible,
  (isVis) => {
    if (!isVis) return
    errorMessage.value = ''
    showPassword.value = false
    password.value = ''
    if (props.user) {
      email.value = props.user.email
      fullName.value = props.user.full_name
      role.value = props.user.role
      warehouseId.value = props.user.warehouse_id ?? 'global'
      isActive.value = props.user.is_active
    } else {
      email.value = ''
      fullName.value = ''
      role.value = 'picker'
      warehouseId.value = props.warehouses[0]?.id ?? 'global'
      isActive.value = true
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

  const resolvedWarehouseId = warehouseId.value === 'global' ? null : warehouseId.value

  try {
    let saved: User
    if (isEditMode.value && props.user?.id) {
      const updatePayload: UpdateUserInput = {
        full_name: fullName.value.trim(),
        role: role.value,
        warehouse_id: resolvedWarehouseId,
        is_active: isActive.value,
      }
      saved = await usersStore.editUser(props.user.id, updatePayload)
    } else {
      const createPayload: CreateUserInput = {
        email: email.value.trim().toLowerCase(),
        full_name: fullName.value.trim(),
        role: role.value,
        warehouse_id: resolvedWarehouseId,
        password: password.value,
      }
      saved = await usersStore.addUser(createPayload)
    }
    emit('saved', saved)
    closeDialog()
  } catch (err: unknown) {
    errorMessage.value = err instanceof Error ? err.message : 'Failed to save user'
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
        <DialogTitle>{{ isEditMode ? 'Edit User Account' : 'Create User Account' }}</DialogTitle>
        <DialogDescription>Role-based access and warehouse scoping.</DialogDescription>
      </DialogHeader>

      <form class="grid gap-4" @submit.prevent="saveUser">
        <p
          v-if="errorMessage"
          class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ errorMessage }}
        </p>

        <div class="grid gap-2">
          <Label for="usr-name">Full Name *</Label>
          <Input id="usr-name" v-model="fullName" placeholder="e.g. John Smith" required />
        </div>

        <div class="grid gap-2">
          <Label for="usr-email">Email Address *</Label>
          <Input
            id="usr-email"
            v-model="email"
            type="email"
            placeholder="e.g. jsmith@bwims.local"
            :disabled="isEditMode"
            required
          />
        </div>

        <div class="grid grid-cols-2 gap-4 max-[520px]:grid-cols-1">
          <div class="grid gap-2">
            <Label for="usr-role">Role *</Label>
            <Select v-model="role">
              <SelectTrigger id="usr-role" class="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="admin">Admin (Full access)</SelectItem>
                <SelectItem value="warehouse_manager">Warehouse Manager</SelectItem>
                <SelectItem value="picker">Picker (Receive &amp; Pick)</SelectItem>
                <SelectItem value="viewer">Viewer (Read-only)</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="grid gap-2">
            <Label for="usr-wh">Assigned Warehouse</Label>
            <Select v-model="warehouseId">
              <SelectTrigger id="usr-wh" class="w-full">
                <SelectValue placeholder="All Warehouses (Global)" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="global">All Warehouses (Global)</SelectItem>
                <SelectItem v-for="wh in warehouses" :key="wh.id" :value="wh.id">
                  {{ wh.name }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div v-if="!isEditMode" class="grid gap-2 border-t pt-4">
          <Label for="usr-pw">Initial Password (min 12 chars) *</Label>
          <div class="relative">
            <Input
              id="usr-pw"
              v-model="password"
              :type="showPassword ? 'text' : 'password'"
              placeholder="••••••••••••"
              class="pr-10"
              required
            />
            <button
              type="button"
              class="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
              :aria-label="showPassword ? 'Hide password' : 'Show password'"
              @click="showPassword = !showPassword"
            >
              <EyeOff v-if="showPassword" class="size-4" />
              <Eye v-else class="size-4" />
            </button>
          </div>
        </div>

        <div v-if="isEditMode" class="flex items-center justify-between rounded-lg border bg-muted/40 p-3">
          <div class="grid gap-0.5">
            <strong class="text-sm">Account Active Status</strong>
            <small class="text-xs text-muted-foreground">Deactivated users cannot sign in to BWIMS</small>
          </div>
          <Switch v-model="isActive" />
        </div>
      </form>

      <DialogFooter>
        <Button variant="outline" @click="closeDialog">Cancel</Button>
        <Button :disabled="usersStore.loading" @click="saveUser">
          <Check class="size-4" />
          {{ isEditMode ? 'Update User' : 'Create User' }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
