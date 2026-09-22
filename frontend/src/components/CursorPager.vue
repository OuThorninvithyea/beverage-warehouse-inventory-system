<script setup lang="ts">
/**
 * Previous/Next pager for the cursor-based list endpoints.
 *
 * Deliberately not numbered pages: the API returns an opaque cursor and
 * `has_more` with no total count, so there is no offset to jump to and no
 * total to divide into pages. Showing page numbers would mean inventing them.
 */
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'

import { Button } from '@/components/ui/button'

const props = defineProps<{
  pageNumber: number
  canGoBack: boolean
  canGoForward: boolean
  busy?: boolean
  itemCount: number
}>()

const emit = defineEmits<{
  (e: 'previous'): void
  (e: 'next'): void
}>()
</script>

<template>
  <div
    v-if="props.canGoBack || props.canGoForward"
    class="flex items-center justify-between gap-4 border-t px-1 pt-3"
  >
    <p class="text-xs text-muted-foreground">
      Page {{ props.pageNumber }} · {{ props.itemCount }}
      {{ props.itemCount === 1 ? 'row' : 'rows' }}
    </p>
    <div class="flex items-center gap-2">
      <Button
        variant="outline"
        size="sm"
        :disabled="!props.canGoBack || props.busy"
        @click="emit('previous')"
      >
        <ChevronLeft class="size-4" /> Previous
      </Button>
      <Button
        variant="outline"
        size="sm"
        :disabled="!props.canGoForward || props.busy"
        @click="emit('next')"
      >
        Next <ChevronRight class="size-4" />
      </Button>
    </div>
  </div>
</template>
