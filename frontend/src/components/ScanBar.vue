<script setup lang="ts">
/**
 * The scan control at the top of a movement form: one button for shelf labels
 * and product cartons alike, and a line saying what the last scan did.
 */
import { CircleAlert, CircleCheck, ScanLine } from 'lucide-vue-next'
import { ref } from 'vue'

import BarcodeScannerModal from '@/components/BarcodeScannerModal.vue'
import { Button } from '@/components/ui/button'
import type { ScanFeedback } from '@/lib/scan'

defineProps<{
  feedback: ScanFeedback | null
  /** What to scan, e.g. "Scan the shelf, then the carton". */
  hint: string
}>()

const emit = defineEmits<{ scan: [code: string] }>()

const scannerVisible = ref(false)
</script>

<template>
  <div class="grid gap-2 rounded-lg border border-dashed p-3">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <span class="text-sm text-muted-foreground">{{ hint }}</span>
      <Button type="button" variant="outline" size="sm" @click="scannerVisible = true">
        <ScanLine class="size-4" />
        Scan
      </Button>
    </div>
    <p
      v-if="feedback"
      role="status"
      class="flex items-start gap-2 text-sm"
      :class="feedback.tone === 'success' ? 'text-emerald-700 dark:text-emerald-400' : 'text-amber-700 dark:text-amber-400'"
    >
      <CircleCheck v-if="feedback.tone === 'success'" class="mt-0.5 size-4 shrink-0" />
      <CircleAlert v-else class="mt-0.5 size-4 shrink-0" />
      <span>{{ feedback.text }}</span>
      <slot name="action" />
    </p>
  </div>

  <BarcodeScannerModal v-model:visible="scannerVisible" @select="emit('scan', $event)" />
</template>
