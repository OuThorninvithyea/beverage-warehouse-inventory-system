import Aura from '@primeuix/themes/aura'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import PrimeVue from 'primevue/config'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiClientError } from '@/api/client'
import * as warehousesApi from '@/api/warehouses'
import LocationFormDialog from '@/components/LocationFormDialog.vue'

vi.mock('@/api/warehouses')

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

function mountDialog(location: warehousesApi.Location | null = null) {
  return mount(LocationFormDialog, {
    props: { visible: true, warehouseId: 'wh-1', location },
    global: {
      plugins: [[PrimeVue, { theme: { preset: Aura } }]],
      stubs: { Portal: { template: '<div><slot /></div>' } },
    },
  })
}

describe('LocationFormDialog', () => {
  it('blocks submit and shows an error when code is blank', async () => {
    const wrapper = mountDialog()
    await wrapper.find('[data-testid="submit"]').trigger('click')
    expect(wrapper.text()).toContain('Code is required.')
    expect(warehousesApi.createLocation).not.toHaveBeenCalled()
  })

  it('creates a location scoped to the warehouse and emits saved', async () => {
    vi.mocked(warehousesApi.createLocation).mockResolvedValue({
      id: 'loc-1',
      warehouse_id: 'wh-1',
      code: 'A-01',
      zone: null,
      aisle: null,
      rack: null,
      shelf: null,
      barcode: null,
      is_pickable: true,
      is_active: true,
      created_at: '',
      updated_at: '',
    })
    const wrapper = mountDialog()
    await wrapper.find('#location-code').setValue('A-01')
    await wrapper.find('[data-testid="submit"]').trigger('click')
    await flushPromises()
    expect(warehousesApi.createLocation).toHaveBeenCalledWith('wh-1', {
      code: 'A-01',
      zone: null,
      aisle: null,
      rack: null,
      shelf: null,
      barcode: null,
      is_pickable: true,
      is_active: true,
    })
    expect(wrapper.emitted('saved')).toBeTruthy()
  })

  it('shows a barcode-conflict error on the barcode field', async () => {
    vi.mocked(warehousesApi.createLocation).mockRejectedValue(
      new ApiClientError(409, 'LOCATION_BARCODE_CONFLICT', 'Barcode already exists'),
    )
    const wrapper = mountDialog()
    await wrapper.find('#location-code').setValue('A-01')
    await wrapper.find('#location-barcode').setValue('4006381333931')
    await wrapper.find('[data-testid="submit"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('This barcode is already in use.')
  })

  it('shows a valid-checksum hint for a well-formed EAN-13 barcode', async () => {
    const wrapper = mountDialog()
    await wrapper.find('#location-barcode').setValue('4006381333931')
    expect(wrapper.find('[data-testid="barcode-hint"]').text()).toContain('Valid EAN-13')
  })
})

async function flushPromises() {
  await new Promise((resolve) => setTimeout(resolve, 0))
}
