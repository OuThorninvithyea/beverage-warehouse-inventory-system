# Catalog (Products & Categories) Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Products list screen (search, filter, CRUD) and a Categories management dialog against the already-complete `/api/v1/products` and `/api/v1/categories` API, following the same TDD pattern and Tailwind/Figma visual language established by the Warehouses & Locations slice.

**Architecture:** Typed `api/catalog.ts` functions (mirrors `api/warehouses.ts`), a Pinia store (`stores/catalog.ts`) holding both products and categories state, one routed view (`ProductsView.vue`), and two dialogs (`ProductFormDialog.vue`, `CategoryFormDialog.vue`). Category names are resolved client-side from the loaded categories list since `Product` only stores `category_id`.

**Tech Stack:** Vue 3 (Composition API, `<script setup>`), TypeScript, Pinia, PrimeVue 5, Tailwind CSS (existing `@theme` tokens), Vitest + `@vue/test-utils` + jsdom.

Design doc: `docs/superpowers/specs/2026-08-19-catalog-frontend-design.md`.

---

## File Structure

```
frontend/src/
  api/catalog.ts                       # CREATE
  api/catalog.test.ts                  # CREATE
  stores/catalog.ts                    # CREATE
  stores/catalog.test.ts               # CREATE
  components/ProductFormDialog.vue     # CREATE
  components/ProductFormDialog.spec.ts # CREATE
  components/CategoryFormDialog.vue    # CREATE
  components/CategoryFormDialog.spec.ts # CREATE
  views/ProductsView.vue               # CREATE
  views/ProductsView.spec.ts           # CREATE
  router/index.ts                      # MODIFY — add `products` route
  layouts/AppLayout.vue                # MODIFY — add "Products" nav link
```

---

## Task 1: `api/catalog.ts` — category functions

**Files:**
- Create: `frontend/src/api/catalog.ts`
- Create: `frontend/src/api/catalog.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { apiRequest } from '@/api/client'
import {
  createCategory,
  deactivateCategory,
  getCategory,
  listCategories,
  updateCategory,
} from '@/api/catalog'

vi.mock('@/api/client', () => ({
  apiRequest: vi.fn(),
}))

const mockedApiRequest = vi.mocked(apiRequest)

beforeEach(() => {
  mockedApiRequest.mockReset()
  mockedApiRequest.mockResolvedValue({} as never)
})

describe('categories api', () => {
  it('lists categories with no filters and no query string', async () => {
    await listCategories()
    expect(mockedApiRequest).toHaveBeenCalledWith('/categories')
  })

  it('lists categories with filters as query params', async () => {
    await listCategories({ search: 'Soft', is_active: true })
    expect(mockedApiRequest).toHaveBeenCalledWith('/categories?search=Soft&is_active=true')
  })

  it('gets a single category by id', async () => {
    await getCategory('cat-1')
    expect(mockedApiRequest).toHaveBeenCalledWith('/categories/cat-1')
  })

  it('creates a category with a POST body', async () => {
    await createCategory({ name: 'Soft Drinks' })
    expect(mockedApiRequest).toHaveBeenCalledWith('/categories', {
      method: 'POST',
      body: JSON.stringify({ name: 'Soft Drinks' }),
    })
  })

  it('updates a category with a PUT body', async () => {
    await updateCategory('cat-1', { name: 'Renamed' })
    expect(mockedApiRequest).toHaveBeenCalledWith('/categories/cat-1', {
      method: 'PUT',
      body: JSON.stringify({ name: 'Renamed' }),
    })
  })

  it('deactivates a category with DELETE', async () => {
    await deactivateCategory('cat-1')
    expect(mockedApiRequest).toHaveBeenCalledWith('/categories/cat-1', {
      method: 'DELETE',
    })
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/api/catalog.test.ts`
Expected: FAIL — `Failed to resolve import "@/api/catalog"`.

- [ ] **Step 3: Write the implementation**

