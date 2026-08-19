# Warehouses & Locations Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Warehouses list screen and Warehouse detail (with its Locations) screen against the already-complete `/api/v1/warehouses` API, as the first real interactive frontend slice.

**Architecture:** Typed `api/warehouses.ts` functions (mirroring the existing `api/auth.ts` pattern, using the existing `apiRequest<T>` — no new HTTP client) feed a Pinia composition-style store (`stores/warehouses.ts`, mirroring `stores/auth.ts`), consumed by two routed views (`WarehousesListView.vue`, `WarehouseDetailView.vue`) and two reusable form dialogs (`WarehouseFormDialog.vue`, `LocationFormDialog.vue`).

**Tech Stack:** Vue 3 (Composition API, `<script setup>`), TypeScript, Pinia, PrimeVue 5 (`Button`, `DataTable`, `Column`, `Dialog`, `InputText`, `ToggleSwitch`, `Tag`, `Message`), Vitest + `@vue/test-utils` + jsdom (new for this plan — no component-mounting tests exist yet).

Design doc: `docs/superpowers/specs/2026-08-19-warehouses-frontend-design.md`.

---

## File Structure

```
frontend/src/
  types/pagination.ts                 # CREATE — Page<T> cursor-pagination type
  api/warehouses.ts                   # CREATE — typed fetch calls, warehouses + locations
  api/warehouses.test.ts              # CREATE
  stores/warehouses.ts                # CREATE — Pinia store
  stores/warehouses.test.ts           # CREATE
  components/WarehouseFormDialog.vue  # CREATE
  components/WarehouseFormDialog.spec.ts  # CREATE
  components/LocationFormDialog.vue   # CREATE
  components/LocationFormDialog.spec.ts   # CREATE
  views/WarehousesListView.vue        # CREATE
  views/WarehousesListView.spec.ts    # CREATE
  views/WarehouseDetailView.vue       # CREATE
  views/WarehouseDetailView.spec.ts   # CREATE
  router/index.ts                     # MODIFY — add 2 routes
  layouts/AppLayout.vue                # MODIFY — real Warehouses nav link, refresh stale note

frontend/vite.config.ts               # MODIFY — jsdom test environment
frontend/package.json                 # MODIFY — add @vue/test-utils, jsdom
```

---

## Task 1: Add component-testing infrastructure

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/vite.config.ts`

- [ ] **Step 1: Install the new dev dependencies**

```bash
cd frontend
npm install -D @vue/test-utils jsdom
```

Expected: `package.json`'s `devDependencies` gains `@vue/test-utils` and
`jsdom` entries (npm resolves the versions).

- [ ] **Step 2: Switch the Vitest config source and set the jsdom environment**

Open `frontend/vite.config.ts`. It currently starts:

```ts
import { fileURLToPath, URL } from 'node:url'

import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'
```

Change the `defineConfig` import source and add a `test` block:

```ts
import { fileURLToPath, URL } from 'node:url'

import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      '/health': 'http://localhost:8080',
      '/ready': 'http://localhost:8080',
    },
  },
  test: {
    environment: 'jsdom',
  },
})
```

(`vitest/config`'s `defineConfig` is a drop-in superset of Vite's — it
just adds the `test` key's typing. Nothing else in the file changes.)

- [ ] **Step 3: Verify the existing test suite still passes under the new config**

Run: `cd frontend && npm test`
Expected: PASS — the existing `lib/barcode.test.ts` (4 tests) still passes
under `jsdom` (it's a pure-function test, jsdom is a superset environment).

- [ ] **Step 4: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/vite.config.ts
git commit -m "chore: add component-testing infrastructure (@vue/test-utils, jsdom)"
```

---

## Task 2: `Page<T>` cursor-pagination type

**Files:**
- Create: `frontend/src/types/pagination.ts`

No test for this task — it is a pure TypeScript type declaration with no
runtime behavior to assert.

- [ ] **Step 1: Create the type**

```ts
export interface PageInfo {
  next_cursor: string | null
  has_more: boolean
}

export interface Page<T> {
  items: T[]
  page: PageInfo
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/types/pagination.ts
git commit -m "feat: add Page<T> cursor pagination type"
```

---

## Task 3: `api/warehouses.ts` — warehouse functions

**Files:**
- Create: `frontend/src/api/warehouses.ts`
- Create: `frontend/src/api/warehouses.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { apiRequest } from '@/api/client'
import {
  createWarehouse,
  deactivateWarehouse,
  getWarehouse,
  listWarehouses,
  updateWarehouse,
} from '@/api/warehouses'

vi.mock('@/api/client', () => ({
  apiRequest: vi.fn(),
}))

const mockedApiRequest = vi.mocked(apiRequest)

beforeEach(() => {
  mockedApiRequest.mockReset()
  mockedApiRequest.mockResolvedValue({} as never)
})

describe('warehouses api', () => {
  it('lists warehouses with no filters and no query string', async () => {
    await listWarehouses()
    expect(mockedApiRequest).toHaveBeenCalledWith('/warehouses')
  })

  it('lists warehouses with filters as query params', async () => {
    await listWarehouses({ search: 'PP', is_active: true, limit: 10 })
    expect(mockedApiRequest).toHaveBeenCalledWith(
      '/warehouses?search=PP&is_active=true&limit=10',
    )
  })

  it('gets a single warehouse by id', async () => {
    await getWarehouse('wh-1')
    expect(mockedApiRequest).toHaveBeenCalledWith('/warehouses/wh-1')
  })

  it('creates a warehouse with a POST body', async () => {
    await createWarehouse({ code: 'PP-01', name: 'Phnom Penh Main' })
    expect(mockedApiRequest).toHaveBeenCalledWith('/warehouses', {
      method: 'POST',
      body: JSON.stringify({ code: 'PP-01', name: 'Phnom Penh Main' }),
    })
  })

  it('updates a warehouse with a PUT body', async () => {
    await updateWarehouse('wh-1', { code: 'PP-01', name: 'Renamed' })
    expect(mockedApiRequest).toHaveBeenCalledWith('/warehouses/wh-1', {
      method: 'PUT',
      body: JSON.stringify({ code: 'PP-01', name: 'Renamed' }),
    })
  })

  it('deactivates a warehouse with DELETE', async () => {
    await deactivateWarehouse('wh-1')
    expect(mockedApiRequest).toHaveBeenCalledWith('/warehouses/wh-1', {
      method: 'DELETE',
    })
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/api/warehouses.test.ts`
Expected: FAIL — `Failed to resolve import "@/api/warehouses"`.

