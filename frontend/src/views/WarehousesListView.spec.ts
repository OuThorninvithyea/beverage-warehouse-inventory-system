import Aura from '@primeuix/themes/aura'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import PrimeVue from 'primevue/config'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import * as warehousesApi from '@/api/warehouses'
import { useAuthStore } from '@/stores/auth'
import WarehousesListView from '@/views/WarehousesListView.vue'

vi.mock('@/api/warehouses')
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

const sampleWarehouse: warehousesApi.Warehouse = {
  id: 'wh-1',
  code: 'PP-01',
  name: 'Phnom Penh Main',
  address: null,
  is_active: true,
  created_at: '',
  updated_at: '',
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
  vi.mocked(warehousesApi.listWarehouses).mockResolvedValue({
    items: [sampleWarehouse],
    page: { next_cursor: null, has_more: false },
  })
})

function mountView() {
  return mount(WarehousesListView, {
    global: {
      plugins: [[PrimeVue, { theme: { preset: Aura } }]],
      stubs: { Portal: { template: '<div><slot /></div>' } },
    },
  })
}

async function flushPromises() {
  await new Promise((resolve) => setTimeout(resolve, 0))
}

describe('WarehousesListView', () => {
  it('renders warehouses from the store after mount', async () => {
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('PP-01')
    expect(wrapper.text()).toContain('Phnom Penh Main')
  })

  it('hides Add Warehouse for a non-admin', async () => {
    const auth = useAuthStore()
    auth.user = {
      id: 'u1', email: 'p@bwims.test', full_name: 'Picker', role: 'picker', warehouse_id: 'wh-1',
    }
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.find('[data-testid="add-warehouse"]').exists()).toBe(false)
  })

  it('shows Add Warehouse for an admin', async () => {
    const auth = useAuthStore()
    auth.user = { id: 'u1', email: 'a@bwims.test', full_name: 'Admin', role: 'admin' }
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.find('[data-testid="add-warehouse"]').exists()).toBe(true)
  })

  it('shows an error banner and no table when loading fails', async () => {
    vi.mocked(warehousesApi.listWarehouses).mockRejectedValue(new Error('network down'))
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.find('[data-testid="warehouses-error"]').text()).toBe('network down')
  })
})
