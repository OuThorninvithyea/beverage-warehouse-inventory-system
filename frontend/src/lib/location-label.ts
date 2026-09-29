import { computed } from 'vue'

import type { Location, Warehouse } from '@/api/warehouses'
import { useWarehousesStore } from '@/stores/warehouses'

/**
 * Location codes repeat across warehouses (every branch has an A-01-01), so a
 * bare code does not say where stock is going. Everything that names a
 * location to the user goes through here: "A-01-01 (Siem Reap Depot)".
 */

export interface LocationGroup {
  warehouse: Warehouse | null
  locations: Location[]
}

export function locationLabel(location: Location, warehouse: Warehouse | null | undefined): string {
  return warehouse ? `${location.code} (${warehouse.name})` : location.code
}

export function useLocationLabels() {
  const warehouseStore = useWarehousesStore()

  const warehousesById = computed(
    () => new Map(warehouseStore.warehouses.map((warehouse) => [warehouse.id, warehouse])),
  )

  function warehouseOf(location: Location | undefined | null): Warehouse | null {
    return location ? (warehousesById.value.get(location.warehouse_id) ?? null) : null
  }

  function label(location: Location | undefined | null): string {
    return location ? locationLabel(location, warehouseOf(location)) : ''
  }

  /** Locations grouped under their warehouse, warehouses and codes in order. */
  function groupByWarehouse(locations: Location[]): LocationGroup[] {
    const groups = new Map<string, LocationGroup>()
    for (const location of locations) {
      const group = groups.get(location.warehouse_id) ?? {
        warehouse: warehouseOf(location),
        locations: [],
      }
      group.locations.push(location)
      groups.set(location.warehouse_id, group)
    }
    return [...groups.values()]
      .map((group) => ({
        ...group,
        locations: [...group.locations].sort((a, b) => a.code.localeCompare(b.code)),
      }))
      .sort((a, b) => (a.warehouse?.name ?? '').localeCompare(b.warehouse?.name ?? ''))
  }

  return { warehouseOf, label, groupByWarehouse }
}
