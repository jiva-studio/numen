import { describe, expect, it } from 'vitest'
import { shallowRef } from 'vue'
import { useDocumentNavigation } from './useDocumentNavigation'
import type { Page } from '../types'

describe('useDocumentNavigation', () => {
  it('navigates within page bounds', async () => {
    const pages = shallowRef<readonly Page[]>([
      { width: 612, height: 792 },
      { width: 612, height: 792 },
      { width: 612, height: 792 },
    ])
    const nav = useDocumentNavigation(pages)

    expect(nav.pageNumber.value).toBe(0)

    await nav.nextPage()
    expect(nav.pageNumber.value).toBe(1)

    await nav.nextPage()
    expect(nav.pageNumber.value).toBe(2)

    await nav.nextPage()
    expect(nav.pageNumber.value).toBe(2)

    await nav.prevPage()
    expect(nav.pageNumber.value).toBe(1)

    await nav.goToPage(0)
    expect(nav.pageNumber.value).toBe(0)
  })

  it('does nothing when pages are empty', async () => {
    const pages = shallowRef<readonly Page[]>([])
    const nav = useDocumentNavigation(pages)

    await nav.goToPage(5)
    expect(nav.pageNumber.value).toBe(0)
  })
})
