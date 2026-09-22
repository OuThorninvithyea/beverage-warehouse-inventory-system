<script setup lang="ts">
import { Barcode, Camera, Pencil, Printer, Plus, Search, Trash2 } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'

import type { Product } from '@/api/catalog'
import BarcodePrintModal from '@/components/BarcodePrintModal.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import CursorPager from '@/components/CursorPager.vue'
import BarcodeScannerModal from '@/components/BarcodeScannerModal.vue'
import ProductFormDialog from '@/components/ProductFormDialog.vue'
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
import { useCursorPages } from '@/lib/cursor-pages'
import { useAuthStore } from '@/stores/auth'
import { useCatalogStore } from '@/stores/catalog'

const auth = useAuthStore()
const catalogStore = useCatalogStore()

const searchQuery = ref('')
const categoryFilter = ref<string>('all')
const statusFilter = ref<'active' | 'inactive'>('active')

const selectedCategoryId = computed(() =>
  categoryFilter.value === 'all' ? undefined : categoryFilter.value,
)
const activeFilter = computed(() => statusFilter.value === 'active')

const productFormVisible = ref(false)
const selectedProduct = ref<Product | null>(null)
const barcodeScannerVisible = ref(false)
const barcodePrintVisible = ref(false)
const pendingProduct = ref<Product | null>(null)
const confirmVisible = ref(false)

function openPrintLabel(product: Product) {
  selectedProduct.value = product
  barcodePrintVisible.value = true
}

const canManageCatalog = computed(() => {
  return auth.user?.role === 'admin' || auth.user?.role === 'warehouse_manager'
})

const columnCount = computed(() => 7)

const pager = useCursorPages((after) =>
  catalogStore.fetchProducts(
    searchQuery.value,
    selectedCategoryId.value,
    activeFilter.value,
    after,
  ),
)

onMounted(async () => {
  await Promise.all([catalogStore.fetchCategories(), pager.reset()])
})

// Any filter change invalidates the cursor history, so paging restarts.
watch([searchQuery, selectedCategoryId, activeFilter], () => {
  void pager.reset()
})

function openAddProduct() {
  selectedProduct.value = null
  productFormVisible.value = true
}

function openEditProduct(product: Product) {
  selectedProduct.value = product
  productFormVisible.value = true
}

function deactivateProduct(product: Product) {
  pendingProduct.value = product
  confirmVisible.value = true
}

