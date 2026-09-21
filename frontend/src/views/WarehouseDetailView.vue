<script setup lang="ts">
import { Ban, Pencil, Plus } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'

import type { Location, Warehouse } from '@/api/warehouses'
import { getWarehouse } from '@/api/warehouses'
import LocationFormDialog from '@/components/LocationFormDialog.vue'
import WarehouseFormDialog from '@/components/WarehouseFormDialog.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
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

const route = useRoute()
const store = useWarehousesStore()
const auth = useAuthStore()

const warehouseId = computed(() => route.params.warehouseId as string)
const warehouse = ref<Warehouse | null>(null)
const warehouseError = ref<string | null>(null)

const canManageWarehouses = computed(() => auth.user?.role === 'admin')
const canManageLocations = computed(
  () => auth.user?.role === 'admin' || auth.user?.role === 'warehouse_manager',
)
const columnCount = computed(() => (canManageLocations.value ? 9 : 8))

const warehouseDialogVisible = ref(false)
const locationDialogVisible = ref(false)
const editingLocation = ref<Location | null>(null)

async function loadWarehouse() {
  warehouseError.value = null
  try {
    warehouse.value = await getWarehouse(warehouseId.value)
  } catch (err) {
    warehouseError.value = err instanceof Error ? err.message : 'Failed to load warehouse.'
  }
}

onMounted(() => {
  loadWarehouse()
  store.fetchLocations(warehouseId.value)
})

watch(warehouseId, () => {
  loadWarehouse()
  store.fetchLocations(warehouseId.value)
})

function openCreateLocationDialog() {
  editingLocation.value = null
  locationDialogVisible.value = true
}

function openEditLocationDialog(location: Location) {
  editingLocation.value = location
  locationDialogVisible.value = true
}

async function deactivateLocationRow(location: Location) {
  await store.deactivateLocation(warehouseId.value, location.id)
}

function handleWarehouseSaved() {
  loadWarehouse()
}
</script>

<template>
  <div class="grid gap-6">
    <p
      v-if="warehouseError"
      role="alert"
      data-testid="warehouse-error"
      class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
    >
      {{ warehouseError }}
    </p>

    <template v-else-if="warehouse">
      <nav class="text-sm text-muted-foreground">
        <RouterLink to="/warehouses" class="hover:underline">Warehouses</RouterLink>
        <span class="mx-1">&rsaquo;</span>
        <span class="text-foreground">{{ warehouse.name }}</span>
      </nav>

      <div class="flex flex-wrap items-end justify-between gap-4">
        <div class="grid gap-1">
          <h1 class="text-2xl font-semibold tracking-tight">{{ warehouse.name }}</h1>
          <p class="text-sm text-muted-foreground">
            <span class="font-mono">{{ warehouse.code }}</span>
            &middot; {{ warehouse.address ?? 'No address on file' }}
          </p>
        </div>
        <Button
          v-if="canManageWarehouses"
          variant="outline"
          data-testid="edit-warehouse"
          @click="warehouseDialogVisible = true"
        >
          <Pencil class="size-4" />
          Edit Warehouse
        </Button>
      </div>

      <Card>
        <CardContent class="grid gap-4">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="flex items-center gap-2">
              <h2 class="text-lg font-semibold">Locations</h2>
              <Badge variant="secondary">{{ store.locations.length }} shown</Badge>
            </div>
            <Button v-if="canManageLocations" data-testid="add-location" @click="openCreateLocationDialog">
              <Plus class="size-4" />
              Add Location
            </Button>
          </div>

          <p
            v-if="store.locationsError"
            role="alert"
            data-testid="locations-error"
            class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
          >
            {{ store.locationsError }}
          </p>

          <div v-else class="overflow-x-auto rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Location code</TableHead>
                  <TableHead>Zone</TableHead>
                  <TableHead>Aisle</TableHead>
                  <TableHead>Rack</TableHead>
                  <TableHead>Shelf</TableHead>
                  <TableHead>Barcode</TableHead>
                  <TableHead>Pickable</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead v-if="canManageLocations" class="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-if="store.locationsLoading">
                  <TableCell :colspan="columnCount">
                    <div class="grid gap-2 py-2">
                      <Skeleton class="h-6 w-full" />
                      <Skeleton class="h-6 w-2/3" />
                    </div>
                  </TableCell>
                </TableRow>

                <TableRow v-else-if="store.locations.length === 0">
                  <TableCell :colspan="columnCount" class="py-10 text-center text-sm text-muted-foreground">
                    No locations yet.
                  </TableCell>
                </TableRow>

                <TableRow v-for="location in store.locations" v-else :key="location.id">
                  <TableCell class="font-mono text-sm">{{ location.code }}</TableCell>
                  <TableCell>{{ location.zone || '—' }}</TableCell>
                  <TableCell>{{ location.aisle || '—' }}</TableCell>
                  <TableCell>{{ location.rack || '—' }}</TableCell>
                  <TableCell>{{ location.shelf || '—' }}</TableCell>
                  <TableCell>
                    <span v-if="location.barcode" class="font-mono text-sm">{{ location.barcode }}</span>
                    <span v-else class="text-muted-foreground">—</span>
                  </TableCell>
                  <TableCell>
                    <Badge :variant="location.is_pickable ? 'default' : 'secondary'">
                      {{ location.is_pickable ? 'Yes' : 'No' }}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge :variant="location.is_active ? 'default' : 'secondary'">
                      {{ location.is_active ? 'Active' : 'Inactive' }}
                    </Badge>
                  </TableCell>
                  <TableCell v-if="canManageLocations" class="text-right">
                    <div class="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label="Edit location"
                        data-testid="edit-location"
                        @click="openEditLocationDialog(location)"
                      >
                        <Pencil class="size-4" />
                      </Button>
                      <Button
                        v-if="location.is_active"
                        variant="ghost"
                        size="icon"
                        class="text-destructive hover:text-destructive"
                        aria-label="Deactivate location"
                        data-testid="deactivate-location"
                        @click="deactivateLocationRow(location)"
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
            v-if="store.locationsHasMore"
            variant="outline"
            class="justify-self-start"
            data-testid="load-more-locations"
            @click="store.loadMoreLocations(warehouseId)"
          >
            Load more
          </Button>
        </CardContent>
      </Card>

      <WarehouseFormDialog
        v-model:visible="warehouseDialogVisible"
        :warehouse="warehouse"
        @saved="handleWarehouseSaved"
      />
      <LocationFormDialog
        v-model:visible="locationDialogVisible"
        :warehouse-id="warehouseId"
        :location="editingLocation"
      />
    </template>
  </div>
</template>
