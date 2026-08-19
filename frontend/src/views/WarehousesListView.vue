<script setup lang="ts">
import Button from 'primevue/button'
import Column from 'primevue/column'
import DataTable from 'primevue/datatable'
import InputText from 'primevue/inputtext'
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
  <div class="flex flex-col gap-4">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-semibold text-ink">Warehouses</h1>
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
      class="max-w-[320px]"
    />

    <p v-if="store.error" class="text-danger-text" data-testid="warehouses-error">{{ store.error }}</p>

    <DataTable
      v-else
      :value="store.warehouses"
      :loading="store.loading"
      data-key="id"
      class="overflow-hidden rounded-[12px] border border-border"
      @row-click="(event: { data: Warehouse }) => openDetail(event.data)"
    >
      <template #empty>
        <p>No warehouses yet.</p>
      </template>
      <Column field="code" header="Code">
        <template #body="{ data }">
          <span class="font-mono-code text-[0.85rem]">{{ data.code }}</span>
        </template>
      </Column>
      <Column field="name" header="Name" />
      <Column field="address" header="Address" />
      <Column header="Status">
        <template #body="{ data }">
          <span
            class="rounded-badge px-2 py-1 text-xs font-medium"
            :class="data.is_active ? 'bg-success-bg text-success-text' : 'bg-danger-bg text-danger-text'"
          >{{ data.is_active ? 'Active' : 'Inactive' }}</span>
        </template>
      </Column>
      <Column v-if="canManageWarehouses" header="Actions">
        <template #body="{ data }">
          <Button
            icon="pi pi-pencil"
            size="small"
            severity="secondary"
            text
            rounded
            aria-label="Edit warehouse"
            data-testid="edit-warehouse"
            @click.stop="openEditDialog(data)"
          />
          <Button
            v-if="data.is_active"
            icon="pi pi-ban"
            size="small"
            severity="danger"
            text
            rounded
            aria-label="Deactivate warehouse"
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
