import { DOMWrapper, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiClientError } from '@/api/client'
import * as warehousesApi from '@/api/warehouses'
import WarehouseFormDialog from '@/components/WarehouseFormDialog.vue'

vi.mock('@/api/warehouses')

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

afterEach(() => {
  document.body.innerHTML = ''
})

function mountDialog(warehouse: warehousesApi.Warehouse | null = null) {
  return mount(WarehouseFormDialog, {
    props: { visible: true, warehouse },
    attachTo: document.body,
  })
}

function body() {
  return new DOMWrapper(document.body)
}

describe('WarehouseFormDialog', () => {
  it('blocks submit and shows errors when code and name are blank', async () => {
    const wrapper = mountDialog()
    await flushPromises()
    await body().find('[data-testid="submit"]').trigger('click')
    expect(body().text()).toContain('Code is required.')
    expect(body().text()).toContain('Name is required.')
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
    await flushPromises()
    await body().find('#warehouse-code').setValue('PP-01')
    await body().find('#warehouse-name').setValue('Phnom Penh Main')
    await body().find('[data-testid="submit"]').trigger('click')
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
    await flushPromises()
    await body().find('#warehouse-code').setValue('PP-01')
    await body().find('#warehouse-name').setValue('Phnom Penh Main')
    await body().find('[data-testid="submit"]').trigger('click')
    await flushPromises()
    expect(body().text()).toContain('This code is already in use.')
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
    await flushPromises()
    expect((body().find('#warehouse-code').element as HTMLInputElement).value).toBe('PP-01')
    await body().find('[data-testid="submit"]').trigger('click')
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
  await new Promise((resolve) => setTimeout(resolve, 20))
}
