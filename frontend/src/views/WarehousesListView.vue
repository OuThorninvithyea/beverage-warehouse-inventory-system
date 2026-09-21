<script setup lang="ts">
import { Ban, Pencil, Plus, Search } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'

import type { Warehouse } from '@/api/warehouses'
import WarehouseFormDialog from '@/components/WarehouseFormDialog.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
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
import { useWarehousesStore } from '@/stores/warehouses'

const store = useWarehousesStore()
const auth = useAuthStore()
const router = useRouter()

const search = ref('')
const dialogVisible = ref(false)
const editingWarehouse = ref<Warehouse | null>(null)

const canManageWarehouses = computed(() => auth.user?.role === 'admin')
const columnCount = computed(() => (canManageWarehouses.value ? 5 : 4))

onMounted(() => {
  store.fetchWarehouses()
})

let searchTimeout: ReturnType<typeof setTimeout> | undefined
watch(search, (value) => {
  if (searchTimeout) {
    clearTimeout(searchTimeout)
  }
  searchTimeout = setTimeout(() => {
    store.fetchWarehouses(value)
  }, 300)
})

function openCreateDialog() {
  editingWarehouse.value = null
  dialogVisible.value = true
}

function openEditDialog(warehouse: Warehouse) {
  editingWarehouse.value = warehouse
  dialogVisible.value = true
}

async function deactivate(warehouse: Warehouse) {
  await store.deactivateWarehouse(warehouse.id)
}

function openDetail(warehouse: Warehouse) {
  router.push({ name: 'warehouse-detail', params: { warehouseId: warehouse.id } })
}
</script>

<template>
  <div class="grid gap-6">
    <div class="flex flex-wrap items-end justify-between gap-4">
      <div class="grid gap-1">
        <h1 class="text-2xl font-semibold tracking-tight">Warehouses</h1>
        <p class="text-sm text-muted-foreground">
          Facilities, storage locations, and warehouse scoping.
        </p>
      </div>

      <Button v-if="canManageWarehouses" data-testid="add-warehouse" @click="openCreateDialog">
        <Plus class="size-4" />
        Add Warehouse
      </Button>
    </div>

    <Card>
      <CardContent class="grid gap-4">
        <div class="relative max-w-sm">
          <Search class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            v-model="search"
            placeholder="Search warehouses..."
            data-testid="warehouse-search"
            class="pl-9"
          />
        </div>

        <p
          v-if="store.error"
          role="alert"
          data-testid="warehouses-error"
          class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ store.error }}
        </p>

        <div v-else class="overflow-hidden rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Code</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Address</TableHead>
                <TableHead>Status</TableHead>
                <TableHead v-if="canManageWarehouses" class="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="store.loading">
                <TableCell :colspan="columnCount">
                  <div class="grid gap-2 py-2">
                    <Skeleton class="h-6 w-full" />
                    <Skeleton class="h-6 w-2/3" />
                  </div>
                </TableCell>
              </TableRow>

              <TableRow v-else-if="store.warehouses.length === 0">
                <TableCell :colspan="columnCount" class="py-10 text-center text-sm text-muted-foreground">
                  No warehouses yet.
                </TableCell>
              </TableRow>

              <TableRow
                v-for="warehouse in store.warehouses"
                v-else
                :key="warehouse.id"
                class="cursor-pointer"
                @click="openDetail(warehouse)"
              >
                <TableCell class="font-mono text-sm">{{ warehouse.code }}</TableCell>
                <TableCell class="font-medium">{{ warehouse.name }}</TableCell>
                <TableCell class="text-sm text-muted-foreground">
                  {{ warehouse.address || '—' }}
                </TableCell>
                <TableCell>
                  <Badge :variant="warehouse.is_active ? 'default' : 'secondary'">
                    {{ warehouse.is_active ? 'Active' : 'Inactive' }}
                  </Badge>
                </TableCell>
                <TableCell v-if="canManageWarehouses" class="text-right">
                  <div class="flex items-center justify-end gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label="Edit warehouse"
                      data-testid="edit-warehouse"
                      @click.stop="openEditDialog(warehouse)"
                    >
                      <Pencil class="size-4" />
                    </Button>
                    <Button
                      v-if="warehouse.is_active"
                      variant="ghost"
                      size="icon"
                      class="text-destructive hover:text-destructive"
                      aria-label="Deactivate warehouse"
                      data-testid="deactivate-warehouse"
                      @click.stop="deactivate(warehouse)"
                    >
                      <Ban class="size-4" />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>

        <Button
          v-if="store.hasMore"
          variant="outline"
          class="justify-self-start"
          data-testid="load-more"
          @click="store.loadMoreWarehouses(search)"
        >
          Load more
        </Button>
      </CardContent>
    </Card>

    <WarehouseFormDialog v-model:visible="dialogVisible" :warehouse="editingWarehouse" />
  </div>
</template>
