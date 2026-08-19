<script setup lang="ts">
import Button from 'primevue/button'
import Column from 'primevue/column'
import DataTable from 'primevue/datatable'
import InputText from 'primevue/inputtext'
import Tag from 'primevue/tag'
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'

import type { Warehouse } from '@/api/warehouses'
import WarehouseFormDialog from '@/components/WarehouseFormDialog.vue'
import { useAuthStore } from '@/stores/auth'
import { useWarehousesStore } from '@/stores/warehouses'

const store = useWarehousesStore()
const auth = useAuthStore()
const router = useRouter()

const search = ref('')
const dialogVisible = ref(false)
const editingWarehouse = ref<Warehouse | null>(null)

const canManageWarehouses = computed(() => auth.user?.role === 'admin')

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
  <div class="page">
    <div class="page-header">
      <h1>Warehouses</h1>
      <Button
        v-if="canManageWarehouses"
        label="Add Warehouse"
        data-testid="add-warehouse"
        @click="openCreateDialog"
      />
    </div>

    <InputText
      v-model="search"
      placeholder="Search warehouses..."
      data-testid="warehouse-search"
      class="search-field"
    />

    <p v-if="store.error" class="error-banner" data-testid="warehouses-error">{{ store.error }}</p>

    <DataTable
      v-else
      :value="store.warehouses"
      :loading="store.loading"
      data-key="id"
      @row-click="(event: { data: Warehouse }) => openDetail(event.data)"
    >
      <template #empty>
        <p>No warehouses yet.</p>
      </template>
      <Column field="code" header="Code" />
      <Column field="name" header="Name" />
      <Column field="address" header="Address" />
      <Column header="Status">
        <template #body="{ data }">
          <Tag :severity="data.is_active ? 'success' : 'danger'" :value="data.is_active ? 'Active' : 'Inactive'" />
        </template>
      </Column>
      <Column v-if="canManageWarehouses" header="Actions">
        <template #body="{ data }">
          <Button
            label="Edit"
            size="small"
            severity="secondary"
            data-testid="edit-warehouse"
            @click.stop="openEditDialog(data)"
          />
          <Button
            v-if="data.is_active"
            label="Deactivate"
            size="small"
            severity="danger"
            data-testid="deactivate-warehouse"
            @click.stop="deactivate(data)"
          />
        </template>
      </Column>
    </DataTable>

    <Button
      v-if="store.hasMore"
      label="Load more"
      severity="secondary"
      data-testid="load-more"
      @click="store.loadMoreWarehouses(search)"
    />

    <WarehouseFormDialog v-model:visible="dialogVisible" :warehouse="editingWarehouse" />
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
.search-field {
  max-width: 320px;
}
.error-banner {
  color: var(--p-red-600, #dc2626);
}
</style>