```ts
import { apiRequest } from '@/api/client'
import type { Page } from '@/types/pagination'

export interface Category {
  id: string
  parent_id: string | null
  name: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CategoryInput {
  name: string
  parent_id?: string | null
  is_active?: boolean
}

export interface CategoryListFilter {
  limit?: number
  after?: string
  search?: string
  parent_id?: string
  is_active?: boolean
}

function buildQuery(filter: object): string {
  const params = new URLSearchParams()
  const entries = Object.entries(
    filter as Record<string, string | number | boolean | undefined>,
  )
  for (const [key, value] of entries) {
    if (value !== undefined) {
      params.set(key, String(value))
    }
  }
  const query = params.toString()
  return query ? `?${query}` : ''
}

export function listCategories(filter: CategoryListFilter = {}) {
  return apiRequest<Page<Category>>(`/categories${buildQuery(filter)}`)
}

export function getCategory(id: string) {
  return apiRequest<Category>(`/categories/${id}`)
}

export function createCategory(input: CategoryInput) {
  return apiRequest<Category>('/categories', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateCategory(id: string, input: CategoryInput) {
  return apiRequest<Category>(`/categories/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export function deactivateCategory(id: string) {
  return apiRequest<void>(`/categories/${id}`, { method: 'DELETE' })
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npx vitest run src/api/catalog.test.ts`
Expected: PASS (6 tests).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/api/catalog.ts frontend/src/api/catalog.test.ts
git commit -m "feat: add categories API client functions"
```

---

## Task 2: `api/catalog.ts` — product functions

**Files:**
- Modify: `frontend/src/api/catalog.ts` (append product functions)
- Modify: `frontend/src/api/catalog.test.ts` (append product tests)

- [ ] **Step 1: Append the failing tests**

Add to the import line in `frontend/src/api/catalog.test.ts`:

```ts
import {
  createCategory,
  createProduct,
  deactivateCategory,
  deactivateProduct,
  getCategory,
  getProduct,
  listCategories,
  listProducts,
  updateCategory,
  updateProduct,
} from '@/api/catalog'
```

Append a new `describe` block at the end of the file:

```ts
describe('products api', () => {
  it('lists products with no filters and no query string', async () => {
    await listProducts()
    expect(mockedApiRequest).toHaveBeenCalledWith('/products')
  })

  it('lists products with filters as query params', async () => {
    await listProducts({ search: 'coke', category_id: 'cat-1', is_active: true })
    expect(mockedApiRequest).toHaveBeenCalledWith(
      '/products?search=coke&category_id=cat-1&is_active=true',
    )
  })

  it('gets a single product', async () => {
    await getProduct('prod-1')
    expect(mockedApiRequest).toHaveBeenCalledWith('/products/prod-1')
  })

  it('creates a product with a POST body', async () => {
    await createProduct({ sku: 'COKE-330', name: 'Coca-Cola 330ml', unit: 'case' })
    expect(mockedApiRequest).toHaveBeenCalledWith('/products', {
      method: 'POST',
      body: JSON.stringify({ sku: 'COKE-330', name: 'Coca-Cola 330ml', unit: 'case' }),
    })
  })

  it('updates a product', async () => {
    await updateProduct('prod-1', { sku: 'COKE-330', name: 'Renamed', unit: 'case' })
    expect(mockedApiRequest).toHaveBeenCalledWith('/products/prod-1', {
      method: 'PUT',
      body: JSON.stringify({ sku: 'COKE-330', name: 'Renamed', unit: 'case' }),
    })
  })

  it('deactivates a product', async () => {
    await deactivateProduct('prod-1')
    expect(mockedApiRequest).toHaveBeenCalledWith('/products/prod-1', {
      method: 'DELETE',
    })
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/api/catalog.test.ts`
Expected: FAIL — product functions are not exported yet.

- [ ] **Step 3: Append the implementation**

Add to the end of `frontend/src/api/catalog.ts`:

```ts
export interface Product {
  id: string
  category_id: string | null
  sku: string
  barcode: string | null
  name: string
  unit: string
  is_lot_tracked: boolean
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface ProductInput {
  category_id?: string | null
  sku: string
  barcode?: string | null
  name: string
  unit: string
  is_lot_tracked?: boolean
  is_active?: boolean
}

export interface ProductListFilter {
  limit?: number
  after?: string
  search?: string
  category_id?: string
  is_active?: boolean
}

export function listProducts(filter: ProductListFilter = {}) {
  return apiRequest<Page<Product>>(`/products${buildQuery(filter)}`)
}

export function getProduct(id: string) {
  return apiRequest<Product>(`/products/${id}`)
}

export function createProduct(input: ProductInput) {
  return apiRequest<Product>('/products', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateProduct(id: string, input: ProductInput) {
  return apiRequest<Product>(`/products/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export function deactivateProduct(id: string) {
  return apiRequest<void>(`/products/${id}`, { method: 'DELETE' })
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npx vitest run src/api/catalog.test.ts`
Expected: PASS (12 tests total).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/api/catalog.ts frontend/src/api/catalog.test.ts
git commit -m "feat: add products API client functions"
```

---

## Task 3: `stores/catalog.ts`

**Files:**
- Create: `frontend/src/stores/catalog.ts`
- Create: `frontend/src/stores/catalog.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import * as catalogApi from '@/api/catalog'
import { useCatalogStore } from '@/stores/catalog'

vi.mock('@/api/catalog')

const sampleCategory: catalogApi.Category = {
  id: 'cat-1',
  parent_id: null,
  name: 'Soft Drinks',
  is_active: true,
  created_at: '2026-08-19T00:00:00Z',
  updated_at: '2026-08-19T00:00:00Z',
}

const sampleProduct: catalogApi.Product = {
  id: 'prod-1',
  category_id: 'cat-1',
  sku: 'COKE-330',
  barcode: '4006381333931',
  name: 'Coca-Cola 330ml Can',
  unit: 'case',
  is_lot_tracked: true,
  is_active: true,
  created_at: '2026-08-19T00:00:00Z',
  updated_at: '2026-08-19T00:00:00Z',
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('useCatalogStore categories', () => {
  it('fetchCategories populates state from the API', async () => {
    vi.mocked(catalogApi.listCategories).mockResolvedValue({
      items: [sampleCategory],
      page: { next_cursor: null, has_more: false },
    })
    const store = useCatalogStore()
    await store.fetchCategories()
    expect(store.categories).toEqual([sampleCategory])
  })

  it('categoryName resolves a known id and falls back for unknown/null', async () => {
    vi.mocked(catalogApi.listCategories).mockResolvedValue({
      items: [sampleCategory],
      page: { next_cursor: null, has_more: false },
    })
    const store = useCatalogStore()
    await store.fetchCategories()
    expect(store.categoryName('cat-1')).toBe('Soft Drinks')
    expect(store.categoryName('missing')).toBe('—')
    expect(store.categoryName(null)).toBe('—')
  })

  it('createCategory prepends the new category to state', async () => {
    vi.mocked(catalogApi.createCategory).mockResolvedValue(sampleCategory)
    const store = useCatalogStore()
    const result = await store.createCategory({ name: 'Soft Drinks' })
    expect(result).toEqual(sampleCategory)
    expect(store.categories[0]).toEqual(sampleCategory)
  })

  it('updateCategory replaces the matching category in place', async () => {
    const updated = { ...sampleCategory, name: 'Renamed' }
    vi.mocked(catalogApi.listCategories).mockResolvedValue({
      items: [sampleCategory],
      page: { next_cursor: null, has_more: false },
    })
    vi.mocked(catalogApi.updateCategory).mockResolvedValue(updated)
    const store = useCatalogStore()
    await store.fetchCategories()
    await store.updateCategory('cat-1', { name: 'Renamed' })
    expect(store.categories[0]).toEqual(updated)
  })

  it('deactivateCategory marks the matching category inactive in place', async () => {
    vi.mocked(catalogApi.listCategories).mockResolvedValue({
      items: [sampleCategory],
      page: { next_cursor: null, has_more: false },
    })
    vi.mocked(catalogApi.deactivateCategory).mockResolvedValue(undefined)
    const store = useCatalogStore()
    await store.fetchCategories()
    await store.deactivateCategory('cat-1')
    expect(store.categories[0].is_active).toBe(false)
  })
})

describe('useCatalogStore products', () => {
  it('fetchProducts populates state from the API', async () => {
    vi.mocked(catalogApi.listProducts).mockResolvedValue({
      items: [sampleProduct],
      page: { next_cursor: null, has_more: false },
    })
    const store = useCatalogStore()
    await store.fetchProducts()
    expect(store.products).toEqual([sampleProduct])
    expect(store.loading).toBe(false)
    expect(store.error).toBeNull()
  })

  it('fetchProducts records a message on failure', async () => {
    vi.mocked(catalogApi.listProducts).mockRejectedValue(new Error('boom'))
    const store = useCatalogStore()
    await store.fetchProducts()
    expect(store.error).toBe('boom')
  })

  it('loadMoreProducts appends using the stored cursor', async () => {
    vi.mocked(catalogApi.listProducts).mockResolvedValueOnce({
      items: [sampleProduct],
      page: { next_cursor: 'cursor-1', has_more: true },
    })
    const store = useCatalogStore()
    await store.fetchProducts()

    const secondProduct = { ...sampleProduct, id: 'prod-2', sku: 'SPRITE-500' }
    vi.mocked(catalogApi.listProducts).mockResolvedValueOnce({
      items: [secondProduct],
      page: { next_cursor: null, has_more: false },
    })
    await store.loadMoreProducts()

    expect(store.products).toEqual([sampleProduct, secondProduct])
    expect(store.hasMore).toBe(false)
  })

  it('createProduct prepends the new product to state', async () => {
    vi.mocked(catalogApi.createProduct).mockResolvedValue(sampleProduct)
    const store = useCatalogStore()
    const result = await store.createProduct({ sku: 'COKE-330', name: 'Coca-Cola 330ml Can', unit: 'case' })
    expect(result).toEqual(sampleProduct)
    expect(store.products[0]).toEqual(sampleProduct)
  })

  it('deactivateProduct marks the matching product inactive in place', async () => {
    vi.mocked(catalogApi.listProducts).mockResolvedValue({
      items: [sampleProduct],
      page: { next_cursor: null, has_more: false },
    })
    vi.mocked(catalogApi.deactivateProduct).mockResolvedValue(undefined)
    const store = useCatalogStore()
    await store.fetchProducts()
    await store.deactivateProduct('prod-1')
    expect(store.products[0].is_active).toBe(false)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/stores/catalog.test.ts`
Expected: FAIL — `Failed to resolve import "@/stores/catalog"`.

- [ ] **Step 3: Write the implementation**

```ts
import { defineStore } from 'pinia'
import { ref } from 'vue'

import * as catalogApi from '@/api/catalog'
import type { Category, CategoryInput, Product, ProductInput } from '@/api/catalog'

function messageOf(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback
}

export const useCatalogStore = defineStore('catalog', () => {
  const categories = ref<Category[]>([])
  const categoriesLoading = ref(false)
  const categoriesError = ref<string | null>(null)

  async function fetchCategories() {
    categoriesLoading.value = true
    categoriesError.value = null
    try {
      const page = await catalogApi.listCategories({ limit: 100 })
      categories.value = page.items
    } catch (err) {
      categoriesError.value = messageOf(err, 'Failed to load categories.')
    } finally {
      categoriesLoading.value = false
    }
  }

  function categoryName(id: string | null): string {
    if (!id) {
      return '—'
    }
    return categories.value.find((category) => category.id === id)?.name ?? '—'
  }

  async function createCategory(input: CategoryInput) {
    const category = await catalogApi.createCategory(input)
    categories.value = [category, ...categories.value]
    return category
  }

  async function updateCategory(id: string, input: CategoryInput) {
    const category = await catalogApi.updateCategory(id, input)
    categories.value = categories.value.map((item) => (item.id === id ? category : item))
    return category
  }

  async function deactivateCategory(id: string) {
    await catalogApi.deactivateCategory(id)
    categories.value = categories.value.map((item) =>
      item.id === id ? { ...item, is_active: false } : item,
    )
  }

  const products = ref<Product[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const hasMore = ref(false)
  const nextCursor = ref<string | null>(null)

  async function fetchProducts(filter: { search?: string; category_id?: string; is_active?: boolean } = {}) {
    loading.value = true
    error.value = null
    try {
      const page = await catalogApi.listProducts(filter)
      products.value = page.items
      hasMore.value = page.page.has_more
      nextCursor.value = page.page.next_cursor
    } catch (err) {
      error.value = messageOf(err, 'Failed to load products.')
    } finally {
      loading.value = false
    }
  }

  async function loadMoreProducts(filter: { search?: string; category_id?: string; is_active?: boolean } = {}) {
    if (!hasMore.value || !nextCursor.value) {
      return
    }
    loading.value = true
    error.value = null
    try {
      const page = await catalogApi.listProducts({ ...filter, after: nextCursor.value })
      products.value = [...products.value, ...page.items]
      hasMore.value = page.page.has_more
      nextCursor.value = page.page.next_cursor
    } catch (err) {
      error.value = messageOf(err, 'Failed to load products.')
    } finally {
      loading.value = false
    }
  }

  async function createProduct(input: ProductInput) {
    const product = await catalogApi.createProduct(input)
    products.value = [product, ...products.value]
    return product
  }

  async function updateProduct(id: string, input: ProductInput) {
    const product = await catalogApi.updateProduct(id, input)
    products.value = products.value.map((item) => (item.id === id ? product : item))
    return product
  }

  async function deactivateProduct(id: string) {
    await catalogApi.deactivateProduct(id)
    products.value = products.value.map((item) =>
      item.id === id ? { ...item, is_active: false } : item,
    )
  }

  return {
    categories,
    categoriesLoading,
    categoriesError,
    fetchCategories,
    categoryName,
    createCategory,
    updateCategory,
    deactivateCategory,
    products,
    loading,
    error,
    hasMore,
    fetchProducts,
    loadMoreProducts,
    createProduct,
    updateProduct,
    deactivateProduct,
  }
})
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npx vitest run src/stores/catalog.test.ts`
Expected: PASS (11 tests).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/stores/catalog.ts frontend/src/stores/catalog.test.ts
git commit -m "feat: add catalog Pinia store"
```

---

## Task 4: `components/ProductFormDialog.vue`

**Files:**
- Create: `frontend/src/components/ProductFormDialog.vue`
- Create: `frontend/src/components/ProductFormDialog.spec.ts`

- [ ] **Step 1: Write the failing test**

```ts
import Aura from '@primeuix/themes/aura'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import PrimeVue from 'primevue/config'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiClientError } from '@/api/client'
import * as catalogApi from '@/api/catalog'
import ProductFormDialog from '@/components/ProductFormDialog.vue'

vi.mock('@/api/catalog')

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
  vi.mocked(catalogApi.listCategories).mockResolvedValue({
    items: [],
    page: { next_cursor: null, has_more: false },
  })
})

