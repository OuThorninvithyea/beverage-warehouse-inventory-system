<script setup lang="ts">
import { CheckCircle2, ShieldCheck } from 'lucide-vue-next'

export interface SummaryRow {
  label: string
  value: string
  mono?: boolean
  strong?: boolean
}

withDefaults(
  defineProps<{
    rows: SummaryRow[]
    mode?: 'review' | 'success'
    message?: string
  }>(),
  {
    mode: 'review',
    message: '',
  },
)
</script>

<template>
  <div class="grid gap-4">
    <div
      v-if="mode === 'success'"
      class="flex items-start gap-3 rounded-xl border border-emerald-500/30 bg-emerald-500/10 p-4"
      role="status"
    >
      <span class="grid size-10 shrink-0 place-items-center rounded-full bg-emerald-500/15 text-emerald-700 dark:text-emerald-400">
        <CheckCircle2 class="size-5" />
      </span>
      <div class="grid gap-1">
        <strong class="text-sm text-emerald-800 dark:text-emerald-300">Operation completed successfully</strong>
        <p class="text-sm text-emerald-800/80 dark:text-emerald-300/80">
          {{ message || 'The inventory record and audit trail have been updated.' }}
        </p>
      </div>
    </div>

    <div v-else class="flex items-start gap-3 rounded-xl border border-amber-500/30 bg-amber-500/10 p-4">
      <ShieldCheck class="mt-0.5 size-5 shrink-0 text-amber-700 dark:text-amber-400" />
      <p class="text-sm text-amber-900 dark:text-amber-200">
        Check these details carefully. Confirming will change warehouse stock and create a permanent audit record.
      </p>
    </div>

    <dl class="divide-y rounded-xl border bg-card px-4">
      <div v-for="row in rows" :key="row.label" class="grid grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)] gap-4 py-3 text-sm max-[520px]:grid-cols-1 max-[520px]:gap-1">
        <dt class="text-muted-foreground">{{ row.label }}</dt>
        <dd
          class="break-words text-right max-[520px]:text-left"
          :class="[
            row.mono ? 'font-mono text-xs' : '',
            row.strong ? 'font-semibold text-foreground' : 'text-foreground',
          ]"
        >
          {{ row.value }}
        </dd>
      </div>
    </dl>
  </div>
</template>
