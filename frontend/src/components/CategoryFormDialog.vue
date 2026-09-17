<script setup lang="ts">
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'
import Select from 'primevue/select'
import { computed, ref, watch } from 'vue'

import type { Category, CategoryInput } from '@/api/catalog'
import { useCatalogStore } from '@/stores/catalog'

const props = defineProps<{
  visible: boolean
  category?: Category | null
  categories: Category[]
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'saved', category: Category): void
}>()

const catalogStore = useCatalogStore()

const name = ref('')
const parentId = ref<string | null>(null)
const errorMessage = ref('')

const isEditMode = computed(() => Boolean(props.category?.id))

const parentCategoryOptions = computed(() => {
  if (!props.category?.id) return props.categories
  return props.categories.filter((c) => c.id !== props.category?.id)
})

watch(
  () => props.visible,
  (isVis) => {
    if (isVis) {
      errorMessage.value = ''
      if (props.category) {
        name.value = props.category.name
        parentId.value = props.category.parent_id
      } else {
        name.value = ''
        parentId.value = null
      }
    }
  },
)

async function saveCategory() {
  errorMessage.value = ''
  if (!name.value.trim()) {
    errorMessage.value = 'Category name is required'
    return
  }

  const payload: CategoryInput = {
    name: name.value.trim(),
    parent_id: parentId.value || null,
  }

  try {
    let saved: Category
    if (isEditMode.value && props.category?.id) {
      saved = await catalogStore.editCategory(props.category.id, payload)
    } else {
      saved = await catalogStore.addCategory(payload)
    }
    emit('saved', saved)
    closeDialog()
  } catch (err: any) {
    errorMessage.value = err.message || 'Failed to save category'
  }
}

function closeDialog() {
  emit('update:visible', false)
}
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    :header="isEditMode ? 'Edit Category' : 'Add New Category'"
    :style="{ width: '90vw', maxWidth: '480px' }"
    @update:visible="closeDialog"
  >
    <form class="grid gap-4 py-2" @submit.prevent="saveCategory">
      <Message v-if="errorMessage" severity="error">
        {{ errorMessage }}
      </Message>

      <div class="grid gap-1">
        <label for="cat-name" class="text-xs font-semibold text-brand-muted">Category Name *</label>
        <InputText
          id="cat-name"
          v-model="name"
          placeholder="e.g. Soft Drinks, Energy Drinks, Water"
          required
        />
      </div>

      <div class="grid gap-1">
        <label for="parent-cat" class="text-xs font-semibold text-brand-muted">Parent Category</label>
        <Select
          id="parent-cat"
          v-model="parentId"
          :options="parentCategoryOptions"
          option-label="name"
          option-value="id"
          placeholder="None (Top Level)"
          show-clear
        />
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button label="Cancel" severity="secondary" outlined @click="closeDialog" />
        <Button
          :label="isEditMode ? 'Update' : 'Create'"
          icon="pi pi-check"
          :loading="catalogStore.loading"
          @click="saveCategory"
        />
      </div>
    </template>
  </Dialog>
</template>
