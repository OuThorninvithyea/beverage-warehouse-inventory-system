import { describe, expect, it } from 'vitest'

import { useCursorPages } from './cursor-pages'

/** A list of 5 pages worth of cursors: page N is fetched with cursors[N-1]. */
function fakeEndpoint(pageCount: number) {
  const calls: (string | undefined)[] = []
  const load = async (after?: string) => {
    calls.push(after)
    const index = after ? Number(after.replace('cursor-', '')) : 0
    return index + 1 < pageCount ? `cursor-${index + 1}` : null
  }
  return { calls, load }
}

describe('useCursorPages', () => {
  it('starts on page one with no cursor and no way back', async () => {
    const endpoint = fakeEndpoint(3)
    const pager = useCursorPages(endpoint.load)
    await pager.reset()

    expect(endpoint.calls).toEqual([undefined])
    expect(pager.pageNumber).toBe(1)
    expect(pager.canGoBack).toBe(false)
    expect(pager.canGoForward).toBe(true)
  })

  it('walks forward and back over the cursor history', async () => {
    const endpoint = fakeEndpoint(3)
    const pager = useCursorPages(endpoint.load)
    await pager.reset()

    await pager.next()
    expect(pager.pageNumber).toBe(2)
    await pager.next()
    expect(pager.pageNumber).toBe(3)
    // Last page: nothing further to fetch.
    expect(pager.canGoForward).toBe(false)

    await pager.next()
    expect(pager.pageNumber).toBe(3)

    await pager.previous()
    expect(pager.pageNumber).toBe(2)
    await pager.previous()
    expect(pager.pageNumber).toBe(1)
    expect(pager.canGoBack).toBe(false)

    // Going back re-fetches with the cursor that produced each page, which is
    // the only way a cursor API can move backwards.
    expect(endpoint.calls).toEqual([
      undefined, 'cursor-1', 'cursor-2', 'cursor-1', undefined,
    ])
  })

  it('cannot be pushed before the first page', async () => {
    const endpoint = fakeEndpoint(2)
    const pager = useCursorPages(endpoint.load)
    await pager.reset()

    await pager.previous()
    expect(pager.pageNumber).toBe(1)
    expect(endpoint.calls).toEqual([undefined])
  })

  it('reset returns to the first page after a filter change', async () => {
    const endpoint = fakeEndpoint(4)
    const pager = useCursorPages(endpoint.load)
    await pager.reset()
    await pager.next()
    await pager.next()
    expect(pager.pageNumber).toBe(3)

    await pager.reset()
    expect(pager.pageNumber).toBe(1)
    expect(pager.canGoBack).toBe(false)
    expect(endpoint.calls.at(-1)).toBeUndefined()
  })

  it('reports a single page as having nowhere to go', async () => {
    const endpoint = fakeEndpoint(1)
    const pager = useCursorPages(endpoint.load)
    await pager.reset()

    expect(pager.canGoBack).toBe(false)
    expect(pager.canGoForward).toBe(false)
  })
})
