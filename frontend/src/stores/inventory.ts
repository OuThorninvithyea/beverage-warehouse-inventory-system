import { defineStore } from 'pinia'
import { ref } from 'vue'

import {
  type AdjustInput,
  type BalanceListFilter,
  type ExpiryAlert,
  type ExpiryAlertFilter,
  type Lot,
  type MovementListFilter,
  type PickInput,
  type ReceiveInput,
  type StockBalance,
  type StockMovement,
  type TransferInput,
  adjustStock,
  listBalances,
  listExpiryAlerts,
  listLots,
  listMovements,
  pickStock,
  receiveStock,
  transferStock,
} from '@/api/inventory'

export const useInventoryStore = defineStore('inventory', () => {
  const balances = ref<StockBalance[]>([])
  const movements = ref<StockMovement[]>([])
  const currentProductLots = ref<Lot[]>([])
  const expiryAlerts = ref<ExpiryAlert[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const hasMore = ref(false)
  const nextCursor = ref<string | null>(null)

  async function fetchBalances(filter: BalanceListFilter = {}): Promise<string | null> {
    loading.value = true
    error.value = null
    try {
      const page = await listBalances(filter)
      balances.value = page.items
      nextCursor.value = page.page.next_cursor
      hasMore.value = page.page.has_more
      return page.page.has_more ? page.page.next_cursor : null
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch inventory balances'
      return null
    } finally {
      loading.value = false
    }
  }

  async function fetchMovements(filter: MovementListFilter = {}): Promise<string | null> {
    loading.value = true
    error.value = null
    try {
      const page = await listMovements(filter)
      movements.value = page.items
      nextCursor.value = page.page.next_cursor
      hasMore.value = page.page.has_more
      return page.page.has_more ? page.page.next_cursor : null
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch stock movements'
      return null
    } finally {
      loading.value = false
    }
  }

  async function fetchExpiryAlerts(filter: ExpiryAlertFilter = {}) {
    loading.value = true
    error.value = null
    try {
      expiryAlerts.value = await listExpiryAlerts(filter)
      return expiryAlerts.value
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch expiry alerts'
      return []
    } finally {
      loading.value = false
    }
  }

  async function fetchLots(productId: string) {
    loading.value = true
    error.value = null
    try {
      const res = await listLots(productId)
      currentProductLots.value = res
      return res
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch product lots'
      return []
    } finally {
      loading.value = false
    }
  }

  async function doReceive(input: ReceiveInput) {
    loading.value = true
    error.value = null
    try {
      const res = await receiveStock(input)
      movements.value.unshift(res.movement)
      return res
    } catch (e: any) {
      error.value = e.message || 'Stock receive failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function doPick(input: PickInput) {
    loading.value = true
    error.value = null
    try {
      const res = await pickStock(input)
      if (res.movements) {
        movements.value.unshift(...res.movements)
      }
      return res
    } catch (e: any) {
      error.value = e.message || 'FEFO stock pick failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function doTransfer(input: TransferInput) {
    loading.value = true
    error.value = null
    try {
      const res = await transferStock(input)
      movements.value.unshift(res.movement)
      return res
    } catch (e: any) {
      error.value = e.message || 'Stock transfer failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function doAdjust(input: AdjustInput) {
    loading.value = true
    error.value = null
    try {
      const res = await adjustStock(input)
      movements.value.unshift(res.movement)
      return res
    } catch (e: any) {
      error.value = e.message || 'Stock adjustment failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    balances,
    movements,
    currentProductLots,
    expiryAlerts,
    loading,
    error,
    hasMore,
    fetchBalances,
    fetchMovements,
    fetchLots,
    fetchExpiryAlerts,
    doReceive,
    doPick,
    doTransfer,
    doAdjust,
  }
})
