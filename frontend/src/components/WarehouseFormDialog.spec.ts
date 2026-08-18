import Aura from '@primeuix/themes/aura'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import PrimeVue from 'primevue/config'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiClientError } from '@/api/client'
import * as warehousesApi from '@/api/warehouses'
import WarehouseFormDialog from '@/components/WarehouseFormDialog.vue'

vi.mock('@/api/warehouses')

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

function mountDialog(warehouse: warehousesApi.Warehouse | null = null) {
  return mount(WarehouseFormDialog, {
    props: { visible: true, warehouse },
    global: {
      plugins: [[PrimeVue, { theme: { preset: Aura } }]],
      stubs: { Portal: { template: '<div><slot /></div>' } },
    },
  })
}

describe('WarehouseFormDialog', () => {
  it('blocks submit and shows errors when code and name are blank', async () => {
    const wrapper = mountDialog()
    await wrapper.find('[data-testid="submit"]').trigger('click')
    expect(wrapper.text()).toContain('Code is required.')
    expect(wrapper.text()).toContain('Name is required.')
    expect(warehousesApi.createWarehouse).not.toHaveBeenCalled()
  })

  it('creates a warehouse and emits saved on valid submit', async () => {
    vi.mocked(warehousesApi.createWarehouse).mockResolvedValue({
      id: 'wh-1',
      code: 'PP-01',
      name: 'Phnom Penh Main',
      address: null,
      is_active: true,
      created_at: '',
      updated_at: '',
    })
    const wrapper = mountDialog()
    await wrapper.find('#warehouse-code').setValue('PP-01')
    await wrapper.find('#warehouse-name').setValue('Phnom Penh Main')
    await wrapper.find('[data-testid="submit"]').trigger('click')
    await flushPromises()
    expect(warehousesApi.createWarehouse).toHaveBeenCalledWith({
      code: 'PP-01',
      name: 'Phnom Penh Main',
      address: null,
      is_active: true,
    })
    expect(wrapper.emitted('saved')).toBeTruthy()
    expect(wrapper.emitted('update:visible')).toEqual([[false]])
  })

  it('shows a code-conflict error on the code field instead of a generic message', async () => {
    vi.mocked(warehousesApi.createWarehouse).mockRejectedValue(
      new ApiClientError(409, 'WAREHOUSE_CODE_CONFLICT', 'Warehouse code already exists'),
    )
    const wrapper = mountDialog()
    await wrapper.find('#warehouse-code').setValue('PP-01')
    await wrapper.find('#warehouse-name').setValue('Phnom Penh Main')
    await wrapper.find('[data-testid="submit"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('This code is already in use.')
  })

  it('pre-fills the form and calls updateWarehouse in edit mode', async () => {
    const existing: warehousesApi.Warehouse = {
      id: 'wh-1',
      code: 'PP-01',
      name: 'Phnom Penh Main',
      address: 'Sen Sok',
      is_active: true,
      created_at: '',
      updated_at: '',
    }
    vi.mocked(warehousesApi.updateWarehouse).mockResolvedValue(existing)
    const wrapper = mountDialog(existing)
    expect((wrapper.find('#warehouse-code').element as HTMLInputElement).value).toBe('PP-01')
    await wrapper.find('[data-testid="submit"]').trigger('click')
    await flushPromises()
    expect(warehousesApi.updateWarehouse).toHaveBeenCalledWith('wh-1', {
      code: 'PP-01',
      name: 'Phnom Penh Main',
      address: 'Sen Sok',
      is_active: true,
    })
  })
})

async function flushPromises() {
  await new Promise((resolve) => setTimeout(resolve, 0))
}