async function confirmDeactivateProduct() {
  const product = pendingProduct.value
  confirmVisible.value = false
  if (!product) return
  await catalogStore.removeProduct(product.id)
  pendingProduct.value = null
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
    <div class="flex flex-wrap items-end justify-between gap-4">
      <div class="grid gap-1">
        <h1 class="text-2xl font-semibold tracking-tight">Products Catalog</h1>
        <p class="text-sm text-muted-foreground">
          Manage SKUs, barcodes, categories, and lot tracking properties.
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button variant="outline" @click="barcodeScannerVisible = true">
          <Camera class="size-4" />
          Scan Barcode
        </Button>
        <Button v-if="canManageCatalog" @click="openAddProduct">
          <Plus class="size-4" />
          Add Product
        </Button>
      </div>
    </div>

    <Card>
      <CardContent class="grid gap-4">
        <div class="flex flex-wrap items-center gap-3">
          <div class="relative min-w-[260px] flex-1 max-w-sm">
            <Search class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input v-model="searchQuery" placeholder="Search SKU, name, or barcode..." class="pl-9" />
          </div>

          <Select v-model="categoryFilter">
            <SelectTrigger class="w-[200px]">
              <SelectValue placeholder="All Categories" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Categories</SelectItem>
              <SelectItem v-for="cat in catalogStore.categories" :key="cat.id" :value="cat.id">
                {{ cat.name }}
              </SelectItem>
            </SelectContent>
          </Select>

          <Select v-model="statusFilter">
            <SelectTrigger class="w-[160px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="active">Active Items</SelectItem>
              <SelectItem value="inactive">Inactive Items</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <p
          v-if="catalogStore.error"
          role="alert"
          class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ catalogStore.error }}
        </p>

        <div class="overflow-hidden rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>SKU</TableHead>
                <TableHead>Product Name</TableHead>
                <TableHead>Barcode</TableHead>
                <TableHead>Unit</TableHead>
                <TableHead>Lot Tracking</TableHead>
                <TableHead>Status</TableHead>
                <TableHead class="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="catalogStore.loading">
                <TableCell :colspan="columnCount">
                  <div class="grid gap-2 py-2">
                    <Skeleton class="h-6 w-full" />
                    <Skeleton class="h-6 w-full" />
                    <Skeleton class="h-6 w-2/3" />
                  </div>
                </TableCell>
              </TableRow>

              <TableRow v-else-if="catalogStore.products.length === 0">
                <TableCell :colspan="columnCount" class="py-10 text-center text-sm text-muted-foreground">
                  No products found matching your search.
                </TableCell>
              </TableRow>

              <TableRow v-for="product in catalogStore.products" v-else :key="product.id">
                <TableCell class="font-mono text-sm font-semibold">{{ product.sku }}</TableCell>
                <TableCell>
                  <div class="grid leading-tight">
                    <strong class="text-sm">{{ product.name }}</strong>
                    <small class="text-xs text-muted-foreground">{{ getCategoryName(product.category_id) }}</small>
                  </div>
                </TableCell>
                <TableCell>
                  <span
                    v-if="product.barcode"
                    class="inline-flex items-center gap-1 rounded border bg-muted px-2 py-0.5 font-mono text-xs"
                  >
                    <Barcode class="size-3" />
                    {{ product.barcode }}
                  </span>
                  <span v-else class="text-xs text-muted-foreground">&mdash;</span>
                </TableCell>
                <TableCell>
                  <Badge variant="secondary" class="capitalize">{{ product.unit }}</Badge>
                </TableCell>
                <TableCell>
                  <Badge :variant="product.is_lot_tracked ? 'default' : 'outline'">
                    {{ product.is_lot_tracked ? 'FEFO Lot Tracked' : 'Standard' }}
                  </Badge>
                </TableCell>
                <TableCell>
                  <Badge :variant="product.is_active ? 'default' : 'secondary'">
                    {{ product.is_active ? 'Active' : 'Inactive' }}
                  </Badge>
                </TableCell>
                <TableCell class="text-right">
                  <div class="flex items-center justify-end gap-1">
                    <Button variant="ghost" size="icon" title="Print barcode label" @click="openPrintLabel(product)">
                      <Printer class="size-4" />
                    </Button>
                    <Button
                      v-if="canManageCatalog"
                      variant="ghost"
                      size="icon"
                      title="Edit product"
                      @click="openEditProduct(product)"
                    >
                      <Pencil class="size-4" />
                    </Button>
                    <Button
                      v-if="canManageCatalog && product.is_active"
                      variant="ghost"
                      size="icon"
                      class="text-destructive hover:text-destructive"
                      title="Deactivate product"
                      @click="deactivateProduct(product)"
                    >
                      <Trash2 class="size-4" />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>

          <CursorPager
            :page-number="pager.pageNumber"
            :can-go-back="pager.canGoBack"
            :can-go-forward="pager.canGoForward"
            :busy="pager.busy || catalogStore.loading"
            :item-count="catalogStore.products.length"
            @previous="pager.previous()"
            @next="pager.next()"
          />
      </CardContent>
    </Card>

    <ProductFormDialog
      v-model:visible="productFormVisible"
      :product="selectedProduct"
      :categories="catalogStore.categories"
      @saved="pager.reset()"
    />

    <BarcodeScannerModal v-model:visible="barcodeScannerVisible" @select="onBarcodeScanned" />

    <BarcodePrintModal v-model:visible="barcodePrintVisible" :product="selectedProduct" />

    <ConfirmDialog
      v-model:open="confirmVisible"
      title="Deactivate this product?"
      :description="`${pendingProduct?.name ?? ''} will stop appearing in lookups and new movements. Existing stock and history are kept.`"
      confirm-label="Deactivate"
      @confirm="confirmDeactivateProduct"
    />
  </div>
</template>
