import { defineStore } from 'pinia'
import { ref } from 'vue'

import {
  type Category,
  type CategoryInput,
  type Product,
  type ProductInput,
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
import { apiRequest } from '@/api/client'

export const useCatalogStore = defineStore('catalog', () => {
  const products = ref<Product[]>([])
  const categories = ref<Category[]>([])
  const currentProduct = ref<Product | null>(null)
  const currentCategory = ref<Category | null>(null)
  const barcodeLookupResult = ref<Product | null>(null)

  const loading = ref(false)
  const error = ref<string | null>(null)
  const nextCursor = ref<string | null>(null)
  const hasMore = ref(false)

  // Returns the next cursor so a caller can page; null ends the list.
  async function fetchProducts(
    search?: string,
    categoryId?: string,
    is_active?: boolean,
    after?: string,
  ): Promise<string | null> {
    loading.value = true
    error.value = null
    try {
      const page = await listProducts({ search, category_id: categoryId, is_active, after })
      products.value = page.items
      nextCursor.value = page.page.next_cursor
      hasMore.value = page.page.has_more
      return page.page.has_more ? page.page.next_cursor : null
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch products'
      return null
    } finally {
      loading.value = false
    }
  }

  async function fetchCategories(search?: string, is_active?: boolean) {
    loading.value = true
    error.value = null
    try {
      const page = await listCategories({ search, is_active })
      categories.value = page.items
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch categories'
    } finally {
      loading.value = false
    }
  }

  async function lookupBarcode(code: string): Promise<Product | null> {
    loading.value = true
    error.value = null
    barcodeLookupResult.value = null
    try {
      const res = await apiRequest<Product>(`/products/by-barcode/${encodeURIComponent(code)}`)
      barcodeLookupResult.value = res
      return res
    } catch (e: any) {
      error.value = e.message || 'Product not found by barcode'
      return null
    } finally {
      loading.value = false
    }
  }

  async function addProduct(input: ProductInput) {
    loading.value = true
    error.value = null
    try {
      const newProduct = await createProduct(input)
      products.value.unshift(newProduct)
      return newProduct
    } catch (e: any) {
      error.value = e.message || 'Failed to create product'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function editProduct(id: string, input: ProductInput) {
    loading.value = true
    error.value = null
    try {
      const updated = await updateProduct(id, input)
      const idx = products.value.findIndex((p) => p.id === id)
      if (idx !== -1) {
        products.value[idx] = updated
      }
      return updated
    } catch (e: any) {
      error.value = e.message || 'Failed to update product'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function removeProduct(id: string) {
    loading.value = true
    error.value = null
    try {
      await deactivateProduct(id)
      const idx = products.value.findIndex((p) => p.id === id)
      if (idx !== -1) {
        products.value[idx].is_active = false
      }
    } catch (e: any) {
      error.value = e.message || 'Failed to deactivate product'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function addCategory(input: CategoryInput) {
    loading.value = true
    error.value = null
    try {
      const cat = await createCategory(input)
      categories.value.push(cat)
      return cat
    } catch (e: any) {
      error.value = e.message || 'Failed to create category'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function editCategory(id: string, input: CategoryInput) {
    loading.value = true
    error.value = null
    try {
      const cat = await updateCategory(id, input)
      const idx = categories.value.findIndex((c) => c.id === id)
      if (idx !== -1) {
        categories.value[idx] = cat
      }
      return cat
    } catch (e: any) {
      error.value = e.message || 'Failed to update category'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function removeCategory(id: string) {
    loading.value = true
    error.value = null
    try {
      await deactivateCategory(id)
      const idx = categories.value.findIndex((c) => c.id === id)
      if (idx !== -1) {
        categories.value[idx].is_active = false
      }
    } catch (e: any) {
      error.value = e.message || 'Failed to deactivate category'
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    products,
    categories,
    currentProduct,
    currentCategory,
    barcodeLookupResult,
    loading,
    error,
    hasMore,
    fetchProducts,
    fetchCategories,
    lookupBarcode,
    addProduct,
    editProduct,
    removeProduct,
    addCategory,
    editCategory,
    removeCategory,
  }
})
