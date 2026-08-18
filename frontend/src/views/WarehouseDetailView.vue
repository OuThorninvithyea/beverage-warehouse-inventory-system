<script setup lang="ts">
import Button from 'primevue/button'
import Column from 'primevue/column'
import DataTable from 'primevue/datatable'
import Tag from 'primevue/tag'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'

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
  <div class="page">
    <p v-if="warehouseError" class="error-banner" data-testid="warehouse-error">{{ warehouseError }}</p>
    <template v-else-if="warehouse">
      <div class="page-header">
        <div>
          <h1>{{ warehouse.name }}</h1>
          <p>{{ warehouse.code }} · {{ warehouse.address ?? 'No address on file' }}</p>
        </div>
        <Button
          v-if="canManageWarehouses"
          label="Edit Warehouse"
          data-testid="edit-warehouse"
          @click="warehouseDialogVisible = true"
        />
      </div>

      <div class="page-header">
        <h2>Locations</h2>
        <Button
          v-if="canManageLocations"
          label="Add Location"
          data-testid="add-location"
          @click="openCreateLocationDialog"
        />
      </div>

      <p v-if="store.locationsError" class="error-banner" data-testid="locations-error">
        {{ store.locationsError }}
      </p>
      <DataTable v-else :value="store.locations" :loading="store.locationsLoading" data-key="id">
        <template #empty>
          <p>No locations yet.</p>
        </template>
        <Column field="code" header="Code" />
        <Column field="zone" header="Zone" />
        <Column field="aisle" header="Aisle" />
        <Column field="rack" header="Rack" />
        <Column field="shelf" header="Shelf" />
        <Column field="barcode" header="Barcode" />
        <Column header="Pickable">
          <template #body="{ data }">
            <Tag :severity="data.is_pickable ? 'info' : 'secondary'" :value="data.is_pickable ? 'Yes' : 'No'" />
          </template>
        </Column>
        <Column header="Status">
          <template #body="{ data }">
            <Tag :severity="data.is_active ? 'success' : 'danger'" :value="data.is_active ? 'Active' : 'Inactive'" />
          </template>
        </Column>
        <Column v-if="canManageLocations" header="Actions">
          <template #body="{ data }">
            <Button
              label="Edit"
              size="small"
              severity="secondary"
              data-testid="edit-location"
              @click="openEditLocationDialog(data)"
            />
            <Button
              v-if="data.is_active"
              label="Deactivate"
              size="small"
              severity="danger"
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

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.error-banner {
  color: var(--p-red-600, #dc2626);
}
</style>
