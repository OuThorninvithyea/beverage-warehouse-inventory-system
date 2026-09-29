import type { Category } from '@/api/catalog'

/**
 * A barcode is only a number: the product's name and size live in a database.
 * For a product that is not in our catalog yet, Open Food Facts (a free, open
 * product database) often knows it, so the new-product form can start filled
 * in rather than empty. Every suggestion stays editable, and a failed or empty
 * lookup simply leaves the form for the user to fill.
 */

const OPEN_FOOD_FACTS = 'https://world.openfoodfacts.org/api/v2/product'
const LOOKUP_TIMEOUT_MS = 6000

export interface ExternalProduct {
  name: string
  brand: string | null
  /** Pack size as printed, e.g. "330 ml". */
  quantity: string | null
  categoryTags: string[]
}

export interface ProductSuggestion {
  name: string
  sku: string
  categoryId: string | null
  brand: string | null
}

interface OpenFoodFactsResponse {
  status?: number
  product?: {
    product_name?: string
    product_name_en?: string
    brands?: string
    quantity?: string
    categories_tags?: string[]
  }
}

/** Looks a barcode up in Open Food Facts. Null when unknown or unreachable. */
export async function lookupExternalProduct(
  barcode: string,
  fetcher: typeof fetch = fetch,
): Promise<ExternalProduct | null> {
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), LOOKUP_TIMEOUT_MS)
  try {
    const fields = 'product_name,product_name_en,brands,quantity,categories_tags'
    const response = await fetcher(
      `${OPEN_FOOD_FACTS}/${encodeURIComponent(barcode)}?fields=${fields}`,
      { signal: controller.signal },
    )
    if (!response.ok) return null
    const body = (await response.json()) as OpenFoodFactsResponse
    const product = body.product
    const name = (product?.product_name_en || product?.product_name || '').trim()
    if (body.status !== 1 || !product || !name) return null
    return {
      name,
      brand: product.brands?.split(',')[0]?.trim() || null,
      quantity: product.quantity?.trim() || null,
      categoryTags: product.categories_tags ?? [],
    }
  } catch {
    // Offline, blocked or slow: the form still works by hand.
    return null
  } finally {
    clearTimeout(timer)
  }
}

// Most specific first: a cola is a better match than a soft drink.
const CATEGORY_RULES: Array<{ tags: string[]; categories: string[] }> = [
  { tags: ['colas'], categories: ['Cola', 'Carbonated Soft Drinks'] },
  { tags: ['lemon', 'lime'], categories: ['Lemon and Lime', 'Carbonated Soft Drinks'] },
  { tags: ['tonic', 'mixers'], categories: ['Mixers', 'Carbonated Soft Drinks'] },
  { tags: ['energy-drinks'], categories: ['Energy Drinks'] },
  { tags: ['sparkling-waters', 'carbonated-waters'], categories: ['Sparkling Water', 'Water'] },
  { tags: ['waters', 'mineral-waters', 'spring-waters'], categories: ['Still Water', 'Water'] },
  { tags: ['orange-juices', 'citrus'], categories: ['Citrus Juice', 'Juice'] },
  { tags: ['juices', 'nectars', 'fruit-based-beverages'], categories: ['Juice'] },
  { tags: ['stouts', 'dark-beers'], categories: ['Stout and Dark', 'Beer'] },
  { tags: ['lagers'], categories: ['Lager', 'Beer'] },
  { tags: ['beers'], categories: ['Beer'] },
  { tags: ['wines', 'spirits', 'liquors'], categories: ['Wine and Spirits'] },
  { tags: ['iced-teas'], categories: ['Iced Tea', 'Tea and Coffee'] },
  { tags: ['coffee-drinks', 'iced-coffees', 'coffees'], categories: ['Iced Coffee', 'Tea and Coffee'] },
  { tags: ['teas'], categories: ['Tea and Coffee'] },
  { tags: ['milks', 'dairies', 'dairy-drinks', 'yogurts'], categories: ['Dairy'] },
  { tags: ['sodas', 'soft-drinks', 'carbonated-drinks'], categories: ['Carbonated Soft Drinks'] },
]

/** Matches Open Food Facts category tags to one of our active categories. */
export function suggestCategory(tags: string[], categories: Category[]): string | null {
  const plain = tags.map((tag) => tag.replace(/^[a-z]{2}:/, ''))
  const byName = new Map(
    categories.filter((row) => row.is_active).map((row) => [row.name.toLowerCase(), row.id]),
  )
  for (const rule of CATEGORY_RULES) {
    if (!plain.some((tag) => rule.tags.some((word) => tag === word || tag.endsWith(`-${word}`)))) {
      continue
    }
    for (const name of rule.categories) {
      const id = byName.get(name.toLowerCase())
      if (id) return id
    }
  }
  return null
}

/** "Coca-Cola" + "330 ml" → "COCA-COLA-330". Only a starting point to edit. */
export function suggestSku(name: string, quantity: string | null): string {
  const words = name
    .toUpperCase()
    .replace(/[^A-Z0-9]+/g, '-')
    .split('-')
    .filter(Boolean)
  const size = quantity?.match(/\d+(?:[.,]\d+)?/)?.[0]?.replace(/[.,]/g, '') ?? ''
  const base = words.filter((word) => word !== size).slice(0, 3)
  return [...base, size].filter(Boolean).join('-').slice(0, 40)
}

export function suggestProduct(found: ExternalProduct, categories: Category[]): ProductSuggestion {
  const hasSize = found.quantity && found.name.toLowerCase().includes(found.quantity.toLowerCase())
  return {
    name: found.quantity && !hasSize ? `${found.name} ${found.quantity}` : found.name,
    sku: suggestSku(found.name, found.quantity),
    categoryId: suggestCategory(found.categoryTags, categories),
    brand: found.brand,
  }
}
