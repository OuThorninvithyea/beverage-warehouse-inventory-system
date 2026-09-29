import { describe, expect, it, vi } from 'vitest'

import type { Category } from '@/api/catalog'

import { lookupExternalProduct, suggestCategory, suggestProduct, suggestSku } from './product-lookup'

const category = (id: string, name: string, is_active = true) => ({ id, name, is_active }) as Category
const categories = [
  category('csd', 'Carbonated Soft Drinks'),
  category('cola', 'Cola'),
  category('water', 'Water'),
  category('beer', 'Beer'),
  category('old', 'Lager', false),
]

const respond = (body: unknown, ok = true) =>
  vi.fn().mockResolvedValue({ ok, json: () => Promise.resolve(body) }) as unknown as typeof fetch

describe('lookupExternalProduct', () => {
  it('reads name, brand, size and categories', async () => {
    const fetcher = respond({
      status: 1,
      product: {
        product_name: 'Coca-Cola Original',
        product_name_en: 'Coca-Cola',
        brands: 'Coca-Cola, The Coca-Cola Company',
        quantity: '330 ml',
        categories_tags: ['en:sodas', 'en:colas'],
      },
    })
    expect(await lookupExternalProduct('5449000000996', fetcher)).toEqual({
      name: 'Coca-Cola',
      brand: 'Coca-Cola',
      quantity: '330 ml',
      categoryTags: ['en:sodas', 'en:colas'],
    })
  })

  it('returns null for an unknown barcode, an error or no connection', async () => {
    expect(await lookupExternalProduct('1', respond({ status: 0 }))).toBeNull()
    expect(await lookupExternalProduct('1', respond({}, false))).toBeNull()
    const offline = vi.fn().mockRejectedValue(new TypeError('offline')) as unknown as typeof fetch
    expect(await lookupExternalProduct('1', offline)).toBeNull()
  })
})

describe('suggestions', () => {
  it('prefers the most specific category that exists and is active', () => {
    expect(suggestCategory(['en:soft-drinks', 'en:colas'], categories)).toBe('cola')
    expect(suggestCategory(['en:sodas'], categories)).toBe('csd')
    // Lager is inactive, so a lager falls back to Beer.
    expect(suggestCategory(['en:beers', 'en:lagers'], categories)).toBe('beer')
    expect(suggestCategory(['en:snacks'], categories)).toBeNull()
  })

  it('builds an editable SKU from the name and size', () => {
    expect(suggestSku('Coca-Cola', '330 ml')).toBe('COCA-COLA-330')
    expect(suggestSku('Aqua Mineral Water Still', '1.5 L')).toBe('AQUA-MINERAL-WATER-15')
    expect(suggestSku('Red Bull', null)).toBe('RED-BULL')
  })

  it('adds the size to the name only when it is missing', () => {
    const base = { brand: 'Coca-Cola', categoryTags: ['en:colas'] }
    expect(suggestProduct({ ...base, name: 'Coca-Cola', quantity: '330 ml' }, categories).name).toBe(
      'Coca-Cola 330 ml',
    )
    expect(suggestProduct({ ...base, name: 'Coca-Cola 330 ml', quantity: '330 ml' }, categories).name).toBe(
      'Coca-Cola 330 ml',
    )
  })
})
