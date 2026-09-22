import { computed, reactive, ref } from 'vue'

/**
 * Cursor pagination state for a list endpoint.
 *
 * The API returns an opaque `next_cursor` and `has_more` with no total count,
 * so numbered pages are impossible: there is no offset to jump to and no total
 * to divide. This tracks the cursor that produced each visited page, which is
 * what makes a Previous button possible at all.
 *
 * `load` receives the cursor for the page to fetch (undefined for the first)
 * and returns the next cursor, or null when the list ends.
 */
export function useCursorPages(load: (after?: string) => Promise<string | null>) {
  // One entry per visited page: the cursor used to fetch it. The first page
  // was fetched without one.
  const history = ref<(string | undefined)[]>([undefined])
  const nextCursor = ref<string | null>(null)
  const busy = ref(false)

  const pageNumber = computed(() => history.value.length)
  const canGoBack = computed(() => history.value.length > 1)
  const canGoForward = computed(() => nextCursor.value !== null)

  async function fetchPage(after?: string) {
    busy.value = true
    try {
      nextCursor.value = await load(after)
    } finally {
      busy.value = false
    }
  }

  /** Reload from the beginning. Use whenever a filter changes. */
  async function reset() {
    history.value = [undefined]
    await fetchPage(undefined)
  }

  async function next() {
    if (!nextCursor.value || busy.value) return
    const cursor = nextCursor.value
    history.value.push(cursor)
    await fetchPage(cursor)
  }

  async function previous() {
    if (!canGoBack.value || busy.value) return
    history.value.pop()
    await fetchPage(history.value[history.value.length - 1])
  }

  // Returned reactive rather than as loose refs: callers use it as
  // `pager.canGoForward` in templates, where a ref nested in a plain object
  // would not be unwrapped.
  return reactive({ pageNumber, canGoBack, canGoForward, busy, reset, next, previous })
}
