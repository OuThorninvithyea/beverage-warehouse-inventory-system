<script setup lang="ts">
import Button from 'primevue/button'
import Card from 'primevue/card'
import Column from 'primevue/column'
import DataTable from 'primevue/datatable'
import Message from 'primevue/message'
import Select from 'primevue/select'
import Tag from 'primevue/tag'
import { computed, onMounted, ref, watch } from 'vue'

import AdjustFormDialog from '@/components/AdjustFormDialog.vue'
import PickFormDialog from '@/components/PickFormDialog.vue'
import ReceiveFormDialog from '@/components/ReceiveFormDialog.vue'
import TransferFormDialog from '@/components/TransferFormDialog.vue'
import { exportToCSV } from '@/lib/export'
import { useAuthStore } from '@/stores/auth'
import { useCatalogStore } from '@/stores/catalog'
import { useInventoryStore } from '@/stores/inventory'
import { useWarehousesStore } from '@/stores/warehouses'

const auth = useAuthStore()
const inventoryStore = useInventoryStore()
const catalogStore = useCatalogStore()
const warehouseStore = useWarehousesStore()

const movementTypeFilter = ref<'receive' | 'pick' | 'transfer' | 'adjust'>()
const selectedProductId = ref<string>()

const receiveVisible = ref(false)
const pickVisible = ref(false)
const transferVisible = ref(false)
const adjustVisible = ref(false)

function exportMovementsCSV() {
  exportToCSV('stock_movement_audit_log', inventoryStore.movements, [
    { key: 'created_at', label: 'Timestamp' },
    { key: 'type', label: 'Type' },
    { key: 'product_sku', label: 'SKU' },
    { key: 'product_name', label: 'Product Name' },
    { key: 'quantity', label: 'Quantity' },
    { key: 'from_location_code', label: 'From Location' },
    { key: 'to_location_code', label: 'To Location' },
    { key: 'reference', label: 'Reference' },
    { key: 'performer_name', label: 'Operator' },
  ])
}

const canReceiveOrPick = computed(() => {
  return auth.user?.role === 'admin' || auth.user?.role === 'warehouse_manager' || auth.user?.role === 'picker'
})

const canTransferOrAdjust = computed(() => {
  return auth.user?.role === 'admin' || auth.user?.role === 'warehouse_manager'
})

onMounted(async () => {
  await Promise.all([
    warehouseStore.fetchWarehouses(),
    warehouseStore.fetchLocations(),
    catalogStore.fetchProducts(),
    inventoryStore.fetchMovements(),
  ])
})

watch([movementTypeFilter, selectedProductId], () => {
  void inventoryStore.fetchMovements({
    movement_type: movementTypeFilter.value,
    product_id: selectedProductId.value,
  })
})

function getMovementSeverity(type: string): 'success' | 'warn' | 'info' | 'secondary' {
  switch (type) {
    case 'receive':
      return 'success'
    case 'pick':
      return 'warn'
    case 'transfer':
      return 'info'
    case 'adjust':
      return 'secondary'
    default:
      return 'secondary'
  }
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '—'
  const date = new Date(dateStr)
  return date.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}
</script>

