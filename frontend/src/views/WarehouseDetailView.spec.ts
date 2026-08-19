import Aura from '@primeuix/themes/aura'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import PrimeVue from 'primevue/config'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import * as warehousesApi from '@/api/warehouses'
import { useAuthStore } from '@/stores/auth'
import WarehouseDetailView from '@/views/WarehouseDetailView.vue'

vi.mock('@/api/warehouses')
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { warehouseId: 'wh-1' } }),
}))

const sampleWarehouse: warehousesApi.Warehouse = {
  id: 'wh-1',
  code: 'PP-01',
  name: 'Phnom Penh Main',
  address: 'Sen Sok',
  is_active: true,
  created_at: '',
  updated_at: '',
}

const sampleLocation: warehousesApi.Location = {
  id: 'loc-1',
  warehouse_id: 'wh-1',
  code: 'A-01',
  zone: 'Ambient',
  aisle: 'A-01',
  rack: null,
  shelf: null,
  barcode: null,
  is_pickable: true,
  is_active: true,
  created_at: '',
  updated_at: '',
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
  vi.mocked(warehousesApi.getWarehouse).mockResolvedValue(sampleWarehouse)
  vi.mocked(warehousesApi.listLocations).mockResolvedValue({
    items: [sampleLocation],
    page: { next_cursor: null, has_more: false },
  })
})

function mountView() {
  return mount(WarehouseDetailView, {
    global: {
      plugins: [[PrimeVue, { theme: { preset: Aura } }]],
      stubs: { Portal: { template: '<div><slot /></div>' } },
    },
  })
}

async function flushPromises() {
  await new Promise((resolve) => setTimeout(resolve, 0))
}

describe('WarehouseDetailView', () => {
  it('shows the warehouse and its locations', async () => {
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('Phnom Penh Main')
    expect(wrapper.text()).toContain('A-01')
  })

  it('hides Add Location for a viewer', async () => {
    const auth = useAuthStore()
    auth.user = {
      id: 'u1', email: 'v@bwims.test', full_name: 'Viewer', role: 'viewer', warehouse_id: 'wh-1',
    }
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.find('[data-testid="add-location"]').exists()).toBe(false)
  })

  it('shows Add Location for a warehouse manager', async () => {
    const auth = useAuthStore()
    auth.user = {
      id: 'u1', email: 'm@bwims.test', full_name: 'Manager', role: 'warehouse_manager', warehouse_id: 'wh-1',
    }
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.find('[data-testid="add-location"]').exists()).toBe(true)
  })

  it('shows an error banner when the warehouse fails to load', async () => {
    vi.mocked(warehousesApi.getWarehouse).mockRejectedValue(new Error('not found'))
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.find('[data-testid="warehouse-error"]').text()).toBe('not found')
  })
})
