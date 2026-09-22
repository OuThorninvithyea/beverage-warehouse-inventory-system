<script setup lang="ts">
import { Pencil, Plus, Search, Trash2 } from 'lucide-vue-next'
import { computed, onMounted, ref, watch } from 'vue'

import type { Category } from '@/api/catalog'
import CategoryFormDialog from '@/components/CategoryFormDialog.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
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
import { useCatalogStore } from '@/stores/catalog'

const auth = useAuthStore()
const catalogStore = useCatalogStore()

const searchQuery = ref('')
const categoryFormVisible = ref(false)
const selectedCategory = ref<Category | null>(null)
const pendingCategory = ref<Category | null>(null)
const confirmVisible = ref(false)

const canManageCatalog = computed(() => {
  return auth.user?.role === 'admin' || auth.user?.role === 'warehouse_manager'
})

onMounted(async () => {
  await catalogStore.fetchCategories(searchQuery.value)
})

watch(searchQuery, () => {
  void catalogStore.fetchCategories(searchQuery.value)
})

function openAddCategory() {
  selectedCategory.value = null
  categoryFormVisible.value = true
}

function openEditCategory(category: Category) {
  selectedCategory.value = category
  categoryFormVisible.value = true
}

function deactivateCategory(category: Category) {
  pendingCategory.value = category
  confirmVisible.value = true
}

async function confirmDeactivateCategory() {
  const category = pendingCategory.value
  confirmVisible.value = false
  if (!category) return
  await catalogStore.removeCategory(category.id)
  pendingCategory.value = null
}

function getParentCategoryName(parentId: string | null): string {
  if (!parentId) return 'Top Level'
  const parent = catalogStore.categories.find((c) => c.id === parentId)
  return parent ? parent.name : 'Top Level'
}
</script>

<template>
  <div class="grid gap-6">
    <div class="flex flex-wrap items-end justify-between gap-4">
      <div class="grid gap-1">
        <h1 class="text-2xl font-semibold tracking-tight">Product Categories</h1>
        <p class="text-sm text-muted-foreground">
          Organize catalog items by beverage type, brand, or package type.
        </p>
      </div>

      <Button v-if="canManageCatalog" @click="openAddCategory">
        <Plus class="size-4" />
        Add Category
      </Button>
    </div>

    <Card>
      <CardContent class="grid gap-4">
        <div class="relative max-w-sm">
          <Search class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input v-model="searchQuery" placeholder="Search categories..." class="pl-9" />
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
                <TableHead>Category Name</TableHead>
                <TableHead>Parent Category</TableHead>
                <TableHead>Status</TableHead>
                <TableHead v-if="canManageCatalog" class="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="catalogStore.loading">
                <TableCell :colspan="canManageCatalog ? 4 : 3">
                  <div class="grid gap-2 py-2">
                    <Skeleton class="h-6 w-full" />
                    <Skeleton class="h-6 w-full" />
                    <Skeleton class="h-6 w-2/3" />
                  </div>
                </TableCell>
              </TableRow>

              <TableRow v-else-if="catalogStore.categories.length === 0">
                <TableCell
                  :colspan="canManageCatalog ? 4 : 3"
                  class="py-10 text-center text-sm text-muted-foreground"
                >
                  No categories found.
                </TableCell>
              </TableRow>

              <TableRow v-for="category in catalogStore.categories" v-else :key="category.id">
                <TableCell class="font-medium">{{ category.name }}</TableCell>
                <TableCell class="text-sm text-muted-foreground">
                  {{ getParentCategoryName(category.parent_id) }}
                </TableCell>
                <TableCell>
                  <Badge :variant="category.is_active ? 'default' : 'secondary'">
                    {{ category.is_active ? 'Active' : 'Inactive' }}
                  </Badge>
                </TableCell>
                <TableCell v-if="canManageCatalog" class="text-right">
                  <div class="flex items-center justify-end gap-1">
                    <Button variant="ghost" size="icon" title="Edit category" @click="openEditCategory(category)">
                      <Pencil class="size-4" />
                    </Button>
                    <Button
                      v-if="category.is_active"
                      variant="ghost"
                      size="icon"
                      class="text-destructive hover:text-destructive"
                      title="Deactivate category"
                      @click="deactivateCategory(category)"
                    >
                      <Trash2 class="size-4" />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>

    <CategoryFormDialog
      v-model:visible="categoryFormVisible"
      :category="selectedCategory"
      :categories="catalogStore.categories"
      @saved="catalogStore.fetchCategories(searchQuery)"
    />

    <ConfirmDialog
      v-model:open="confirmVisible"
      title="Deactivate this category?"
      :description="`${pendingCategory?.name ?? ''} will be hidden from the catalog. Products keep their history and the category can be reactivated later.`"
      confirm-label="Deactivate"
      @confirm="confirmDeactivateCategory"
    />
  </div>
</template>
