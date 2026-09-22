<script setup lang="ts">
import { KeyRound, Pencil, Search, UserMinus, UserPlus } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'
import { toast } from 'vue-sonner'

import type { User, UserRole } from '@/api/users'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import PasswordResetDialog from '@/components/PasswordResetDialog.vue'
import UserFormDialog from '@/components/UserFormDialog.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useAuthStore } from '@/stores/auth'
import { useUsersStore } from '@/stores/users'
import { useWarehousesStore } from '@/stores/warehouses'

const auth = useAuthStore()
const usersStore = useUsersStore()
const warehouseStore = useWarehousesStore()

const pendingUser = ref<User | null>(null)
const confirmVisible = ref(false)

const searchQuery = ref('')
const roleFilter = ref<'all' | UserRole>('all')
const selectedRole = computed(() =>
  roleFilter.value === 'all' ? undefined : (roleFilter.value as UserRole),
)

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

function deactivateUser(user: User) {
  if (user.id === auth.user?.id) {
    toast.error('You cannot deactivate your own account.')
    return
  }
  pendingUser.value = user
  confirmVisible.value = true
}

async function confirmDeactivateUser() {
  const user = pendingUser.value
  confirmVisible.value = false
  if (!user) return
  await usersStore.removeUser(user.id)
  pendingUser.value = null
}

function getRoleClass(role: UserRole): string {
  switch (role) {
    case 'admin':
      return 'bg-destructive/10 text-destructive border-destructive/20'
    case 'warehouse_manager':
      return 'bg-amber-500/10 text-amber-700 border-amber-500/20'
    case 'picker':
      return 'bg-sky-500/10 text-sky-700 border-sky-500/20'
    default:
      return 'bg-muted text-muted-foreground'
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
    <div class="flex flex-wrap items-end justify-between gap-4">
      <div class="grid gap-1">
        <h1 class="text-2xl font-semibold tracking-tight">User Management</h1>
        <p class="text-sm text-muted-foreground">
          Manage system users, four-role RBAC permissions, and warehouse scoping.
        </p>
      </div>

      <Button v-if="isAdmin" @click="openAddUser">
        <UserPlus class="size-4" />
        Add User
      </Button>
    </div>

    <p
      v-if="!isAdmin"
      role="alert"
      class="rounded-md border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive"
    >
      Access Denied: User Management is strictly restricted to Administrator accounts.
    </p>

    <Card v-else>
      <CardContent class="grid gap-4">
        <div class="flex flex-wrap items-center gap-3">
          <div class="relative w-[260px]">
            <Search class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input v-model="searchQuery" placeholder="Search name or email..." class="pl-9" />
          </div>

          <Select v-model="roleFilter">
            <SelectTrigger class="w-[190px]">
              <SelectValue placeholder="All Roles" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Roles</SelectItem>
              <SelectItem value="admin">Admin</SelectItem>
              <SelectItem value="warehouse_manager">Warehouse Manager</SelectItem>
              <SelectItem value="picker">Picker</SelectItem>
              <SelectItem value="viewer">Viewer</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <p
          v-if="usersStore.error"
          role="alert"
          class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ usersStore.error }}
        </p>

        <div class="overflow-hidden rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Full Name</TableHead>
                <TableHead>Role</TableHead>
                <TableHead>Assigned Warehouse</TableHead>
                <TableHead>Status</TableHead>
                <TableHead class="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="usersStore.loading">
                <TableCell :colspan="5">
                  <div class="grid gap-2 py-2">
                    <Skeleton class="h-6 w-full" />
                    <Skeleton class="h-6 w-full" />
                    <Skeleton class="h-6 w-2/3" />
                  </div>
                </TableCell>
              </TableRow>

              <TableRow v-else-if="usersStore.users.length === 0">
                <TableCell :colspan="5" class="py-10 text-center text-sm text-muted-foreground">
                  No user accounts found.
                </TableCell>
              </TableRow>

              <TableRow v-for="user in usersStore.users" v-else :key="user.id">
                <TableCell>
                  <div class="grid leading-tight">
                    <strong class="text-sm">{{ user.full_name }}</strong>
                    <small class="text-xs text-muted-foreground">{{ user.email }}</small>
                  </div>
                </TableCell>
                <TableCell>
                  <Badge variant="outline" :class="['font-mono text-[10px] uppercase', getRoleClass(user.role)]">
                    {{ user.role.replace('_', ' ') }}
                  </Badge>
                </TableCell>
                <TableCell class="text-sm text-muted-foreground">
                  {{ getWarehouseName(user.warehouse_id) }}
                </TableCell>
                <TableCell>
                  <Badge :variant="user.is_active ? 'default' : 'secondary'">
                    {{ user.is_active ? 'Active' : 'Inactive' }}
                  </Badge>
                </TableCell>
                <TableCell class="text-right">
                  <div class="flex items-center justify-end gap-1">
                    <Button variant="ghost" size="icon" title="Edit user" @click="openEditUser(user)">
                      <Pencil class="size-4" />
                    </Button>
                    <Button variant="ghost" size="icon" title="Reset password" @click="openResetPassword(user)">
                      <KeyRound class="size-4" />
                    </Button>
                    <Button
                      v-if="user.is_active && user.id !== auth.user?.id"
                      variant="ghost"
                      size="icon"
                      class="text-destructive hover:text-destructive"
                      title="Deactivate user"
                      @click="deactivateUser(user)"
                    >
                      <UserMinus class="size-4" />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
      </CardContent>
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

    <ConfirmDialog
      v-model:open="confirmVisible"
      title="Deactivate this user?"
      :description="`${pendingUser?.full_name ?? ''} will lose access immediately and their refresh tokens are revoked. The account can be reactivated later.`"
      confirm-label="Deactivate"
      @confirm="confirmDeactivateUser"
    />
  </div>
</template>
