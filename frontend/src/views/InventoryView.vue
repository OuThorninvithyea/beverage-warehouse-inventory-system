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

const selectedWarehouseId = ref<string>()
const selectedLocationId = ref<string>()
const selectedProductId = ref<string>()

const receiveVisible = ref(false)
const pickVisible = ref(false)
const transferVisible = ref(false)
const adjustVisible = ref(false)

function exportInventoryCSV() {
  exportToCSV('inventory_stock_balances', inventoryStore.balances, [
    { key: 'location_code', label: 'Location' },
    { key: 'product_sku', label: 'SKU' },
    { key: 'product_name', label: 'Product Name' },
    { key: 'lot_number', label: 'Lot Number' },
    { key: 'expiration_date', label: 'Expiration Date' },
    { key: 'quantity', label: 'Available Quantity' },
    { key: 'reserved_qty', label: 'Reserved Quantity' },
  ])
}

const canReceiveOrPick = computed(() => {
  return auth.user?.role === 'admin' || auth.user?.role === 'warehouse_manager' || auth.user?.role === 'picker'
})

const canTransferOrAdjust = computed(() => {
  return auth.user?.role === 'admin' || auth.user?.role === 'warehouse_manager'
})

const currentWarehouseLocations = computed(() => {
  if (!selectedWarehouseId.value) return warehouseStore.locations
  return warehouseStore.locations.filter((l) => l.warehouse_id === selectedWarehouseId.value)
})

onMounted(async () => {
  await Promise.all([
    warehouseStore.fetchWarehouses(),
    catalogStore.fetchProducts(),
  ])

  if (warehouseStore.warehouses.length > 0) {
    selectedWarehouseId.value = warehouseStore.warehouses[0].id
    await warehouseStore.fetchLocations(selectedWarehouseId.value)
  }

  await fetchBalances()
})

watch(selectedWarehouseId, async (newWhId) => {
  selectedLocationId.value = undefined
  if (newWhId) {
    await warehouseStore.fetchLocations(newWhId)
  }
  await fetchBalances()
})

watch([selectedLocationId, selectedProductId], () => {
  void fetchBalances()
})

async function fetchBalances() {
  await inventoryStore.fetchBalances({
    warehouse_id: selectedWarehouseId.value,
    location_id: selectedLocationId.value,
    product_id: selectedProductId.value,
  })
}

function isExpiringSoon(expirationDateStr?: string | null): boolean {
  if (!expirationDateStr) return false
  const expDate = new Date(expirationDateStr)
  const today = new Date()
  const diffDays = Math.ceil((expDate.getTime() - today.getTime()) / (1000 * 3600 * 24))
  return diffDays <= 30
}

function isLowStock(qtyStr: string): boolean {
  const q = parseFloat(qtyStr)
  return q < 10
}
</script>

<template>
  <div class="grid gap-6">
    <div class="flex flex-wrap items-center justify-between gap-4">
      <div>
        <h1 class="m-0 text-2xl font-bold text-brand-navy">Inventory Stock Control</h1>
        <p class="m-0 text-sm text-brand-muted">Real-time balances, lot FEFO expiry tracking, and movement actions</p>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button
          label="Export CSV"
          icon="pi pi-file-excel"
          severity="secondary"
          outlined
          @click="exportInventoryCSV"
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
        <div class="mb-4 flex flex-wrap items-center justify-between gap-4">
          <div class="flex flex-wrap items-center gap-3">
            <Select
              v-model="selectedWarehouseId"
              :options="warehouseStore.warehouses"
              option-label="name"
              option-value="id"
              placeholder="Select Warehouse"
              class="w-[220px]"
            />

            <Select
              v-model="selectedLocationId"
              :options="currentWarehouseLocations"
              option-label="code"
              option-value="id"
              placeholder="All Locations"
              show-clear
              class="w-[180px]"
            />

            <Select
              v-model="selectedProductId"
              :options="catalogStore.products"
              option-label="name"
              option-value="id"
              placeholder="All Products"
              show-clear
              filter
              class="w-[220px]"
            />
          </div>
        </div>

        <Message v-if="inventoryStore.error" severity="error" class="mb-4">
          {{ inventoryStore.error }}
        </Message>

        <DataTable
          :value="inventoryStore.balances"
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
              No stock balances found for the selected location or filters.
            </div>
          </template>

          <Column field="location_code" header="Location" sortable>
            <template #body="{ data }">
              <span class="inline-flex items-center gap-1 rounded bg-slate-100 px-2 py-0.5 font-mono text-xs font-bold text-slate-800 border border-slate-200">
                <i class="pi pi-map-marker text-[10px] text-brand-navy" />
                {{ data.location_code || data.location_id.substring(0, 8) }}
              </span>
            </template>
          </Column>

          <Column field="product_name" header="Product" sortable>
            <template #body="{ data }">
              <div>
                <strong class="block text-brand-navy">{{ data.product_name || 'Product' }}</strong>
                <small class="font-mono text-brand-muted">SKU: {{ data.product_sku }}</small>
              </div>
            </template>
          </Column>

          <Column field="lot_number" header="Lot / Expiration">
            <template #body="{ data }">
              <div v-if="data.lot_number" class="grid gap-0.5">
                <span class="font-mono text-xs font-semibold text-slate-800">
                  Lot: {{ data.lot_number }}
                </span>
                <span
                  v-if="data.expiration_date"
                  class="text-[11px] font-medium"
                  :class="isExpiringSoon(data.expiration_date) ? 'text-amber-600 font-bold' : 'text-slate-500'"
                >
                  Exp: {{ data.expiration_date }}
                  <span v-if="isExpiringSoon(data.expiration_date)" class="ml-1 inline-flex items-center text-[10px] bg-amber-100 text-amber-800 rounded px-1">
                    <i class="pi pi-exclamation-triangle mr-0.5 text-[9px]" /> Expiring
                  </span>
                </span>
              </div>
              <span v-else class="text-xs text-slate-400">Non-lot item</span>
            </template>
          </Column>

          <Column field="quantity" header="Available Qty" sortable>
            <template #body="{ data }">
              <div class="flex items-center gap-2">
                <span class="text-base font-extrabold text-brand-navy">{{ data.quantity }}</span>
                <Tag
                  v-if="isLowStock(data.quantity)"
                  value="Low Stock"
                  severity="warn"
                  class="text-[10px]"
                />
              </div>
            </template>
          </Column>

          <Column field="reserved_qty" header="Reserved Qty">
            <template #body="{ data }">
              <span class="text-xs font-medium text-slate-500">{{ data.reserved_qty || '0.000' }}</span>
            </template>
          </Column>
        </DataTable>
      </template>
    </Card>

    <ReceiveFormDialog
      v-model:visible="receiveVisible"
      :locations="currentWarehouseLocations"
      :products="catalogStore.products"
      @submitted="fetchBalances"
    />

    <PickFormDialog
      v-model:visible="pickVisible"
      :locations="currentWarehouseLocations"
      :products="catalogStore.products"
      @submitted="fetchBalances"
    />

    <TransferFormDialog
      v-model:visible="transferVisible"
      :locations="currentWarehouseLocations"
      :products="catalogStore.products"
      @submitted="fetchBalances"
    />

    <AdjustFormDialog
      v-model:visible="adjustVisible"
      :locations="currentWarehouseLocations"
      :products="catalogStore.products"
      @submitted="fetchBalances"
    />
  </div>
</template>
