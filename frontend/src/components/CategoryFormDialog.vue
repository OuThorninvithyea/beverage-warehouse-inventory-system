<script setup lang="ts">
import { Check } from 'lucide-vue-next'
import { computed, ref, watch } from 'vue'

import type { Category, CategoryInput } from '@/api/catalog'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
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
const parentId = ref<string>('none')
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
      name.value = props.category?.name ?? ''
      parentId.value = props.category?.parent_id ?? 'none'
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
    parent_id: parentId.value === 'none' ? null : parentId.value,
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
  } catch (err: unknown) {
    errorMessage.value = err instanceof Error ? err.message : 'Failed to save category'
  }
}

function closeDialog() {
  emit('update:visible', false)
}
</script>

<template>
  <Dialog :open="visible" @update:open="(value: boolean) => emit('update:visible', value)">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>{{ isEditMode ? 'Edit Category' : 'Add New Category' }}</DialogTitle>
        <DialogDescription>Group catalog products by beverage type or brand.</DialogDescription>
      </DialogHeader>

      <form class="grid gap-4" @submit.prevent="saveCategory">
        <p
          v-if="errorMessage"
          class="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {{ errorMessage }}
        </p>

        <div class="grid gap-2">
          <Label for="cat-name">Category Name *</Label>
          <Input
            id="cat-name"
            v-model="name"
            placeholder="e.g. Soft Drinks, Energy Drinks, Water"
            required
          />
        </div>

        <div class="grid gap-2">
          <Label for="parent-cat">Parent Category</Label>
          <Select v-model="parentId">
            <SelectTrigger id="parent-cat" class="w-full">
              <SelectValue placeholder="None (Top Level)" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="none">None (Top Level)</SelectItem>
              <SelectItem v-for="cat in parentCategoryOptions" :key="cat.id" :value="cat.id">
                {{ cat.name }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
      </form>

      <DialogFooter>
        <Button variant="outline" @click="closeDialog">Cancel</Button>
        <Button :disabled="catalogStore.loading" @click="saveCategory">
          <Check class="size-4" />
          {{ isEditMode ? 'Update' : 'Create' }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
