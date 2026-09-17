<script setup lang="ts">
import Button from 'primevue/button'
import Card from 'primevue/card'
import Column from 'primevue/column'
import DataTable from 'primevue/datatable'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'
import Tag from 'primevue/tag'
import { computed, onMounted, ref, watch } from 'vue'

import type { Category } from '@/api/catalog'
import CategoryFormDialog from '@/components/CategoryFormDialog.vue'
import { useAuthStore } from '@/stores/auth'
import { useCatalogStore } from '@/stores/catalog'

const auth = useAuthStore()
const catalogStore = useCatalogStore()

const searchQuery = ref('')
const categoryFormVisible = ref(false)
const selectedCategory = ref<Category | null>(null)

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

async function deactivateCategory(category: Category) {
  if (confirm(`Are you sure you want to deactivate "${category.name}"?`)) {
    await catalogStore.removeCategory(category.id)
  }
}

function getParentCategoryName(parentId: string | null): string {
  if (!parentId) return 'Top Level'
  const parent = catalogStore.categories.find((c) => c.id === parentId)
  return parent ? parent.name : 'Top Level'
}
</script>

<template>
  <div class="grid gap-6">
    <div class="flex flex-wrap items-center justify-between gap-4">
      <div>
        <h1 class="m-0 text-2xl font-bold text-brand-navy">Product Categories</h1>
        <p class="m-0 text-sm text-brand-muted">Organize catalog items by beverage type, brand, or package type</p>
      </div>

      <Button
        v-if="canManageCatalog"
        label="Add Category"
        icon="pi pi-plus"
        @click="openAddCategory"
      />
    </div>

    <Card>
      <template #content>
        <div class="mb-4 flex items-center justify-between gap-4">
          <InputText
            v-model="searchQuery"
            placeholder="Search categories..."
            class="w-full max-w-[320px]"
          />
        </div>

        <Message v-if="catalogStore.error" severity="error" class="mb-4">
          {{ catalogStore.error }}
        </Message>

        <DataTable
          :value="catalogStore.categories"
          :loading="catalogStore.loading"
          data-key="id"
          responsive-layout="scroll"
          striped-rows
          paginator
          :rows="10"
          class="p-datatable-sm"
        >
          <template #empty>
            <div class="py-8 text-center text-brand-muted">
              No categories found.
            </div>
          </template>

          <Column field="name" header="Category Name" sortable>
            <template #body="{ data }">
              <strong class="font-bold text-brand-navy">{{ data.name }}</strong>
            </template>
          </Column>

          <Column field="parent_id" header="Parent Category">
            <template #body="{ data }">
              <span class="text-xs font-medium text-slate-600">
                {{ getParentCategoryName(data.parent_id) }}
              </span>
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

          <Column v-if="canManageCatalog" header="Actions" align-frozen="right" freeze>
            <template #body="{ data }">
              <div class="flex items-center gap-2">
                <Button
                  icon="pi pi-pencil"
                  severity="secondary"
                  text
                  rounded
                  size="small"
                  title="Edit category"
                  @click="openEditCategory(data)"
                />
                <Button
                  v-if="data.is_active"
                  icon="pi pi-trash"
                  severity="danger"
                  text
                  rounded
                  size="small"
                  title="Deactivate category"
                  @click="deactivateCategory(data)"
                />
              </div>
            </template>
          </Column>
        </DataTable>
      </template>
    </Card>

    <CategoryFormDialog
      v-model:visible="categoryFormVisible"
      :category="selectedCategory"
      :categories="catalogStore.categories"
      @saved="catalogStore.fetchCategories(searchQuery)"
    />
  </div>
</template>
