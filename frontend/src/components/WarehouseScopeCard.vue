<script setup lang="ts">
/**
 * Tells the user which warehouse the dashboard is showing and what locations
 * it has, so nobody has to guess where "here" is before receiving or picking.
 *
 * One warehouse: its name, address and every location as a chip.
 * All warehouses (admin only): one tile per warehouse; clicking a tile narrows
 * the dashboard to it.
 */
import { Building2, MapPin } from 'lucide-vue-next'
import { computed } from 'vue'

import type { Location, Warehouse } from '@/api/warehouses'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'

const props = defineProps<{
  /** The warehouse in view, or null for "all warehouses". */
  warehouse: Warehouse | null
  warehouses: Warehouse[]
  locations: Location[]
  /** Admins can switch warehouse from here; everyone else is fixed to theirs. */
  canSwitch: boolean
}>()

const emit = defineEmits<{ select: [warehouseId: string] }>()

const warehouseLocations = computed(() =>
  props.warehouse
    ? props.locations
        .filter((location) => location.warehouse_id === props.warehouse?.id)
        .sort((a, b) => Number(b.is_active) - Number(a.is_active) || a.code.localeCompare(b.code))
    : [],
)

const activeCount = (warehouseId: string) =>
  props.locations.filter((location) => location.warehouse_id === warehouseId && location.is_active)
    .length

const sortedWarehouses = computed(() =>
  [...props.warehouses].sort(
    (a, b) => Number(b.is_active) - Number(a.is_active) || a.code.localeCompare(b.code),
  ),
)
</script>

<template>
  <Card>
    <CardContent class="grid gap-4">
      <!-- One warehouse in view -->
      <template v-if="warehouse">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="flex items-start gap-3">
            <span class="grid size-10 shrink-0 place-items-center rounded-lg bg-primary/10 text-primary">
              <MapPin class="size-5" />
            </span>
            <div class="grid gap-0.5">
              <span class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                You are viewing
              </span>
              <strong class="text-lg font-semibold leading-tight">
                {{ warehouse.name }}
                <span class="font-mono text-sm font-normal text-muted-foreground">
                  {{ warehouse.code }}
                </span>
              </strong>
              <span v-if="warehouse.address" class="text-sm text-muted-foreground">
                {{ warehouse.address }}
              </span>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <Badge v-if="!warehouse.is_active" variant="destructive">Inactive</Badge>
            <button
              v-if="canSwitch"
              type="button"
              class="text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
              @click="emit('select', 'all')"
            >
              Show all warehouses
            </button>
          </div>
        </div>

        <div class="grid gap-2">
          <span class="text-xs font-medium text-muted-foreground">
            {{ activeCount(warehouse.id) }} active
            {{ activeCount(warehouse.id) === 1 ? 'location' : 'locations' }}
          </span>
          <p v-if="warehouseLocations.length === 0" class="text-sm text-muted-foreground">
            No locations set up in this warehouse yet.
          </p>
          <div v-else class="flex flex-wrap gap-2">
            <span
              v-for="location in warehouseLocations"
              :key="location.id"
              class="inline-flex items-center gap-1.5 rounded-md border px-2 py-1 text-xs"
              :class="{ 'opacity-50 line-through': !location.is_active }"
              :title="location.is_active ? undefined : 'Inactive location'"
            >
              <span class="font-mono font-medium">{{ location.code }}</span>
              <span v-if="location.zone" class="text-muted-foreground">{{ location.zone }}</span>
            </span>
          </div>
        </div>
      </template>

      <!-- Admin looking across every warehouse -->
      <template v-else>
        <div class="flex items-center gap-3">
          <span class="grid size-10 shrink-0 place-items-center rounded-lg bg-primary/10 text-primary">
            <Building2 class="size-5" />
          </span>
          <div class="grid gap-0.5">
            <span class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              You are viewing
            </span>
            <strong class="text-lg font-semibold leading-tight">All warehouses</strong>
          </div>
        </div>
        <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
          <button
            v-for="item in sortedWarehouses"
            :key="item.id"
            type="button"
            class="grid gap-1 rounded-lg border p-3 text-left transition-colors hover:bg-muted"
            :class="{ 'opacity-60': !item.is_active }"
            @click="emit('select', item.id)"
          >
            <span class="flex items-center justify-between gap-2">
              <strong class="truncate text-sm">{{ item.name }}</strong>
              <Badge v-if="!item.is_active" variant="secondary">Inactive</Badge>
            </span>
            <span class="font-mono text-xs text-muted-foreground">{{ item.code }}</span>
            <span class="text-xs text-muted-foreground">
              {{ activeCount(item.id) }} active
              {{ activeCount(item.id) === 1 ? 'location' : 'locations' }}
            </span>
          </button>
        </div>
      </template>
    </CardContent>
  </Card>
</template>
