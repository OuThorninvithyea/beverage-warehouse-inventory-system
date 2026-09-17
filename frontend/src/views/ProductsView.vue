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

import type { Product } from '@/api/catalog'
import BarcodePrintModal from '@/components/BarcodePrintModal.vue'
import BarcodeScannerModal from '@/components/BarcodeScannerModal.vue'
import ProductFormDialog from '@/components/ProductFormDialog.vue'
import { useAuthStore } from '@/stores/auth'
import { useCatalogStore } from '@/stores/catalog'

const auth = useAuthStore()
const catalogStore = useCatalogStore()

const searchQuery = ref('')
const selectedCategoryId = ref<string>()
const activeFilter = ref<boolean>(true)

const productFormVisible = ref(false)
const selectedProduct = ref<Product | null>(null)
const barcodeScannerVisible = ref(false)
const barcodePrintVisible = ref(false)

function openPrintLabel(product: Product) {
  selectedProduct.value = product
  barcodePrintVisible.value = true
}

const canManageCatalog = computed(() => {
  return auth.user?.role === 'admin' || auth.user?.role === 'warehouse_manager'
})

onMounted(async () => {
  await Promise.all([
    catalogStore.fetchCategories(),
    catalogStore.fetchProducts(searchQuery.value, selectedCategoryId.value, activeFilter.value),
  ])
})

watch([searchQuery, selectedCategoryId, activeFilter], () => {
  void catalogStore.fetchProducts(searchQuery.value, selectedCategoryId.value, activeFilter.value)
})

function openAddProduct() {
  selectedProduct.value = null
  productFormVisible.value = true
}

function openEditProduct(product: Product) {
  selectedProduct.value = product
  productFormVisible.value = true
}

async function deactivateProduct(product: Product) {
  if (confirm(`Are you sure you want to deactivate "${product.name}"?`)) {
    await catalogStore.removeProduct(product.id)
  }
}

function onBarcodeScanned(code: string) {
  searchQuery.value = code
}

function getCategoryName(catId: string | null): string {
  if (!catId) return 'Uncategorized'
  const cat = catalogStore.categories.find((c) => c.id === catId)
  return cat ? cat.name : 'Uncategorized'
}
</script>

<template>
  <div class="grid gap-6">
    <div class="flex flex-wrap items-center justify-between gap-4">
      <div>
        <h1 class="m-0 text-2xl font-bold text-brand-navy">Products Catalog</h1>
        <p class="m-0 text-sm text-brand-muted">Manage SKUs, barcodes, categories, and lot tracking properties</p>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button
          label="Scan Barcode"
          icon="pi pi-camera"
          severity="secondary"
          outlined
          @click="barcodeScannerVisible = true"
        />
        <Button
          v-if="canManageCatalog"
          label="Add Product"
          icon="pi pi-plus"
          @click="openAddProduct"
        />
      </div>
    </div>

    <Card>
      <template #content>
        <div class="mb-4 flex flex-wrap items-center justify-between gap-4">
          <div class="flex flex-wrap items-center gap-3">
            <span class="p-input-icon-left min-w-[240px]">
              <InputText
                v-model="searchQuery"
                placeholder="Search SKU, name, or barcode..."
                class="w-full"
              />
            </span>

            <Select
              v-model="selectedCategoryId"
              :options="catalogStore.categories"
              option-label="name"
              option-value="id"
              placeholder="All Categories"
              show-clear
              class="w-[200px]"
            />

            <Select
              v-model="activeFilter"
              :options="[
                { label: 'Active Items', value: true },
                { label: 'Inactive Items', value: false },
              ]"
              option-label="label"
              option-value="value"
              class="w-[160px]"
            />
          </div>
        </div>

        <Message v-if="catalogStore.error" severity="error" class="mb-4">
          {{ catalogStore.error }}
        </Message>

        <DataTable
          :value="catalogStore.products"
          :loading="catalogStore.loading"
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
              No products found matching your search.
            </div>
          </template>

          <Column field="sku" header="SKU" sortable>
            <template #body="{ data }">
              <span class="font-mono font-bold text-brand-navy">{{ data.sku }}</span>
            </template>
          </Column>

          <Column field="name" header="Product Name" sortable>
            <template #body="{ data }">
              <div>
                <strong class="block text-brand-ink">{{ data.name }}</strong>
                <small class="text-brand-muted">{{ getCategoryName(data.category_id) }}</small>
              </div>
            </template>
          </Column>

          <Column field="barcode" header="Barcode">
            <template #body="{ data }">
              <span v-if="data.barcode" class="inline-flex items-center gap-1 rounded bg-slate-100 px-2 py-0.5 font-mono text-xs text-slate-800 border border-slate-200">
                <i class="pi pi-barcode text-[10px]" />
                {{ data.barcode }}
              </span>
              <span v-else class="text-xs text-slate-400">—</span>
            </template>
          </Column>

          <Column field="unit" header="Unit" sortable>
            <template #body="{ data }">
              <Tag :value="data.unit" severity="secondary" class="capitalize" />
            </template>
          </Column>

          <Column field="is_lot_tracked" header="Lot Tracking">
            <template #body="{ data }">
              <Tag
                :value="data.is_lot_tracked ? 'FEFO Lot Tracked' : 'Standard'"
                :severity="data.is_lot_tracked ? 'info' : 'secondary'"
              />
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
              <div class="flex items-center gap-1">
                <Button
                  icon="pi pi-print"
                  severity="success"
                  text
                  rounded
                  size="small"
                  title="Print barcode label"
                  @click="openPrintLabel(data)"
                />
                <Button
                  v-if="canManageCatalog"
                  icon="pi pi-pencil"
                  severity="secondary"
                  text
                  rounded
                  size="small"
                  title="Edit product"
                  @click="openEditProduct(data)"
                />
                <Button
                  v-if="canManageCatalog && data.is_active"
                  icon="pi pi-trash"
                  severity="danger"
                  text
                  rounded
                  size="small"
                  title="Deactivate product"
                  @click="deactivateProduct(data)"
                />
              </div>
            </template>
          </Column>
        </DataTable>
      </template>
    </Card>

    <ProductFormDialog
      v-model:visible="productFormVisible"
      :product="selectedProduct"
      :categories="catalogStore.categories"
      @saved="catalogStore.fetchProducts(searchQuery, selectedCategoryId, activeFilter)"
    />

    <BarcodeScannerModal
      v-model:visible="barcodeScannerVisible"
      @select="onBarcodeScanned"
    />

    <BarcodePrintModal
      v-model:visible="barcodePrintVisible"
      :product="selectedProduct"
    />
  </div>
</template>
