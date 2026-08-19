<script setup lang="ts">
import Button from 'primevue/button'
import Column from 'primevue/column'
import DataTable from 'primevue/datatable'
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'

import type { Location, Warehouse } from '@/api/warehouses'
import { getWarehouse } from '@/api/warehouses'
import LocationFormDialog from '@/components/LocationFormDialog.vue'
import WarehouseFormDialog from '@/components/WarehouseFormDialog.vue'
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
  <div class="flex flex-col gap-4">
    <p v-if="warehouseError" class="text-danger-text" data-testid="warehouse-error">{{ warehouseError }}</p>
    <template v-else-if="warehouse">
      <nav class="text-sm text-ink-faint">
        <RouterLink to="/warehouses" class="hover:underline">Warehouses</RouterLink>
        <span class="mx-1">›</span>
        <span class="text-ink">{{ warehouse.name }}</span>
      </nav>

      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-ink">{{ warehouse.name }}</h1>
          <p class="text-ink-muted">
            <span class="font-mono-code">{{ warehouse.code }}</span>
            · {{ warehouse.address ?? 'No address on file' }}
          </p>
        </div>
        <Button
          v-if="canManageWarehouses"
          label="Edit Warehouse"
          data-testid="edit-warehouse"
          @click="warehouseDialogVisible = true"
        />
      </div>

      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <h2 class="text-lg font-semibold text-ink">Locations</h2>
          <span class="rounded-full bg-surface px-2 py-0.5 text-xs font-medium text-ink-faint">
            {{ store.locations.length }} shown
          </span>
        </div>
        <Button
          v-if="canManageLocations"
          label="Add Location"
          data-testid="add-location"
          @click="openCreateLocationDialog"
        />
      </div>

      <p v-if="store.locationsError" class="text-danger-text" data-testid="locations-error">
        {{ store.locationsError }}
      </p>
      <DataTable
        v-else
        :value="store.locations"
        :loading="store.locationsLoading"
        data-key="id"
        class="overflow-hidden rounded-[12px] border border-border"
      >
        <template #empty>
          <p>No locations yet.</p>
        </template>
        <Column field="code" header="Location code">
          <template #body="{ data }">
            <span class="font-mono-code text-[0.85rem]">{{ data.code }}</span>
          </template>
        </Column>
        <Column field="zone" header="Zone" />
        <Column field="aisle" header="Aisle" />
        <Column field="rack" header="Rack" />
        <Column field="shelf" header="Shelf" />
        <Column field="barcode" header="Barcode">
          <template #body="{ data }">
            <span v-if="data.barcode" class="font-mono-code text-[0.85rem]">{{ data.barcode }}</span>
            <span v-else class="text-ink-faint">—</span>
          </template>
        </Column>
        <Column header="Pickable">
          <template #body="{ data }">
            <span
              class="rounded-badge px-2 py-1 text-xs font-medium"
              :class="data.is_pickable ? 'bg-success-bg text-success-text' : 'bg-danger-bg text-danger-text'"
            >{{ data.is_pickable ? 'Yes' : 'No' }}</span>
          </template>
        </Column>
        <Column header="Status">
          <template #body="{ data }">
            <span
              class="rounded-badge px-2 py-1 text-xs font-medium"
              :class="data.is_active ? 'bg-success-bg text-success-text' : 'bg-danger-bg text-danger-text'"
            >{{ data.is_active ? 'Active' : 'Inactive' }}</span>
          </template>
        </Column>
        <Column v-if="canManageLocations" header="Actions">
          <template #body="{ data }">
            <Button
              icon="pi pi-pencil"
              size="small"
              severity="secondary"
              text
              rounded
              aria-label="Edit location"
              data-testid="edit-location"
              @click="openEditLocationDialog(data)"
            />
            <Button
              v-if="data.is_active"
              icon="pi pi-ban"
              size="small"
              severity="danger"
              text
              rounded
              aria-label="Deactivate location"
              data-testid="deactivate-location"
              @click="deactivateLocationRow(data)"
            />
          </template>
        </Column>
      </DataTable>

      <Button
        v-if="store.locationsHasMore"
        label="Load more"
        severity="secondary"
        data-testid="load-more-locations"
        @click="store.loadMoreLocations(warehouseId)"
      />

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
