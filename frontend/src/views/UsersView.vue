<script setup lang="ts">
import Button from 'primevue/button'
import Card from 'primevue/card'
import Column from 'primevue/column'
import DataTable from 'primevue/datatable'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'
import Select from 'primevue/select'
import Tag from 'primevue/tag'
import { computed, onMounted, ref, watch } from 'vue'

import type { User, UserRole } from '@/api/users'
import PasswordResetDialog from '@/components/PasswordResetDialog.vue'
import UserFormDialog from '@/components/UserFormDialog.vue'
import { useAuthStore } from '@/stores/auth'
import { useUsersStore } from '@/stores/users'
import { useWarehousesStore } from '@/stores/warehouses'

const auth = useAuthStore()
const usersStore = useUsersStore()
const warehouseStore = useWarehousesStore()

const searchQuery = ref('')
const selectedRole = ref<UserRole>()

const userFormVisible = ref(false)
const resetPasswordVisible = ref(false)
const selectedUser = ref<User | null>(null)

const isAdmin = computed(() => auth.user?.role === 'admin')

onMounted(async () => {
  if (isAdmin.value) {
    await Promise.all([
      warehouseStore.fetchWarehouses(),
      usersStore.fetchUsers({ search: searchQuery.value, role: selectedRole.value }),
    ])
  }
})

watch([searchQuery, selectedRole], () => {
  if (isAdmin.value) {
    void usersStore.fetchUsers({ search: searchQuery.value, role: selectedRole.value })
  }
})

function openAddUser() {
  selectedUser.value = null
  userFormVisible.value = true
}

function openEditUser(user: User) {
  selectedUser.value = user
  userFormVisible.value = true
}

function openResetPassword(user: User) {
  selectedUser.value = user
  resetPasswordVisible.value = true
}

async function deactivateUser(user: User) {
  if (user.id === auth.user?.id) {
    alert('You cannot deactivate your own account.')
    return
  }
  if (confirm(`Are you sure you want to deactivate "${user.full_name}"?`)) {
    await usersStore.removeUser(user.id)
  }
}

function getRoleSeverity(role: UserRole): 'danger' | 'warn' | 'info' | 'secondary' {
  switch (role) {
    case 'admin':
      return 'danger'
    case 'warehouse_manager':
      return 'warn'
    case 'picker':
      return 'info'
    case 'viewer':
      return 'secondary'
    default:
      return 'secondary'
  }
}

function getWarehouseName(warehouseId: string | null): string {
  if (!warehouseId) return 'All Warehouses (Global)'
  const wh = warehouseStore.warehouses.find((w) => w.id === warehouseId)
  return wh ? wh.name : 'All Warehouses'
}
</script>

<template>
  <div class="grid gap-6">
    <div class="flex flex-wrap items-center justify-between gap-4">
      <div>
        <h1 class="m-0 text-2xl font-bold text-brand-navy">User Management</h1>
        <p class="m-0 text-sm text-brand-muted">Manage system users, four-role RBAC permissions, and warehouse scoping</p>
      </div>

      <Button
        v-if="isAdmin"
        label="Add User"
        icon="pi pi-user-plus"
        @click="openAddUser"
      />
    </div>

    <Message v-if="!isAdmin" severity="error">
      Access Denied: User Management is strictly restricted to Administrator accounts.
    </Message>

    <Card v-else>
      <template #content>
        <div class="mb-4 flex flex-wrap items-center justify-between gap-4">
          <div class="flex flex-wrap items-center gap-3">
            <InputText
              v-model="searchQuery"
              placeholder="Search name or email..."
              class="w-[260px]"
            />

            <Select
              v-model="selectedRole"
              :options="[
                { label: 'Admin', value: 'admin' },
                { label: 'Warehouse Manager', value: 'warehouse_manager' },
                { label: 'Picker', value: 'picker' },
                { label: 'Viewer', value: 'viewer' },
              ]"
              option-label="label"
              option-value="value"
              placeholder="All Roles"
              show-clear
              class="w-[180px]"
            />
          </div>
        </div>

        <Message v-if="usersStore.error" severity="error" class="mb-4">
          {{ usersStore.error }}
        </Message>

        <DataTable
          :value="usersStore.users"
          :loading="usersStore.loading"
          data-key="id"
          responsive-layout="scroll"
          striped-rows
          paginator
          :rows="10"
          class="p-datatable-sm"
        >
          <template #empty>
            <div class="py-8 text-center text-brand-muted">
              No user accounts found.
            </div>
          </template>

          <Column field="full_name" header="Full Name" sortable>
            <template #body="{ data }">
              <div>
                <strong class="block text-brand-navy">{{ data.full_name }}</strong>
                <small class="text-brand-muted">{{ data.email }}</small>
              </div>
            </template>
          </Column>

          <Column field="role" header="Role" sortable>
            <template #body="{ data }">
              <Tag
                :value="data.role.replace('_', ' ').toUpperCase()"
                :severity="getRoleSeverity(data.role)"
                class="font-mono text-[10px]"
              />
            </template>
          </Column>

          <Column field="warehouse_id" header="Assigned Warehouse">
            <template #body="{ data }">
              <span class="text-xs font-medium text-slate-700">
                {{ getWarehouseName(data.warehouse_id) }}
              </span>
            </template>
          </Column>

          <Column field="is_active" header="Status">
            <template #body="{ data }">
              <Tag
                :value="data.is_active ? 'Active' : 'Inactive'"
                :severity="data.is_active ? 'success' : 'warn'"
              />
            </template>
          </Column>

          <Column header="Actions" align-frozen="right" freeze>
            <template #body="{ data }">
              <div class="flex items-center gap-2">
                <Button
                  icon="pi pi-pencil"
                  severity="secondary"
                  text
                  rounded
                  size="small"
                  title="Edit user"
                  @click="openEditUser(data)"
                />
                <Button
                  icon="pi pi-key"
                  severity="warn"
                  text
                  rounded
                  size="small"
                  title="Reset password"
                  @click="openResetPassword(data)"
                />
                <Button
                  v-if="data.is_active && data.id !== auth.user?.id"
                  icon="pi pi-user-minus"
                  severity="danger"
                  text
                  rounded
                  size="small"
                  title="Deactivate user"
                  @click="deactivateUser(data)"
                />
              </div>
            </template>
          </Column>
        </DataTable>
      </template>
    </Card>

    <UserFormDialog
      v-model:visible="userFormVisible"
      :user="selectedUser"
      :warehouses="warehouseStore.warehouses"
      @saved="usersStore.fetchUsers({ search: searchQuery, role: selectedRole })"
    />

    <PasswordResetDialog
      v-model:visible="resetPasswordVisible"
      :user="selectedUser"
      @success="usersStore.fetchUsers({ search: searchQuery, role: selectedRole })"
    />
  </div>
</template>
