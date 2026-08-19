import { defineStore } from 'pinia'
import { ref } from 'vue'

import * as warehousesApi from '@/api/warehouses'
import type { Location, LocationInput, Warehouse, WarehouseInput } from '@/api/warehouses'

function messageOf(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback
}

export const useWarehousesStore = defineStore('warehouses', () => {
  const warehouses = ref<Warehouse[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const hasMore = ref(false)
  const nextCursor = ref<string | null>(null)

  async function fetchWarehouses(search = '') {
    loading.value = true
    error.value = null
    try {
      const page = await warehousesApi.listWarehouses({ search: search || undefined })
      warehouses.value = page.items
      hasMore.value = page.page.has_more
      nextCursor.value = page.page.next_cursor
    } catch (err) {
      error.value = messageOf(err, 'Failed to load warehouses.')
    } finally {
      loading.value = false
    }
  }

  async function loadMoreWarehouses(search = '') {
    if (!hasMore.value || !nextCursor.value) {
      return
    }
    loading.value = true
    error.value = null
    try {
      const page = await warehousesApi.listWarehouses({
        search: search || undefined,
        after: nextCursor.value,
      })
      warehouses.value = [...warehouses.value, ...page.items]
      hasMore.value = page.page.has_more
      nextCursor.value = page.page.next_cursor
    } catch (err) {
      error.value = messageOf(err, 'Failed to load warehouses.')
    } finally {
      loading.value = false
    }
  }

  async function createWarehouse(input: WarehouseInput) {
    const warehouse = await warehousesApi.createWarehouse(input)
    warehouses.value = [warehouse, ...warehouses.value]
    return warehouse
  }

  async function updateWarehouse(id: string, input: WarehouseInput) {
    const warehouse = await warehousesApi.updateWarehouse(id, input)
    warehouses.value = warehouses.value.map((item) => (item.id === id ? warehouse : item))
    return warehouse
  }

  async function deactivateWarehouse(id: string) {
    await warehousesApi.deactivateWarehouse(id)
    warehouses.value = warehouses.value.map((item) =>
      item.id === id ? { ...item, is_active: false } : item,
    )
  }

  const locations = ref<Location[]>([])
  const locationsLoading = ref(false)
  const locationsError = ref<string | null>(null)
  const locationsHasMore = ref(false)
  const locationsNextCursor = ref<string | null>(null)

  async function fetchLocations(warehouseId: string) {
    locationsLoading.value = true
    locationsError.value = null
    try {
      const page = await warehousesApi.listLocations(warehouseId)
      locations.value = page.items
      locationsHasMore.value = page.page.has_more
      locationsNextCursor.value = page.page.next_cursor
    } catch (err) {
      locationsError.value = messageOf(err, 'Failed to load locations.')
    } finally {
      locationsLoading.value = false
    }
  }

  async function loadMoreLocations(warehouseId: string) {
    if (!locationsHasMore.value || !locationsNextCursor.value) {
      return
    }
    locationsLoading.value = true
    locationsError.value = null
    try {
      const page = await warehousesApi.listLocations(warehouseId, {
        after: locationsNextCursor.value,
      })
      locations.value = [...locations.value, ...page.items]
      locationsHasMore.value = page.page.has_more
      locationsNextCursor.value = page.page.next_cursor
    } catch (err) {
      locationsError.value = messageOf(err, 'Failed to load locations.')
    } finally {
      locationsLoading.value = false
    }
  }

  async function createLocation(warehouseId: string, input: LocationInput) {
    const location = await warehousesApi.createLocation(warehouseId, input)
    locations.value = [location, ...locations.value]
    return location
  }

  async function updateLocation(warehouseId: string, locationId: string, input: LocationInput) {
    const location = await warehousesApi.updateLocation(warehouseId, locationId, input)
    locations.value = locations.value.map((item) => (item.id === locationId ? location : item))
    return location
  }

  async function deactivateLocation(warehouseId: string, locationId: string) {
    await warehousesApi.deactivateLocation(warehouseId, locationId)
    locations.value = locations.value.map((item) =>
      item.id === locationId ? { ...item, is_active: false } : item,
    )
  }

  return {
    warehouses,
    loading,
    error,
    hasMore,
    fetchWarehouses,
    loadMoreWarehouses,
    createWarehouse,
    updateWarehouse,
    deactivateWarehouse,
    locations,
    locationsLoading,
    locationsError,
    locationsHasMore,
    fetchLocations,
    loadMoreLocations,
    createLocation,
    updateLocation,
    deactivateLocation,
  }
})
