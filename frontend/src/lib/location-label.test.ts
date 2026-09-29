import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'

import type { Location, Warehouse } from '@/api/warehouses'
import { useWarehousesStore } from '@/stores/warehouses'

import { useLocationLabels } from './location-label'

const warehouse = (id: string, code: string, name: string) => ({ id, code, name }) as Warehouse
const location = (id: string, warehouse_id: string, code: string) =>
  ({ id, warehouse_id, code, is_active: true }) as Location

beforeEach(() => {
  setActivePinia(createPinia())
  useWarehousesStore().warehouses = [
    warehouse('pp', 'PP-CENTRAL', 'Phnom Penh Central DC'),
    warehouse('sr', 'SR-DEPOT', 'Siem Reap Depot'),
  ]
})

describe('useLocationLabels', () => {
  it('names the warehouse, since every branch has the same shelf codes', () => {
    const { label } = useLocationLabels()
    expect(label(location('a', 'sr', 'A-01-01'))).toBe('A-01-01 (Siem Reap Depot)')
    expect(label(location('b', 'pp', 'A-01-01'))).toBe('A-01-01 (Phnom Penh Central DC)')
  })

  it('falls back to the bare code for an unknown warehouse', () => {
    expect(useLocationLabels().label(location('c', 'gone', 'X-01'))).toBe('X-01')
  })

  it('groups locations under their warehouse in name order', () => {
    const groups = useLocationLabels().groupByWarehouse([
      location('1', 'sr', 'COLD-01'),
      location('2', 'pp', 'B-01-01'),
      location('3', 'sr', 'A-01-01'),
    ])
    expect(groups.map((group) => group.warehouse?.code)).toEqual(['PP-CENTRAL', 'SR-DEPOT'])
    expect(groups[1].locations.map((row) => row.code)).toEqual(['A-01-01', 'COLD-01'])
  })
})
