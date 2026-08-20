import { beforeEach, describe, expect, it, vi } from 'vitest'

import { apiRequest } from '@/api/client'
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