function mountDialog(product: catalogApi.Product | null = null) {
  return mount(ProductFormDialog, {
    props: { visible: true, product },
    global: {
      plugins: [[PrimeVue, { theme: { preset: Aura } }]],
      stubs: { Portal: { template: '<div><slot /></div>' } },
    },
  })
}

async function flushPromises() {
  await new Promise((resolve) => setTimeout(resolve, 0))
}

describe('ProductFormDialog', () => {
  it('blocks submit and shows errors when required fields are blank', async () => {
    const wrapper = mountDialog()
    await wrapper.find('[data-testid="submit"]').trigger('click')
    expect(wrapper.text()).toContain('SKU is required.')
    expect(wrapper.text()).toContain('Name is required.')
    expect(wrapper.text()).toContain('Unit is required.')
    expect(catalogApi.createProduct).not.toHaveBeenCalled()
  })

  it('creates a product and emits saved on valid submit', async () => {
    vi.mocked(catalogApi.createProduct).mockResolvedValue({
      id: 'prod-1',
      category_id: null,
      sku: 'COKE-330',
      barcode: null,
      name: 'Coca-Cola 330ml Can',
      unit: 'case',
      is_lot_tracked: true,
      is_active: true,
      created_at: '',
      updated_at: '',
    })
    const wrapper = mountDialog()
    await wrapper.find('#product-sku').setValue('COKE-330')
    await wrapper.find('#product-name').setValue('Coca-Cola 330ml Can')
    await wrapper.find('#product-unit').setValue('case')
    await wrapper.find('[data-testid="submit"]').trigger('click')
    await flushPromises()
    expect(catalogApi.createProduct).toHaveBeenCalledWith({
      sku: 'COKE-330',
      barcode: null,
      name: 'Coca-Cola 330ml Can',
      unit: 'case',
      category_id: null,
      is_lot_tracked: true,
      is_active: true,
    })
    expect(wrapper.emitted('saved')).toBeTruthy()
  })

  it('shows a SKU-conflict error on the SKU field', async () => {
    vi.mocked(catalogApi.createProduct).mockRejectedValue(
      new ApiClientError(409, 'PRODUCT_SKU_CONFLICT', 'SKU already exists'),
    )
    const wrapper = mountDialog()
    await wrapper.find('#product-sku').setValue('COKE-330')
    await wrapper.find('#product-name').setValue('Coca-Cola 330ml Can')
    await wrapper.find('#product-unit').setValue('case')
    await wrapper.find('[data-testid="submit"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('This SKU is already in use.')
  })

  it('shows a valid-checksum hint for a well-formed EAN-13 barcode', async () => {
    const wrapper = mountDialog()
    await wrapper.find('#product-barcode').setValue('4006381333931')
    expect(wrapper.find('[data-testid="barcode-hint"]').text()).toContain('Valid EAN-13')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/components/ProductFormDialog.spec.ts`
Expected: FAIL — the component file does not exist yet.

- [ ] **Step 3: Write the implementation**

```vue
<script setup lang="ts">
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'
import Select from 'primevue/select'
import ToggleSwitch from 'primevue/toggleswitch'
import { computed, onMounted, reactive, ref, watch } from 'vue'

import { ApiClientError } from '@/api/client'
import type { Product, ProductInput } from '@/api/catalog'
import { validateBarcode } from '@/lib/barcode'
import { useCatalogStore } from '@/stores/catalog'

const props = defineProps<{
  visible: boolean
  product: Product | null
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  saved: []
}>()

const store = useCatalogStore()
const isEdit = computed(() => props.product !== null)

const form = reactive({
  sku: '',
  barcode: '',
  name: '',
  unit: '',
  category_id: null as string | null,
  is_lot_tracked: true,
  is_active: true,
})

const skuError = ref<string | null>(null)
const nameError = ref<string | null>(null)
const unitError = ref<string | null>(null)
const barcodeError = ref<string | null>(null)
const generalError = ref<string | null>(null)
const submitting = ref(false)

const categoryOptions = computed(() => [
  { label: 'None', value: null },
  ...store.categories.map((category) => ({ label: category.name, value: category.id })),
])

const barcodeHint = computed(() => {
  if (form.barcode.trim() === '') {
    return null
  }
  const result = validateBarcode(form.barcode)
  return result.valid ? `Valid ${result.format}` : result.message
})

onMounted(() => {
  if (store.categories.length === 0) {
    store.fetchCategories()
  }
})

watch(
  () => props.visible,
  (visible) => {
    if (!visible) {
      return
    }
    skuError.value = null
    nameError.value = null
    unitError.value = null
    barcodeError.value = null
    generalError.value = null
    form.sku = props.product?.sku ?? ''
    form.barcode = props.product?.barcode ?? ''
    form.name = props.product?.name ?? ''
    form.unit = props.product?.unit ?? ''
    form.category_id = props.product?.category_id ?? null
    form.is_lot_tracked = props.product?.is_lot_tracked ?? true
    form.is_active = props.product?.is_active ?? true
  },
  { immediate: true },
)

function validate(): boolean {
  skuError.value = form.sku.trim() === '' ? 'SKU is required.' : null
  nameError.value = form.name.trim() === '' ? 'Name is required.' : null
  unitError.value = form.unit.trim() === '' ? 'Unit is required.' : null
  return skuError.value === null && nameError.value === null && unitError.value === null
}

async function submit() {
  generalError.value = null
  barcodeError.value = null
  if (!validate()) {
    return
  }
  submitting.value = true
  try {
    const input: ProductInput = {
      sku: form.sku,
      barcode: form.barcode || null,
      name: form.name,
      unit: form.unit,
      category_id: form.category_id,
      is_lot_tracked: form.is_lot_tracked,
      is_active: form.is_active,
    }
    if (isEdit.value && props.product) {
      await store.updateProduct(props.product.id, input)
    } else {
      await store.createProduct(input)
    }
    emit('saved')
    emit('update:visible', false)
  } catch (err) {
    if (err instanceof ApiClientError && err.code === 'PRODUCT_SKU_CONFLICT') {
      skuError.value = 'This SKU is already in use.'
    } else if (err instanceof ApiClientError && err.code === 'PRODUCT_BARCODE_CONFLICT') {
      barcodeError.value = 'This barcode is already in use.'
    } else if (err instanceof ApiClientError && err.code === 'INVALID_BARCODE') {
      barcodeError.value = 'This barcode is not a valid UPC-A or EAN-13 value.'
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
    :header="isEdit ? 'Edit product' : 'Add product'"
    @update:visible="(value: boolean) => emit('update:visible', value)"
  >
    <div class="mb-4 flex flex-col gap-1">
      <label for="product-sku">SKU</label>
      <InputText id="product-sku" v-model="form.sku" :invalid="skuError !== null" />
      <Message v-if="skuError" severity="error" size="small" variant="simple">{{ skuError }}</Message>
    </div>
    <div class="mb-4 flex flex-col gap-1">
      <label for="product-name">Name</label>
      <InputText id="product-name" v-model="form.name" :invalid="nameError !== null" />
      <Message v-if="nameError" severity="error" size="small" variant="simple">{{ nameError }}</Message>
    </div>
    <div class="mb-4 flex flex-col gap-1">
      <label for="product-unit">Unit</label>
      <InputText id="product-unit" v-model="form.unit" :invalid="unitError !== null" />
      <Message v-if="unitError" severity="error" size="small" variant="simple">{{ unitError }}</Message>
    </div>
    <div class="mb-4 flex flex-col gap-1">
      <label for="product-barcode">Barcode</label>
      <InputText id="product-barcode" v-model="form.barcode" :invalid="barcodeError !== null" />
      <Message v-if="barcodeError" severity="error" size="small" variant="simple">{{ barcodeError }}</Message>
      <small v-else-if="barcodeHint" data-testid="barcode-hint">{{ barcodeHint }}</small>
    </div>
    <div class="mb-4 flex flex-col gap-1">
      <label for="product-category">Category</label>
      <Select
        id="product-category"
        v-model="form.category_id"
        :options="categoryOptions"
        option-label="label"
        option-value="value"
      />
    </div>
    <div class="mb-4 flex flex-row items-center gap-3">
      <label for="product-lot-tracked">Lot-tracked</label>
      <ToggleSwitch id="product-lot-tracked" v-model="form.is_lot_tracked" />
    </div>
    <div class="mb-4 flex flex-row items-center gap-3">
      <label for="product-active">Active</label>
      <ToggleSwitch id="product-active" v-model="form.is_active" />
    </div>
    <Message v-if="generalError" severity="error" size="small">{{ generalError }}</Message>
    <template #footer>
      <Button label="Cancel" severity="secondary" data-testid="cancel" @click="emit('update:visible', false)" />
      <Button label="Save" :loading="submitting" data-testid="submit" @click="submit" />
    </template>
  </Dialog>
</template>
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npx vitest run src/components/ProductFormDialog.spec.ts`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/ProductFormDialog.vue frontend/src/components/ProductFormDialog.spec.ts
git commit -m "feat: add ProductFormDialog component"
```

---

## Task 5: `components/CategoryFormDialog.vue`

**Files:**
- Create: `frontend/src/components/CategoryFormDialog.vue`
- Create: `frontend/src/components/CategoryFormDialog.spec.ts`

- [ ] **Step 1: Write the failing test**

```ts
import Aura from '@primeuix/themes/aura'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import PrimeVue from 'primevue/config'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiClientError } from '@/api/client'
import * as catalogApi from '@/api/catalog'
import CategoryFormDialog from '@/components/CategoryFormDialog.vue'

vi.mock('@/api/catalog')

const sampleCategory: catalogApi.Category = {
  id: 'cat-1',
  parent_id: null,
  name: 'Soft Drinks',
  is_active: true,
  created_at: '',
  updated_at: '',
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
  vi.mocked(catalogApi.listCategories).mockResolvedValue({
    items: [sampleCategory],
    page: { next_cursor: null, has_more: false },
  })
})

function mountDialog() {
  return mount(CategoryFormDialog, {
    props: { visible: true },
    global: {
      plugins: [[PrimeVue, { theme: { preset: Aura } }]],
      stubs: { Portal: { template: '<div><slot /></div>' } },
    },
  })
}

async function flushPromises() {
  await new Promise((resolve) => setTimeout(resolve, 0))
}

describe('CategoryFormDialog', () => {
  it('lists existing categories', async () => {
    const wrapper = mountDialog()
    await flushPromises()
    expect(wrapper.text()).toContain('Soft Drinks')
  })

  it('blocks submit when name is blank', async () => {
    const wrapper = mountDialog()
    await flushPromises()
    await wrapper.find('[data-testid="submit"]').trigger('click')
    expect(wrapper.text()).toContain('Name is required.')
    expect(catalogApi.createCategory).not.toHaveBeenCalled()
  })

  it('creates a category on valid submit', async () => {
    vi.mocked(catalogApi.createCategory).mockResolvedValue({
      id: 'cat-2',
      parent_id: null,
      name: 'Snacks',
      is_active: true,
      created_at: '',
      updated_at: '',
    })
    const wrapper = mountDialog()
    await flushPromises()
    await wrapper.find('#category-name').setValue('Snacks')
    await wrapper.find('[data-testid="submit"]').trigger('click')
    await flushPromises()
    expect(catalogApi.createCategory).toHaveBeenCalledWith({ name: 'Snacks', parent_id: null, is_active: true })
  })

  it('shows a name-conflict error on the name field', async () => {
    vi.mocked(catalogApi.createCategory).mockRejectedValue(
      new ApiClientError(409, 'CATEGORY_NAME_CONFLICT', 'Category name already exists'),
    )
    const wrapper = mountDialog()
    await flushPromises()
    await wrapper.find('#category-name').setValue('Soft Drinks')
    await wrapper.find('[data-testid="submit"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('This name is already in use.')
  })

  it('starts editing a category when its edit action is clicked', async () => {
    const wrapper = mountDialog()
    await flushPromises()
    await wrapper.find('[data-testid="edit-category-cat-1"]').trigger('click')
    expect((wrapper.find('#category-name').element as HTMLInputElement).value).toBe('Soft Drinks')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/components/CategoryFormDialog.spec.ts`
Expected: FAIL — the component file does not exist yet.

- [ ] **Step 3: Write the implementation**

```vue
<script setup lang="ts">
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'
import Select from 'primevue/select'
import ToggleSwitch from 'primevue/toggleswitch'
import { computed, onMounted, reactive, ref, watch } from 'vue'

import { ApiClientError } from '@/api/client'
import type { Category, CategoryInput } from '@/api/catalog'
import { useCatalogStore } from '@/stores/catalog'

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
}>()

const store = useCatalogStore()
const editingId = ref<string | null>(null)
const isEdit = computed(() => editingId.value !== null)

const form = reactive({
  name: '',
  parent_id: null as string | null,
  is_active: true,
})

const nameError = ref<string | null>(null)
const parentError = ref<string | null>(null)
const generalError = ref<string | null>(null)
const submitting = ref(false)

const parentOptions = computed(() => [
  { label: 'None', value: null },
  ...store.categories
    .filter((category) => category.id !== editingId.value)
    .map((category) => ({ label: category.name, value: category.id })),
])

function resetForm() {
  editingId.value = null
  form.name = ''
  form.parent_id = null
  form.is_active = true
  nameError.value = null
  parentError.value = null
  generalError.value = null
}

function startEdit(category: Category) {
  editingId.value = category.id
  form.name = category.name
  form.parent_id = category.parent_id
  form.is_active = category.is_active
  nameError.value = null
  parentError.value = null
  generalError.value = null
}

onMounted(() => {
  store.fetchCategories()
})

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      resetForm()
      store.fetchCategories()
    }
  },
)

function validate(): boolean {
  nameError.value = form.name.trim() === '' ? 'Name is required.' : null
  return nameError.value === null
}

async function submit() {
  generalError.value = null
  parentError.value = null
  if (!validate()) {
    return
  }
  submitting.value = true
  try {
    const input: CategoryInput = {
      name: form.name,
      parent_id: form.parent_id,
      is_active: form.is_active,
    }
    if (isEdit.value && editingId.value) {
      await store.updateCategory(editingId.value, input)
    } else {
      await store.createCategory(input)
    }
    resetForm()
  } catch (err) {
    if (err instanceof ApiClientError && err.code === 'CATEGORY_NAME_CONFLICT') {
      nameError.value = 'This name is already in use.'
    } else if (err instanceof ApiClientError && err.code === 'CATEGORY_CYCLE') {
      parentError.value = 'This parent would create a category cycle.'
    } else if (err instanceof ApiClientError) {
      generalError.value = err.message
    } else {
      generalError.value = 'Something went wrong. Try again.'
    }
  } finally {
    submitting.value = false
  }
}

async function deactivate(category: Category) {
  generalError.value = null
  try {
    await store.deactivateCategory(category.id)
  } catch (err) {
    if (err instanceof ApiClientError && err.code === 'CATEGORY_IN_USE') {
      generalError.value = 'This category has an active child category or product and cannot be deactivated.'
    } else if (err instanceof ApiClientError) {
      generalError.value = err.message
    } else {
      generalError.value = 'Something went wrong. Try again.'
    }
  }
}
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    header="Manage categories"
    @update:visible="(value: boolean) => emit('update:visible', value)"
  >
    <div class="mb-4 flex flex-col gap-2">
      <div
        v-for="category in store.categories"
        :key="category.id"
        class="flex items-center justify-between rounded-badge border border-border px-3 py-2"
      >
        <span>
          {{ category.name }}
          <span
            class="ml-2 rounded-badge px-2 py-1 text-xs font-medium"
            :class="category.is_active ? 'bg-success-bg text-success-text' : 'bg-danger-bg text-danger-text'"
          >{{ category.is_active ? 'Active' : 'Inactive' }}</span>
        </span>
        <span class="flex gap-1">
          <Button
            icon="pi pi-pencil"
            size="small"
            severity="secondary"
            text
            rounded
            aria-label="Edit category"
            :data-testid="`edit-category-${category.id}`"
            @click="startEdit(category)"
          />
          <Button
            v-if="category.is_active"
            icon="pi pi-ban"
            size="small"
            severity="danger"
            text
            rounded
            aria-label="Deactivate category"
            :data-testid="`deactivate-category-${category.id}`"
            @click="deactivate(category)"
          />
        </span>
      </div>
      <p v-if="store.categories.length === 0" class="text-ink-faint">No categories yet.</p>
    </div>

    <div class="mb-4 flex flex-col gap-1">
      <label for="category-name">{{ isEdit ? 'Edit name' : 'New category name' }}</label>
      <InputText id="category-name" v-model="form.name" :invalid="nameError !== null" />
      <Message v-if="nameError" severity="error" size="small" variant="simple">{{ nameError }}</Message>
    </div>
    <div class="mb-4 flex flex-col gap-1">
      <label for="category-parent">Parent category</label>
      <Select
        id="category-parent"
        v-model="form.parent_id"
        :options="parentOptions"
        option-label="label"
        option-value="value"
      />
      <Message v-if="parentError" severity="error" size="small" variant="simple">{{ parentError }}</Message>
    </div>
    <div class="mb-4 flex flex-row items-center gap-3">
      <label for="category-active">Active</label>
      <ToggleSwitch id="category-active" v-model="form.is_active" />
    </div>
    <Message v-if="generalError" severity="error" size="small">{{ generalError }}</Message>
    <template #footer>
      <Button v-if="isEdit" label="Cancel edit" severity="secondary" data-testid="cancel-edit" @click="resetForm" />
      <Button label="Close" severity="secondary" data-testid="cancel" @click="emit('update:visible', false)" />
      <Button :label="isEdit ? 'Save changes' : 'Add category'" :loading="submitting" data-testid="submit" @click="submit" />
    </template>
  </Dialog>
</template>
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npx vitest run src/components/CategoryFormDialog.spec.ts`
Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/CategoryFormDialog.vue frontend/src/components/CategoryFormDialog.spec.ts
git commit -m "feat: add CategoryFormDialog component"
```

---

## Task 6: `views/ProductsView.vue`

**Files:**
- Create: `frontend/src/views/ProductsView.vue`
- Create: `frontend/src/views/ProductsView.spec.ts`

- [ ] **Step 1: Write the failing test**

```ts
import Aura from '@primeuix/themes/aura'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import PrimeVue from 'primevue/config'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import * as catalogApi from '@/api/catalog'
import { useAuthStore } from '@/stores/auth'
import ProductsView from '@/views/ProductsView.vue'

vi.mock('@/api/catalog')

const sampleCategory: catalogApi.Category = {
  id: 'cat-1',
  parent_id: null,
  name: 'Soft Drinks',
  is_active: true,
  created_at: '',
  updated_at: '',
}

const sampleProduct: catalogApi.Product = {
  id: 'prod-1',
  category_id: 'cat-1',
  sku: 'COKE-330',
  barcode: '4006381333931',
  name: 'Coca-Cola 330ml Can',
  unit: 'case',
  is_lot_tracked: true,
  is_active: true,
  created_at: '',
  updated_at: '',
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
  vi.mocked(catalogApi.listCategories).mockResolvedValue({
    items: [sampleCategory],
    page: { next_cursor: null, has_more: false },
  })
  vi.mocked(catalogApi.listProducts).mockResolvedValue({
    items: [sampleProduct],
    page: { next_cursor: null, has_more: false },
  })
})

function mountView() {
  return mount(ProductsView, {
    global: {
      plugins: [[PrimeVue, { theme: { preset: Aura } }]],
      stubs: { Portal: { template: '<div><slot /></div>' } },
    },
  })
}

async function flushPromises() {
  await new Promise((resolve) => setTimeout(resolve, 0))
}

describe('ProductsView', () => {
  it('renders products with the resolved category name', async () => {
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('COKE-330')
    expect(wrapper.text()).toContain('Coca-Cola 330ml Can')
    expect(wrapper.text()).toContain('Soft Drinks')
  })

  it('hides Add Product and Manage Categories for a picker', async () => {
    const auth = useAuthStore()
    auth.user = { id: 'u1', email: 'p@bwims.test', full_name: 'Picker', role: 'picker' }
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.find('[data-testid="add-product"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="manage-categories"]').exists()).toBe(false)
  })

  it('shows Add Product and Manage Categories for a warehouse manager', async () => {
    const auth = useAuthStore()
    auth.user = { id: 'u1', email: 'm@bwims.test', full_name: 'Manager', role: 'warehouse_manager' }
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.find('[data-testid="add-product"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="manage-categories"]').exists()).toBe(true)
  })

  it('shows an error banner when loading fails', async () => {
    vi.mocked(catalogApi.listProducts).mockRejectedValue(new Error('network down'))
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.find('[data-testid="products-error"]').text()).toBe('network down')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/views/ProductsView.spec.ts`
Expected: FAIL — the view file does not exist yet.

- [ ] **Step 3: Write the implementation**

```vue
<script setup lang="ts">
import Button from 'primevue/button'
import Column from 'primevue/column'
import DataTable from 'primevue/datatable'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import { computed, onMounted, ref, watch } from 'vue'

import type { Product } from '@/api/catalog'
import CategoryFormDialog from '@/components/CategoryFormDialog.vue'
import ProductFormDialog from '@/components/ProductFormDialog.vue'
import { useAuthStore } from '@/stores/auth'
import { useCatalogStore } from '@/stores/catalog'

const store = useCatalogStore()
const auth = useAuthStore()

const search = ref('')
const categoryFilter = ref<string | null>(null)
const statusFilter = ref<'all' | 'active' | 'inactive'>('all')
const productDialogVisible = ref(false)
const categoryDialogVisible = ref(false)
const editingProduct = ref<Product | null>(null)

const canManageCatalog = computed(
  () => auth.user?.role === 'admin' || auth.user?.role === 'warehouse_manager',
)

const categoryFilterOptions = computed(() => [
  { label: 'All categories', value: null },
  ...store.categories.map((category) => ({ label: category.name, value: category.id })),
])

const statusFilterOptions = [
  { label: 'All statuses', value: 'all' },
  { label: 'Active', value: 'active' },
  { label: 'Inactive', value: 'inactive' },
]

function currentFilter() {
  return {
    search: search.value || undefined,
    category_id: categoryFilter.value ?? undefined,
    is_active: statusFilter.value === 'all' ? undefined : statusFilter.value === 'active',
  }
}

function refetch() {
  store.fetchProducts(currentFilter())
}

onMounted(() => {
  store.fetchCategories()
  refetch()
})

let searchTimeout: ReturnType<typeof setTimeout> | undefined
watch(search, () => {
  if (searchTimeout) {
    clearTimeout(searchTimeout)
  }
  searchTimeout = setTimeout(refetch, 300)
})
watch([categoryFilter, statusFilter], refetch)

function openCreateDialog() {
  editingProduct.value = null
  productDialogVisible.value = true
}

function openEditDialog(product: Product) {
  editingProduct.value = product
  productDialogVisible.value = true
}

async function deactivate(product: Product) {
  await store.deactivateProduct(product.id)
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold text-ink">Products & Categories</h1>
        <p class="text-ink-muted">Manage product catalog, identifiers, and categorizations.</p>
      </div>
      <div class="flex gap-2">
        <Button
          v-if="canManageCatalog"
          label="Manage Categories"
          severity="secondary"
          data-testid="manage-categories"
          @click="categoryDialogVisible = true"
        />
        <Button
          v-if="canManageCatalog"
          label="Add Product"
          data-testid="add-product"
          @click="openCreateDialog"
        />
      </div>
    </div>

    <div class="flex flex-wrap gap-3">
      <InputText
        v-model="search"
        placeholder="Search SKU, Name, or Barcode..."
        data-testid="product-search"
        class="max-w-[320px] flex-1"
      />
      <Select
        v-model="categoryFilter"
        :options="categoryFilterOptions"
        option-label="label"
        option-value="value"
        class="w-[200px]"
      />
      <Select
        v-model="statusFilter"
        :options="statusFilterOptions"
        option-label="label"
        option-value="value"
        class="w-[160px]"
      />
    </div>

    <p v-if="store.error" class="text-danger-text" data-testid="products-error">{{ store.error }}</p>

    <DataTable
      v-else
      :value="store.products"
      :loading="store.loading"
      data-key="id"
      class="overflow-hidden rounded-[12px] border border-border"
    >
      <template #empty>
        <p>No products yet.</p>
      </template>
      <Column field="sku" header="SKU">
        <template #body="{ data }">
          <span class="font-mono-code text-[0.85rem]">{{ data.sku }}</span>
        </template>
      </Column>
      <Column field="barcode" header="Barcode">
        <template #body="{ data }">
          <span v-if="data.barcode" class="font-mono-code text-[0.85rem]">{{ data.barcode }}</span>
          <span v-else class="text-ink-faint">—</span>
        </template>
      </Column>
      <Column field="name" header="Product Name" />
      <Column header="Category">
        <template #body="{ data }">{{ store.categoryName(data.category_id) }}</template>
      </Column>
      <Column field="unit" header="Unit" />
      <Column header="Lot-Tracked">
        <template #body="{ data }">
          <i v-if="data.is_lot_tracked" class="pi pi-check text-success-text" aria-label="Lot-tracked" />
          <span v-else class="text-ink-faint">—</span>
        </template>
      </Column>
      <Column header="Status">
        <template #body="{ data }">
          <span
            class="rounded-badge px-2 py-1 text-xs font-medium"
            :class="data.is_active ? 'bg-success-bg text-success-text' : 'bg-danger-bg text-danger-text'"
          >{{ data.is_active ? 'Active' : 'Inactive' }}</span>
        </template>
      </Column>
      <Column v-if="canManageCatalog" header="Actions">
        <template #body="{ data }">
          <Button
            icon="pi pi-pencil"
            size="small"
            severity="secondary"
            text
            rounded
            aria-label="Edit product"
            data-testid="edit-product"
            @click="openEditDialog(data)"
          />
          <Button
            v-if="data.is_active"
            icon="pi pi-ban"
            size="small"
            severity="danger"
            text
            rounded
            aria-label="Deactivate product"
            data-testid="deactivate-product"
            @click="deactivate(data)"
          />
        </template>
      </Column>
    </DataTable>

    <Button
      v-if="store.hasMore"
      label="Load more"
      severity="secondary"
      data-testid="load-more"
      @click="store.loadMoreProducts(currentFilter())"
    />

    <ProductFormDialog v-model:visible="productDialogVisible" :product="editingProduct" />
    <CategoryFormDialog v-model:visible="categoryDialogVisible" />
  </div>
</template>
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npx vitest run src/views/ProductsView.spec.ts`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/ProductsView.vue frontend/src/views/ProductsView.spec.ts
git commit -m "feat: add ProductsView"
```

---

## Task 7: Wire up routing and navigation

**Files:**
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/layouts/AppLayout.vue`

- [ ] **Step 1: Add the route**

In `frontend/src/router/index.ts`, inside the `AppLayout` route's `children` array, add (alongside the existing `warehouses`/`warehouse-detail` entries):

```ts
        {
          path: 'products',
          name: 'products',
          component: () => import('@/views/ProductsView.vue'),
        },
```

- [ ] **Step 2: Add the nav link**

In `frontend/src/layouts/AppLayout.vue`, replace:

```html
        <RouterLink
          to="/warehouses"
          class="rounded-[10px] px-[0.9rem] py-[0.8rem] [&.router-link-active]:bg-brand-amber [&.router-link-active]:font-bold [&.router-link-active]:text-brand-ink"
        >Warehouses</RouterLink>
        <RouterLink
          to="/barcode-test"
          class="rounded-[10px] px-[0.9rem] py-[0.8rem] [&.router-link-active]:bg-brand-amber [&.router-link-active]:font-bold [&.router-link-active]:text-brand-ink"
        >Barcode test</RouterLink>
        <span class="cursor-not-allowed rounded-[10px] px-[0.9rem] py-[0.8rem] text-[#9fb1c6]">Inventory</span>
        <span class="cursor-not-allowed rounded-[10px] px-[0.9rem] py-[0.8rem] text-[#9fb1c6]">Movements</span>
```

with:

```html
        <RouterLink
          to="/warehouses"
          class="rounded-[10px] px-[0.9rem] py-[0.8rem] [&.router-link-active]:bg-brand-amber [&.router-link-active]:font-bold [&.router-link-active]:text-brand-ink"
        >Warehouses</RouterLink>
        <RouterLink
          to="/products"
          class="rounded-[10px] px-[0.9rem] py-[0.8rem] [&.router-link-active]:bg-brand-amber [&.router-link-active]:font-bold [&.router-link-active]:text-brand-ink"
        >Products</RouterLink>
        <RouterLink
          to="/barcode-test"
          class="rounded-[10px] px-[0.9rem] py-[0.8rem] [&.router-link-active]:bg-brand-amber [&.router-link-active]:font-bold [&.router-link-active]:text-brand-ink"
        >Barcode test</RouterLink>
        <span class="cursor-not-allowed rounded-[10px] px-[0.9rem] py-[0.8rem] text-[#9fb1c6]">Inventory</span>
        <span class="cursor-not-allowed rounded-[10px] px-[0.9rem] py-[0.8rem] text-[#9fb1c6]">Movements</span>
```

- [ ] **Step 3: Manually verify in the browser**

With the backend running (`docker ps` should already show `api`/`postgres` up), run `npm run dev` and log in as the seeded admin. Click "Products", confirm the list loads, add a product, add a category via "Manage Categories", confirm it appears in the product form's Category dropdown.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/router/index.ts frontend/src/layouts/AppLayout.vue
git commit -m "feat: wire products route and navigation into the app shell"
```

---

## Task 8: Final validation

**Files:** none (verification only)

- [ ] **Step 1: Run the full frontend test suite**

Run: `cd frontend && npm test`
Expected: PASS — all prior tests plus this slice's new tests (12 + 11 + 4 + 5 + 4 = 36 new tests, on top of the existing 42).

- [ ] **Step 2: Typecheck**

Run: `cd frontend && npm run typecheck`
Expected: PASS.

- [ ] **Step 3: Build**

Run: `cd frontend && npm run build`
Expected: PASS.

- [ ] **Step 4: No commit for this task** — verification checkpoint only.

---

## Self-Review Notes

**Spec coverage check:** table columns match the design doc's list exactly ✓, search/category/status filters ✓, Add Product + icon-only Edit/Deactivate actions gated by `canManageCatalog` ✓, category name resolution via store lookup (not denormalized data) ✓, "Manage Categories" dialog with list + inline form ✓, barcode hint reuses `lib/barcode.ts` ✓, conflict-error routing per field (SKU/barcode/name/parent-cycle/in-use) ✓, "Load more" instead of numbered pagination ✓, no bulk-select/export/draft-status built ✓.

**Type consistency check:** `Product`/`Category`/`ProductInput`/`CategoryInput` defined once in `api/catalog.ts` (Tasks 1–2), reused identically in the store (Task 3), both dialogs (Tasks 4–5), and the view (Task 6). Store action names (`fetchProducts`/`loadMoreProducts`/`createProduct`/`updateProduct`/`deactivateProduct`, `fetchCategories`/`categoryName`/`createCategory`/`updateCategory`/`deactivateCategory`) match exactly between the store's own tests and every consumer.

**Placeholder scan:** none — every step has complete, copy-pasteable code.
