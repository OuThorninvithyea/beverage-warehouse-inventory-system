<script setup lang="ts">
/**
 * A location picker grouped by warehouse. The chosen value reads
 * "A-01-01 (Siem Reap Depot)", so the operator always sees which branch a
 * shelf belongs to, not just a code that exists in every warehouse.
 */
import { computed } from 'vue'

import type { Location } from '@/api/warehouses'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useLocationLabels } from '@/lib/location-label'

const props = defineProps<{
  locations: Location[]
  id?: string
  placeholder?: string
  disabled?: boolean
}>()

const model = defineModel<string>({ required: true })

const { label, groupByWarehouse } = useLocationLabels()
const groups = computed(() => groupByWarehouse(props.locations))
</script>

<template>
  <Select v-model="model" :disabled="disabled">
    <SelectTrigger :id="id" class="w-full">
      <SelectValue :placeholder="placeholder ?? 'Select location'" />
    </SelectTrigger>
    <SelectContent>
      <SelectGroup v-for="group in groups" :key="group.warehouse?.id ?? group.locations[0].warehouse_id">
        <SelectLabel v-if="group.warehouse">
          {{ group.warehouse.name }} · {{ group.warehouse.code }}
        </SelectLabel>
        <SelectItem v-for="location in group.locations" :key="location.id" :value="location.id">
          {{ label(location) }}
        </SelectItem>
      </SelectGroup>
    </SelectContent>
  </Select>
</template>