<template>
  <div class="grid gap-6">
    <div class="flex flex-wrap items-center justify-between gap-4">
      <div>
        <h1 class="m-0 text-2xl font-bold text-brand-navy">Stock Movement Audit Log</h1>
        <p class="m-0 text-sm text-brand-muted">Immutable history of receives, FEFO picks, transfers, and adjustments</p>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button
          label="Export CSV"
          icon="pi pi-file-excel"
          severity="secondary"
          outlined
          @click="exportMovementsCSV"
        />
        <Button
          v-if="canReceiveOrPick"
          label="Receive"
          icon="pi pi-download"
          severity="success"
          @click="receiveVisible = true"
        />
        <Button
          v-if="canReceiveOrPick"
          label="FEFO Pick"
          icon="pi pi-upload"
          severity="warn"
          @click="pickVisible = true"
        />
        <Button
          v-if="canTransferOrAdjust"
          label="Transfer"
          icon="pi pi-arrows-h"
          severity="info"
          @click="transferVisible = true"
        />
        <Button
          v-if="canTransferOrAdjust"
          label="Adjust"
          icon="pi pi-sliders-h"
          severity="secondary"
          @click="adjustVisible = true"
        />
      </div>
    </div>

    <Card>
      <template #content>
        <div class="mb-4 flex flex-wrap items-center gap-3">
          <Select
            v-model="movementTypeFilter"
            :options="[
              { label: 'Receive', value: 'receive' },
              { label: 'Pick (FEFO)', value: 'pick' },
              { label: 'Transfer', value: 'transfer' },
              { label: 'Adjustment', value: 'adjust' },
            ]"
            option-label="label"
            option-value="value"
            placeholder="All Movement Types"
            show-clear
            class="w-[200px]"
          />

          <Select
            v-model="selectedProductId"
            :options="catalogStore.products"
            option-label="name"
            option-value="id"
            placeholder="All Products"
            show-clear
            filter
            class="w-[240px]"
          />
        </div>

        <Message v-if="inventoryStore.error" severity="error" class="mb-4">
          {{ inventoryStore.error }}
        </Message>

        <DataTable
          :value="inventoryStore.movements"
          :loading="inventoryStore.loading"
          data-key="id"
          responsive-layout="scroll"
          striped-rows
          paginator
          :rows="10"
          :rows-per-page-options="[10, 20, 50]"
          class="p-datatable-sm"
        >
          <template #empty>
            <div class="py-8 text-center text-brand-muted">
              No movement audit records found.
            </div>
          </template>

          <Column field="created_at" header="Timestamp" sortable>
            <template #body="{ data }">
              <span class="text-xs font-semibold text-slate-700">{{ formatDate(data.created_at) }}</span>
            </template>
          </Column>

          <Column field="type" header="Type" sortable>
            <template #body="{ data }">
              <Tag
                :value="data.type.toUpperCase()"
                :severity="getMovementSeverity(data.type)"
                class="font-mono text-[10px]"
              />
            </template>
          </Column>

          <Column field="product_name" header="Product">
            <template #body="{ data }">
              <div>
                <strong class="block text-brand-navy">{{ data.product_name || 'Product' }}</strong>
                <small class="font-mono text-brand-muted">SKU: {{ data.product_sku }}</small>
              </div>
            </template>
          </Column>

          <Column field="quantity" header="Quantity" sortable>
            <template #body="{ data }">
              <span class="font-extrabold text-brand-navy">{{ data.quantity }}</span>
            </template>
          </Column>

          <Column header="Locations">
            <template #body="{ data }">
              <div class="flex items-center gap-1 text-xs font-mono">
                <span v-if="data.from_location_code" class="rounded bg-red-50 text-red-700 px-1 border border-red-200">
                  {{ data.from_location_code }}
                </span>
                <i v-if="data.from_location_code && data.to_location_code" class="pi pi-arrow-right text-[10px] text-slate-400" />
                <span v-if="data.to_location_code" class="rounded bg-green-50 text-green-700 px-1 border border-green-200">
                  {{ data.to_location_code }}
                </span>
              </div>
            </template>
          </Column>

          <Column field="reference" header="Reference">
            <template #body="{ data }">
              <span v-if="data.reference" class="font-mono text-xs font-semibold text-slate-700">
                {{ data.reference }}
              </span>
              <span v-else class="text-xs text-slate-400">—</span>
            </template>
          </Column>

          <Column field="performed_by" header="Operator">
            <template #body="{ data }">
              <span class="text-xs text-slate-600 font-medium">
                {{ data.performer_name || 'System Operator' }}
              </span>
            </template>
          </Column>
        </DataTable>
      </template>
    </Card>

    <ReceiveFormDialog
      v-model:visible="receiveVisible"
      :locations="warehouseStore.locations"
      :products="catalogStore.products"
      @submitted="inventoryStore.fetchMovements()"
    />

    <PickFormDialog
      v-model:visible="pickVisible"
      :locations="warehouseStore.locations"
      :products="catalogStore.products"
      @submitted="inventoryStore.fetchMovements()"
    />

    <TransferFormDialog
      v-model:visible="transferVisible"
      :locations="warehouseStore.locations"
      :products="catalogStore.products"
      @submitted="inventoryStore.fetchMovements()"
    />

    <AdjustFormDialog
      v-model:visible="adjustVisible"
      :locations="warehouseStore.locations"
      :products="catalogStore.products"
      @submitted="inventoryStore.fetchMovements()"
    />
  </div>
</template>