- [ ] **Step 3: Write the implementation**

```ts
import { apiRequest } from '@/api/client'
import type { Page } from '@/types/pagination'

export interface Warehouse {
  id: string
  code: string
  name: string
  address: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface WarehouseInput {
  code: string
  name: string
  address?: string | null
  is_active?: boolean
}

export interface WarehouseListFilter {
  limit?: number
  after?: string
  search?: string
  is_active?: boolean
}

export interface Location {
  id: string
  warehouse_id: string
  code: string
  zone: string | null
  aisle: string | null
  rack: string | null
  shelf: string | null
  barcode: string | null
  is_pickable: boolean
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface LocationInput {
  code: string
  zone?: string | null
  aisle?: string | null
  rack?: string | null
  shelf?: string | null
  barcode?: string | null
  is_pickable?: boolean
  is_active?: boolean
}

export interface LocationListFilter {
  limit?: number
  after?: string
  search?: string
  is_active?: boolean
  is_pickable?: boolean
}

function buildQuery(
  filter: Record<string, string | number | boolean | undefined>,
): string {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(filter)) {
    if (value !== undefined) {
      params.set(key, String(value))
    }
  }
  const query = params.toString()
  return query ? `?${query}` : ''
}

export function listWarehouses(filter: WarehouseListFilter = {}) {
  return apiRequest<Page<Warehouse>>(`/warehouses${buildQuery(filter)}`)
}

export function getWarehouse(id: string) {
  return apiRequest<Warehouse>(`/warehouses/${id}`)
}

export function createWarehouse(input: WarehouseInput) {
  return apiRequest<Warehouse>('/warehouses', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateWarehouse(id: string, input: WarehouseInput) {
  return apiRequest<Warehouse>(`/warehouses/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export function deactivateWarehouse(id: string) {
  return apiRequest<void>(`/warehouses/${id}`, { method: 'DELETE' })
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npx vitest run src/api/warehouses.test.ts`
Expected: PASS (6 tests).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/api/warehouses.ts frontend/src/api/warehouses.test.ts
git commit -m "feat: add warehouses API client functions"
```

---

## Task 4: `api/warehouses.ts` — location functions

**Files:**
- Modify: `frontend/src/api/warehouses.ts` (append location functions)
- Modify: `frontend/src/api/warehouses.test.ts` (append location tests)

- [ ] **Step 1: Append the failing tests**

Add to `frontend/src/api/warehouses.test.ts`, after the existing imports
add the location functions to the import line:

```ts
import {
  createLocation,
  createWarehouse,
  deactivateLocation,
  deactivateWarehouse,
  getLocation,
  getWarehouse,
  listLocations,
  listWarehouses,
  updateLocation,
  updateWarehouse,
} from '@/api/warehouses'
```

Append a new `describe` block at the end of the file:

```ts
describe('locations api', () => {
  it('lists locations scoped to a warehouse', async () => {
    await listLocations('wh-1', { is_pickable: true })
    expect(mockedApiRequest).toHaveBeenCalledWith(
      '/warehouses/wh-1/locations?is_pickable=true',
    )
  })

  it('lists locations with no filters and no query string', async () => {
    await listLocations('wh-1')
    expect(mockedApiRequest).toHaveBeenCalledWith('/warehouses/wh-1/locations')
  })

  it('gets a single location', async () => {
    await getLocation('wh-1', 'loc-1')
    expect(mockedApiRequest).toHaveBeenCalledWith(
      '/warehouses/wh-1/locations/loc-1',
    )
  })

  it('creates a location scoped to a warehouse', async () => {
    await createLocation('wh-1', { code: 'A-01' })
    expect(mockedApiRequest).toHaveBeenCalledWith('/warehouses/wh-1/locations', {
      method: 'POST',
      body: JSON.stringify({ code: 'A-01' }),
    })
  })

  it('updates a location', async () => {
    await updateLocation('wh-1', 'loc-1', { code: 'A-02' })
    expect(mockedApiRequest).toHaveBeenCalledWith(
      '/warehouses/wh-1/locations/loc-1',
      { method: 'PUT', body: JSON.stringify({ code: 'A-02' }) },
    )
  })

  it('deactivates a location', async () => {
    await deactivateLocation('wh-1', 'loc-1')
    expect(mockedApiRequest).toHaveBeenCalledWith(
      '/warehouses/wh-1/locations/loc-1',
      { method: 'DELETE' },
    )
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/api/warehouses.test.ts`
Expected: FAIL — `listLocations` etc. are not exported yet.

- [ ] **Step 3: Append the implementation**

Add to the end of `frontend/src/api/warehouses.ts`:

```ts
export function listLocations(
  warehouseId: string,
  filter: LocationListFilter = {},
) {
  return apiRequest<Page<Location>>(
    `/warehouses/${warehouseId}/locations${buildQuery(filter)}`,
  )
}

export function getLocation(warehouseId: string, locationId: string) {
  return apiRequest<Location>(
    `/warehouses/${warehouseId}/locations/${locationId}`,
  )
}

export function createLocation(warehouseId: string, input: LocationInput) {
  return apiRequest<Location>(`/warehouses/${warehouseId}/locations`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateLocation(
  warehouseId: string,
  locationId: string,
  input: LocationInput,
) {
  return apiRequest<Location>(
    `/warehouses/${warehouseId}/locations/${locationId}`,
    { method: 'PUT', body: JSON.stringify(input) },
  )
}

export function deactivateLocation(warehouseId: string, locationId: string) {
  return apiRequest<void>(
    `/warehouses/${warehouseId}/locations/${locationId}`,
    { method: 'DELETE' },
  )
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npx vitest run src/api/warehouses.test.ts`
Expected: PASS (12 tests total).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/api/warehouses.ts frontend/src/api/warehouses.test.ts
git commit -m "feat: add locations API client functions"
```

---

## Task 5: `stores/warehouses.ts`

**Files:**
- Create: `frontend/src/stores/warehouses.ts`
- Create: `frontend/src/stores/warehouses.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import * as warehousesApi from '@/api/warehouses'
import { useWarehousesStore } from '@/stores/warehouses'

vi.mock('@/api/warehouses')

const sampleWarehouse: warehousesApi.Warehouse = {
  id: 'wh-1',
  code: 'PP-01',
  name: 'Phnom Penh Main',
  address: null,
  is_active: true,
  created_at: '2026-08-19T00:00:00Z',
  updated_at: '2026-08-19T00:00:00Z',
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
  created_at: '2026-08-19T00:00:00Z',
  updated_at: '2026-08-19T00:00:00Z',
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('useWarehousesStore warehouses', () => {
  it('fetchWarehouses populates state from the API', async () => {
    vi.mocked(warehousesApi.listWarehouses).mockResolvedValue({
      items: [sampleWarehouse],
      page: { next_cursor: null, has_more: false },
    })
    const store = useWarehousesStore()
    await store.fetchWarehouses()
    expect(store.warehouses).toEqual([sampleWarehouse])
    expect(store.loading).toBe(false)
    expect(store.error).toBeNull()
  })

  it('fetchWarehouses records a message on failure', async () => {
    vi.mocked(warehousesApi.listWarehouses).mockRejectedValue(new Error('boom'))
    const store = useWarehousesStore()
    await store.fetchWarehouses()
    expect(store.error).toBe('boom')
    expect(store.loading).toBe(false)
  })

  it('loadMoreWarehouses appends using the stored cursor', async () => {
    vi.mocked(warehousesApi.listWarehouses).mockResolvedValueOnce({
      items: [sampleWarehouse],
      page: { next_cursor: 'cursor-1', has_more: true },
    })
    const store = useWarehousesStore()
    await store.fetchWarehouses()

    const secondWarehouse = { ...sampleWarehouse, id: 'wh-2', code: 'SR-01' }
    vi.mocked(warehousesApi.listWarehouses).mockResolvedValueOnce({
      items: [secondWarehouse],
      page: { next_cursor: null, has_more: false },
    })
    await store.loadMoreWarehouses()

    expect(store.warehouses).toEqual([sampleWarehouse, secondWarehouse])
    expect(warehousesApi.listWarehouses).toHaveBeenLastCalledWith({
      search: undefined,
      after: 'cursor-1',
    })
    expect(store.hasMore).toBe(false)
  })

  it('loadMoreWarehouses does nothing when there is no more to load', async () => {
    const store = useWarehousesStore()
    await store.loadMoreWarehouses()
    expect(warehousesApi.listWarehouses).not.toHaveBeenCalled()
  })

  it('createWarehouse prepends the new warehouse to state', async () => {
    const created = { ...sampleWarehouse, id: 'wh-2', code: 'SR-01' }
    vi.mocked(warehousesApi.createWarehouse).mockResolvedValue(created)
    const store = useWarehousesStore()
    const result = await store.createWarehouse({ code: 'SR-01', name: 'Siem Reap' })
    expect(result).toEqual(created)
    expect(store.warehouses[0]).toEqual(created)
  })

  it('deactivateWarehouse marks the matching warehouse inactive in place', async () => {
    vi.mocked(warehousesApi.listWarehouses).mockResolvedValue({
      items: [sampleWarehouse],
      page: { next_cursor: null, has_more: false },
    })
    vi.mocked(warehousesApi.deactivateWarehouse).mockResolvedValue(undefined)
    const store = useWarehousesStore()
    await store.fetchWarehouses()
    await store.deactivateWarehouse('wh-1')
    expect(store.warehouses[0].is_active).toBe(false)
  })
})

describe('useWarehousesStore locations', () => {
  it('fetchLocations populates location state', async () => {
    vi.mocked(warehousesApi.listLocations).mockResolvedValue({
      items: [sampleLocation],
      page: { next_cursor: null, has_more: false },
    })
    const store = useWarehousesStore()
    await store.fetchLocations('wh-1')
    expect(store.locations).toEqual([sampleLocation])
    expect(store.locationsLoading).toBe(false)
  })

  it('loadMoreLocations appends using the stored cursor', async () => {
    vi.mocked(warehousesApi.listLocations).mockResolvedValueOnce({
      items: [sampleLocation],
      page: { next_cursor: 'cursor-1', has_more: true },
    })
    const store = useWarehousesStore()
    await store.fetchLocations('wh-1')

    const secondLocation = { ...sampleLocation, id: 'loc-2', code: 'A-02' }
    vi.mocked(warehousesApi.listLocations).mockResolvedValueOnce({
      items: [secondLocation],
      page: { next_cursor: null, has_more: false },
    })
    await store.loadMoreLocations('wh-1')

    expect(store.locations).toEqual([sampleLocation, secondLocation])
    expect(warehousesApi.listLocations).toHaveBeenLastCalledWith('wh-1', {
      after: 'cursor-1',
    })
  })

  it('createLocation prepends the new location to state', async () => {
    const created = { ...sampleLocation, id: 'loc-2', code: 'A-02' }
    vi.mocked(warehousesApi.createLocation).mockResolvedValue(created)
    const store = useWarehousesStore()
    const result = await store.createLocation('wh-1', { code: 'A-02' })
    expect(result).toEqual(created)
    expect(store.locations[0]).toEqual(created)
  })

  it('deactivateLocation marks the matching location inactive in place', async () => {
    vi.mocked(warehousesApi.listLocations).mockResolvedValue({
      items: [sampleLocation],
      page: { next_cursor: null, has_more: false },
    })
    vi.mocked(warehousesApi.deactivateLocation).mockResolvedValue(undefined)
    const store = useWarehousesStore()
    await store.fetchLocations('wh-1')
    await store.deactivateLocation('wh-1', 'loc-1')
    expect(store.locations[0].is_active).toBe(false)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/stores/warehouses.test.ts`
Expected: FAIL — `Failed to resolve import "@/stores/warehouses"`.

- [ ] **Step 3: Write the implementation**

```ts
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npx vitest run src/stores/warehouses.test.ts`
Expected: PASS (10 tests).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/stores/warehouses.ts frontend/src/stores/warehouses.test.ts
git commit -m "feat: add warehouses Pinia store"
```

---

## Task 6: `components/WarehouseFormDialog.vue`

**Files:**
- Create: `frontend/src/components/WarehouseFormDialog.vue`
- Create: `frontend/src/components/WarehouseFormDialog.spec.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
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
    global: { stubs: { teleport: true } },
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/components/WarehouseFormDialog.spec.ts`
Expected: FAIL — the component file does not exist yet.

- [ ] **Step 3: Write the implementation**

```vue
<script setup lang="ts">
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'
import ToggleSwitch from 'primevue/toggleswitch'
import { computed, reactive, ref, watch } from 'vue'

import { ApiClientError } from '@/api/client'
import type { Warehouse, WarehouseInput } from '@/api/warehouses'
import { useWarehousesStore } from '@/stores/warehouses'

const props = defineProps<{
  visible: boolean
  warehouse: Warehouse | null
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  saved: []
}>()

const store = useWarehousesStore()
const isEdit = computed(() => props.warehouse !== null)

const form = reactive({
  code: '',
  name: '',
  address: '',
  is_active: true,
})

const codeError = ref<string | null>(null)
const nameError = ref<string | null>(null)
const generalError = ref<string | null>(null)
const submitting = ref(false)

watch(
  () => props.visible,
  (visible) => {
    if (!visible) {
      return
    }
    codeError.value = null
    nameError.value = null
    generalError.value = null
    form.code = props.warehouse?.code ?? ''
    form.name = props.warehouse?.name ?? ''
    form.address = props.warehouse?.address ?? ''
    form.is_active = props.warehouse?.is_active ?? true
  },
  { immediate: true },
)

function validate(): boolean {
  codeError.value = form.code.trim() === '' ? 'Code is required.' : null
  nameError.value = form.name.trim() === '' ? 'Name is required.' : null
  return codeError.value === null && nameError.value === null
}

async function submit() {
  generalError.value = null
  if (!validate()) {
    return
  }
  submitting.value = true
  try {
    const input: WarehouseInput = {
      code: form.code,
      name: form.name,
      address: form.address || null,
      is_active: form.is_active,
    }
    if (isEdit.value && props.warehouse) {
      await store.updateWarehouse(props.warehouse.id, input)
    } else {
      await store.createWarehouse(input)
    }
    emit('saved')
    emit('update:visible', false)
  } catch (err) {
    if (err instanceof ApiClientError && err.code === 'WAREHOUSE_CODE_CONFLICT') {
      codeError.value = 'This code is already in use.'
    } else if (err instanceof ApiClientError) {
      generalError.value = err.message
    } else {
      generalError.value = 'Something went wrong. Try again.'
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    :header="isEdit ? 'Edit warehouse' : 'Add warehouse'"
    @update:visible="(value: boolean) => emit('update:visible', value)"
  >
    <div class="form-field">
      <label for="warehouse-code">Code</label>
      <InputText id="warehouse-code" v-model="form.code" :invalid="codeError !== null" />
      <Message v-if="codeError" severity="error" size="small" variant="simple">{{ codeError }}</Message>
    </div>
    <div class="form-field">
      <label for="warehouse-name">Name</label>
      <InputText id="warehouse-name" v-model="form.name" :invalid="nameError !== null" />
      <Message v-if="nameError" severity="error" size="small" variant="simple">{{ nameError }}</Message>
    </div>
    <div class="form-field">
      <label for="warehouse-address">Address</label>
      <InputText id="warehouse-address" v-model="form.address" />
    </div>
    <div class="form-field form-field--inline">
      <label for="warehouse-active">Active</label>
      <ToggleSwitch id="warehouse-active" v-model="form.is_active" />
    </div>
    <Message v-if="generalError" severity="error" size="small">{{ generalError }}</Message>
    <template #footer>
      <Button label="Cancel" severity="secondary" data-testid="cancel" @click="emit('update:visible', false)" />
      <Button label="Save" :loading="submitting" data-testid="submit" @click="submit" />
    </template>
  </Dialog>
</template>

<style scoped>
.form-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-bottom: 16px;
}
.form-field--inline {
  flex-direction: row;
  align-items: center;
  gap: 12px;
}
</style>
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npx vitest run src/components/WarehouseFormDialog.spec.ts`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/WarehouseFormDialog.vue frontend/src/components/WarehouseFormDialog.spec.ts
git commit -m "feat: add WarehouseFormDialog component"
```

---

## Task 7: `components/LocationFormDialog.vue`

**Files:**
- Create: `frontend/src/components/LocationFormDialog.vue`
- Create: `frontend/src/components/LocationFormDialog.spec.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
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
    global: { stubs: { teleport: true } },
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
    await wrapper.find('#location-barcode').setValue('8851234567890')
    await wrapper.find('[data-testid="submit"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('This barcode is already in use.')
  })

  it('shows a valid-checksum hint for a well-formed EAN-13 barcode', async () => {
    const wrapper = mountDialog()
    await wrapper.find('#location-barcode').setValue('8851234567890')
    expect(wrapper.find('[data-testid="barcode-hint"]').text()).toContain('Valid EAN-13')
  })
})

async function flushPromises() {
  await new Promise((resolve) => setTimeout(resolve, 0))
}
```

Before implementing, verify `8851234567890` is actually a valid EAN-13
checksum against `validateBarcode` — if the test's own fixture value is
wrong, the last test will fail for the wrong reason. Compute it or run
`node -e "console.log(require('./src/lib/barcode.ts'))"`-style check
mentally: the existing `barcode.test.ts` file already has known-valid
fixtures — reuse one of those exact values instead of inventing a new one
if there's any doubt.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/components/LocationFormDialog.spec.ts`
Expected: FAIL — the component file does not exist yet.

- [ ] **Step 3: Write the implementation**

```vue
<script setup lang="ts">
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'
import ToggleSwitch from 'primevue/toggleswitch'
import { computed, reactive, ref, watch } from 'vue'

import { ApiClientError } from '@/api/client'
import type { Location, LocationInput } from '@/api/warehouses'
import { validateBarcode } from '@/lib/barcode'
import { useWarehousesStore } from '@/stores/warehouses'

const props = defineProps<{
  visible: boolean
  warehouseId: string
  location: Location | null
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  saved: []
}>()

const store = useWarehousesStore()
const isEdit = computed(() => props.location !== null)

const form = reactive({
  code: '',
  zone: '',
  aisle: '',
  rack: '',
  shelf: '',
  barcode: '',
  is_pickable: true,
  is_active: true,
})

const codeError = ref<string | null>(null)
const barcodeError = ref<string | null>(null)
const generalError = ref<string | null>(null)
const submitting = ref(false)

const barcodeHint = computed(() => {
  if (form.barcode.trim() === '') {
    return null
  }
  const result = validateBarcode(form.barcode)
  return result.valid ? `Valid ${result.format}` : result.message
})

watch(
  () => props.visible,
  (visible) => {
    if (!visible) {
      return
    }
    codeError.value = null
    barcodeError.value = null
    generalError.value = null
    form.code = props.location?.code ?? ''
    form.zone = props.location?.zone ?? ''
    form.aisle = props.location?.aisle ?? ''
    form.rack = props.location?.rack ?? ''
    form.shelf = props.location?.shelf ?? ''
    form.barcode = props.location?.barcode ?? ''
    form.is_pickable = props.location?.is_pickable ?? true
    form.is_active = props.location?.is_active ?? true
  },
  { immediate: true },
)

function validate(): boolean {
  codeError.value = form.code.trim() === '' ? 'Code is required.' : null
  return codeError.value === null
}

async function submit() {
  generalError.value = null
  barcodeError.value = null
  if (!validate()) {
    return
  }
  submitting.value = true
  try {
    const input: LocationInput = {
      code: form.code,
      zone: form.zone || null,
      aisle: form.aisle || null,
      rack: form.rack || null,
      shelf: form.shelf || null,
      barcode: form.barcode || null,
      is_pickable: form.is_pickable,
      is_active: form.is_active,
    }
    if (isEdit.value && props.location) {
      await store.updateLocation(props.warehouseId, props.location.id, input)
    } else {
      await store.createLocation(props.warehouseId, input)
    }
    emit('saved')
    emit('update:visible', false)
  } catch (err) {
    if (err instanceof ApiClientError && err.code === 'LOCATION_CODE_CONFLICT') {
      codeError.value = 'This code is already in use in this warehouse.'
    } else if (err instanceof ApiClientError && err.code === 'LOCATION_BARCODE_CONFLICT') {
      barcodeError.value = 'This barcode is already in use.'
    } else if (err instanceof ApiClientError) {
      generalError.value = err.message
    } else {
      generalError.value = 'Something went wrong. Try again.'
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    :header="isEdit ? 'Edit location' : 'Add location'"
    @update:visible="(value: boolean) => emit('update:visible', value)"
  >
    <div class="form-field">
      <label for="location-code">Code</label>
      <InputText id="location-code" v-model="form.code" :invalid="codeError !== null" />
      <Message v-if="codeError" severity="error" size="small" variant="simple">{{ codeError }}</Message>
    </div>
    <div class="form-field">
      <label for="location-zone">Zone</label>
      <InputText id="location-zone" v-model="form.zone" />
    </div>
    <div class="form-field">
      <label for="location-aisle">Aisle</label>
      <InputText id="location-aisle" v-model="form.aisle" />
    </div>
    <div class="form-field">
      <label for="location-rack">Rack</label>
      <InputText id="location-rack" v-model="form.rack" />
    </div>
    <div class="form-field">
      <label for="location-shelf">Shelf</label>
      <InputText id="location-shelf" v-model="form.shelf" />
    </div>
    <div class="form-field">
      <label for="location-barcode">Barcode</label>
      <InputText id="location-barcode" v-model="form.barcode" :invalid="barcodeError !== null" />
      <Message v-if="barcodeError" severity="error" size="small" variant="simple">{{ barcodeError }}</Message>
      <small v-else-if="barcodeHint" data-testid="barcode-hint">{{ barcodeHint }}</small>
    </div>
    <div class="form-field form-field--inline">
      <label for="location-pickable">Pickable</label>
      <ToggleSwitch id="location-pickable" v-model="form.is_pickable" />
    </div>
    <div class="form-field form-field--inline">
      <label for="location-active">Active</label>
      <ToggleSwitch id="location-active" v-model="form.is_active" />
    </div>
    <Message v-if="generalError" severity="error" size="small">{{ generalError }}</Message>
    <template #footer>
      <Button label="Cancel" severity="secondary" data-testid="cancel" @click="emit('update:visible', false)" />
      <Button label="Save" :loading="submitting" data-testid="submit" @click="submit" />
    </template>
  </Dialog>
</template>

<style scoped>
.form-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-bottom: 16px;
}
.form-field--inline {
  flex-direction: row;
  align-items: center;
  gap: 12px;
}
</style>
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npx vitest run src/components/LocationFormDialog.spec.ts`
Expected: PASS (4 tests). If the barcode-hint test fails because the
fixture digit string isn't actually a valid checksum, replace it with a
value copied verbatim from `frontend/src/lib/barcode.test.ts`'s own valid
fixtures — do not hand-invent another one.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/LocationFormDialog.vue frontend/src/components/LocationFormDialog.spec.ts
git commit -m "feat: add LocationFormDialog component"
```

---

## Task 8: `views/WarehousesListView.vue`

**Files:**
- Create: `frontend/src/views/WarehousesListView.vue`
- Create: `frontend/src/views/WarehousesListView.spec.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
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
  return mount(WarehousesListView, { global: { stubs: { teleport: true } } })
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/views/WarehousesListView.spec.ts`
Expected: FAIL — the view file does not exist yet.

- [ ] **Step 3: Write the implementation**

```vue
<script setup lang="ts">
import Button from 'primevue/button'
import Column from 'primevue/column'
import DataTable from 'primevue/datatable'
import InputText from 'primevue/inputtext'
import Tag from 'primevue/tag'
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'

import type { Warehouse } from '@/api/warehouses'
import WarehouseFormDialog from '@/components/WarehouseFormDialog.vue'
import { useAuthStore } from '@/stores/auth'
import { useWarehousesStore } from '@/stores/warehouses'

const store = useWarehousesStore()
const auth = useAuthStore()
const router = useRouter()

const search = ref('')
const dialogVisible = ref(false)
const editingWarehouse = ref<Warehouse | null>(null)

const canManageWarehouses = computed(() => auth.user?.role === 'admin')

onMounted(() => {
  store.fetchWarehouses()
})

let searchTimeout: ReturnType<typeof setTimeout> | undefined
watch(search, (value) => {
  if (searchTimeout) {
    clearTimeout(searchTimeout)
  }
  searchTimeout = setTimeout(() => {
    store.fetchWarehouses(value)
  }, 300)
})

function openCreateDialog() {
  editingWarehouse.value = null
  dialogVisible.value = true
}

function openEditDialog(warehouse: Warehouse) {
  editingWarehouse.value = warehouse
  dialogVisible.value = true
}

async function deactivate(warehouse: Warehouse) {
  await store.deactivateWarehouse(warehouse.id)
}

function openDetail(warehouse: Warehouse) {
  router.push({ name: 'warehouse-detail', params: { warehouseId: warehouse.id } })
}
</script>

<template>
  <div class="page">
    <div class="page-header">
      <h1>Warehouses</h1>
      <Button
        v-if="canManageWarehouses"
        label="Add Warehouse"
        data-testid="add-warehouse"
        @click="openCreateDialog"
      />
    </div>

    <InputText
      v-model="search"
      placeholder="Search warehouses..."
      data-testid="warehouse-search"
      class="search-field"
    />

    <p v-if="store.error" class="error-banner" data-testid="warehouses-error">{{ store.error }}</p>

    <DataTable
      v-else
      :value="store.warehouses"
      :loading="store.loading"
      data-key="id"
      @row-click="(event: { data: Warehouse }) => openDetail(event.data)"
    >
      <template #empty>
        <p>No warehouses yet.</p>
      </template>
      <Column field="code" header="Code" />
      <Column field="name" header="Name" />
      <Column field="address" header="Address" />
      <Column header="Status">
        <template #body="{ data }">
          <Tag :severity="data.is_active ? 'success' : 'danger'" :value="data.is_active ? 'Active' : 'Inactive'" />
        </template>
      </Column>
      <Column v-if="canManageWarehouses" header="Actions">
        <template #body="{ data }">
          <Button
            label="Edit"
            size="small"
            severity="secondary"
            data-testid="edit-warehouse"
            @click.stop="openEditDialog(data)"
          />
          <Button
            v-if="data.is_active"
            label="Deactivate"
            size="small"
            severity="danger"
            data-testid="deactivate-warehouse"
            @click.stop="deactivate(data)"
          />
        </template>
      </Column>
    </DataTable>

    <Button
      v-if="store.hasMore"
      label="Load more"
      severity="secondary"
      data-testid="load-more"
      @click="store.loadMoreWarehouses(search)"
    />

    <WarehouseFormDialog v-model:visible="dialogVisible" :warehouse="editingWarehouse" />
  </div>
</template>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.search-field {
  max-width: 320px;
}
.error-banner {
  color: var(--p-red-600, #dc2626);
}
</style>
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npx vitest run src/views/WarehousesListView.spec.ts`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/WarehousesListView.vue frontend/src/views/WarehousesListView.spec.ts
git commit -m "feat: add WarehousesListView"
```

---

## Task 9: `views/WarehouseDetailView.vue`

**Files:**
- Create: `frontend/src/views/WarehouseDetailView.vue`
- Create: `frontend/src/views/WarehouseDetailView.spec.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
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
  return mount(WarehouseDetailView, { global: { stubs: { teleport: true } } })
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/views/WarehouseDetailView.spec.ts`
Expected: FAIL — the view file does not exist yet.

- [ ] **Step 3: Write the implementation**

```vue
<script setup lang="ts">
import Button from 'primevue/button'
import Column from 'primevue/column'
import DataTable from 'primevue/datatable'
import Tag from 'primevue/tag'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'

import type { Location, Warehouse } from '@/api/warehouses'
import { getWarehouse } from '@/api/warehouses'
import LocationFormDialog from '@/components/LocationFormDialog.vue'
import WarehouseFormDialog from '@/components/WarehouseFormDialog.vue'
import { useAuthStore } from '@/stores/auth'
import { useWarehousesStore } from '@/stores/warehouses'

const route = useRoute()
const store = useWarehousesStore()
const auth = useAuthStore()

const warehouseId = computed(() => route.params.warehouseId as string)
const warehouse = ref<Warehouse | null>(null)
const warehouseError = ref<string | null>(null)

const canManageWarehouses = computed(() => auth.user?.role === 'admin')
const canManageLocations = computed(
  () => auth.user?.role === 'admin' || auth.user?.role === 'warehouse_manager',
)

const warehouseDialogVisible = ref(false)
const locationDialogVisible = ref(false)
const editingLocation = ref<Location | null>(null)

async function loadWarehouse() {
  warehouseError.value = null
  try {
    warehouse.value = await getWarehouse(warehouseId.value)
  } catch (err) {
    warehouseError.value = err instanceof Error ? err.message : 'Failed to load warehouse.'
  }
}

onMounted(() => {
  loadWarehouse()
  store.fetchLocations(warehouseId.value)
})

watch(warehouseId, () => {
  loadWarehouse()
  store.fetchLocations(warehouseId.value)
})

function openCreateLocationDialog() {
  editingLocation.value = null
  locationDialogVisible.value = true
}

function openEditLocationDialog(location: Location) {
  editingLocation.value = location
  locationDialogVisible.value = true
}

async function deactivateLocationRow(location: Location) {
  await store.deactivateLocation(warehouseId.value, location.id)
}

function handleWarehouseSaved() {
  loadWarehouse()
}
</script>

<template>
  <div class="page">
    <p v-if="warehouseError" class="error-banner" data-testid="warehouse-error">{{ warehouseError }}</p>
    <template v-else-if="warehouse">
      <div class="page-header">
        <div>
          <h1>{{ warehouse.name }}</h1>
          <p>{{ warehouse.code }} · {{ warehouse.address ?? 'No address on file' }}</p>
        </div>
        <Button
          v-if="canManageWarehouses"
          label="Edit Warehouse"
          data-testid="edit-warehouse"
          @click="warehouseDialogVisible = true"
        />
      </div>

      <div class="page-header">
        <h2>Locations</h2>
        <Button
          v-if="canManageLocations"
          label="Add Location"
          data-testid="add-location"
          @click="openCreateLocationDialog"
        />
      </div>

      <p v-if="store.locationsError" class="error-banner" data-testid="locations-error">
        {{ store.locationsError }}
      </p>
      <DataTable v-else :value="store.locations" :loading="store.locationsLoading" data-key="id">
        <template #empty>
          <p>No locations yet.</p>
        </template>
        <Column field="code" header="Code" />
        <Column field="zone" header="Zone" />
        <Column field="aisle" header="Aisle" />
        <Column field="rack" header="Rack" />
        <Column field="shelf" header="Shelf" />
        <Column field="barcode" header="Barcode" />
        <Column header="Pickable">
          <template #body="{ data }">
            <Tag :severity="data.is_pickable ? 'info' : 'secondary'" :value="data.is_pickable ? 'Yes' : 'No'" />
          </template>
        </Column>
        <Column header="Status">
          <template #body="{ data }">
            <Tag :severity="data.is_active ? 'success' : 'danger'" :value="data.is_active ? 'Active' : 'Inactive'" />
          </template>
        </Column>
        <Column v-if="canManageLocations" header="Actions">
          <template #body="{ data }">
            <Button
              label="Edit"
              size="small"
              severity="secondary"
              data-testid="edit-location"
              @click="openEditLocationDialog(data)"
            />
            <Button
              v-if="data.is_active"
              label="Deactivate"
              size="small"
              severity="danger"
              data-testid="deactivate-location"
              @click="deactivateLocationRow(data)"
            />
          </template>
        </Column>
      </DataTable>

      <Button
        v-if="store.locationsHasMore"
        label="Load more"
        severity="secondary"
        data-testid="load-more-locations"
        @click="store.loadMoreLocations(warehouseId)"
      />

      <WarehouseFormDialog
        v-model:visible="warehouseDialogVisible"
        :warehouse="warehouse"
        @saved="handleWarehouseSaved"
      />
      <LocationFormDialog
        v-model:visible="locationDialogVisible"
        :warehouse-id="warehouseId"
        :location="editingLocation"
      />
    </template>
  </div>
</template>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.error-banner {
  color: var(--p-red-600, #dc2626);
}
</style>
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npx vitest run src/views/WarehouseDetailView.spec.ts`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/WarehouseDetailView.vue frontend/src/views/WarehouseDetailView.spec.ts
git commit -m "feat: add WarehouseDetailView"
```

---

## Task 10: Wire up routing and navigation

**Files:**
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/layouts/AppLayout.vue`

- [ ] **Step 1: Add the two routes**

In `frontend/src/router/index.ts`, inside the `AppLayout` route's
`children` array (alongside the existing `dashboard` and `barcode-test`
entries), add:

```ts
        {
          path: 'warehouses',
          name: 'warehouses',
          component: () => import('@/views/WarehousesListView.vue'),
        },
        {
          path: 'warehouses/:warehouseId',
          name: 'warehouse-detail',
          component: () => import('@/views/WarehouseDetailView.vue'),
        },
```

No `meta: { public: true }` — both should require authentication, which
is the default (the existing guard only treats routes with
`meta.public` as public).

- [ ] **Step 2: Add a real nav link and refresh the stale sidebar note**

In `frontend/src/layouts/AppLayout.vue`, replace:

```html
        <RouterLink to="/" class="nav-link">Dashboard</RouterLink>
        <RouterLink to="/barcode-test" class="nav-link">Barcode test</RouterLink>
        <span class="nav-link nav-link--disabled">Inventory</span>
        <span class="nav-link nav-link--disabled">Movements</span>
```

with:

```html
        <RouterLink to="/" class="nav-link">Dashboard</RouterLink>
        <RouterLink to="/warehouses" class="nav-link">Warehouses</RouterLink>
        <RouterLink to="/barcode-test" class="nav-link">Barcode test</RouterLink>
        <span class="nav-link nav-link--disabled">Inventory</span>
        <span class="nav-link nav-link--disabled">Movements</span>
```

And replace the stale milestone note:

```html
      <div class="sidebar-note">
        <small>Week 6–7 foundation</small>
        <span>Camera validation and authentication are the current milestone.</span>
      </div>
```

with:

```html
      <div class="sidebar-note">
        <small>Week 11</small>
        <span>Backend complete through inventory & movements. Warehouses is the first live frontend screen.</span>
      </div>
```

- [ ] **Step 3: Manually verify in the browser**

```bash
cd backend && go run ./cmd/api &
cd frontend && npm run dev
```

Log in as the seeded admin (`admin@bwims.local` / `ChangeMe123!`), click
"Warehouses" in the sidebar, confirm the list loads (empty state is fine
if no warehouses exist yet — create one via the "Add Warehouse" button to
confirm the whole loop works), click into its detail page, add a
location, confirm it appears in the table.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/router/index.ts frontend/src/layouts/AppLayout.vue
git commit -m "feat: wire warehouses routes and navigation into the app shell"
```

---

## Task 11: Final validation

**Files:** none (verification only)

- [ ] **Step 1: Run the full frontend test suite**

```bash
cd frontend
npm test
```

Expected: PASS — every test file from Tasks 1–9, plus the pre-existing
`lib/barcode.test.ts`.

- [ ] **Step 2: Typecheck**

```bash
cd frontend
npm run typecheck
```

Expected: PASS, no `vue-tsc` errors.

- [ ] **Step 3: Build**

```bash
cd frontend
npm run build
```

Expected: PASS. If bundle-size warnings appear, confirm they're the same
pre-existing `BarcodeTestView` chunk-size warning noted in
`docs/week-8-catalog-validation.md`, not a new regression.

- [ ] **Step 4: Confirm the backend regression suite is still clean**

```bash
cd backend
go build ./...
go vet ./...
go test ./...
```

Expected: PASS (no backend files were touched in this plan, this is a
sanity check, not expected to find anything).

- [ ] **Step 5: No commit for this task** — verification checkpoint only.

---

## Self-Review Notes

**Spec coverage check against the design doc:**
- `api/warehouses.ts` mirrors `api/auth.ts`, no new HTTP client — Task 3/4. ✓
- Pinia composition-style store — Task 5. ✓
- RBAC: `canManageWarehouses`/`canManageLocations` gating buttons, no
  client-side warehouse-scoping (server already scopes reads) — Tasks 8/9. ✓
- Cursor-based "Load more" instead of PrimeVue's numbered paginator —
  Tasks 5/8/9. ✓
- Location barcode field reuses `lib/barcode.ts`'s `validateBarcode`, not
  a reimplementation — Task 7 (and its hint field verified against the
  real `BarcodeValidation` shape: `normalized`/`format`/`valid`/`message`,
  not the `reason` field an earlier draft of this plan mistakenly assumed). ✓
- `@vue/test-utils` + jsdom added — Task 1. ✓
- Form dialogs surface `*_CODE_CONFLICT`/`*_BARCODE_CONFLICT` on the
  specific field, not a generic toast — Tasks 6/7. ✓

**Type/signature consistency check:** `Warehouse`, `WarehouseInput`,
`Location`, `LocationInput` are defined once in Task 3/4's
`api/warehouses.ts` and reused with identical field names in the store
(Task 5), both form dialogs (Tasks 6/7), and both views (Tasks 8/9) — no
renamed fields between tasks. Store action names
(`fetchWarehouses`/`loadMoreWarehouses`/`createWarehouse`/`updateWarehouse`/
`deactivateWarehouse` and their `*Location` counterparts) match exactly
between the store's own tests (Task 5) and every consumer (Tasks 6–9).

**Deviation from the design doc, flagged explicitly:** the design doc's
component list didn't specify exact `data-testid` attribute names for
interactive elements. This plan adds them (`add-warehouse`,
`edit-warehouse`, `deactivate-warehouse`, `submit`, `cancel`, etc.)
throughout, since reliable test selectors need something more stable than
button text — a natural implementation detail, not a scope change.
