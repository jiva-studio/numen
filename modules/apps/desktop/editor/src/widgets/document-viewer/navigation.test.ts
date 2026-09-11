import { describe, expect, it } from 'vitest'
import { shallowRef } from 'vue'
import { useDocumentNavigation } from './navigation'
import type { Page } from './types'

describe('useDocumentNavigation', () => {
  it('navigates within page bounds', async () => {
    const pages = shallowRef<readonly Page[]>([
      { width: 612, height: 792 },
      { width: 612, height: 792 },
      { width: 612, height: 792 },
    ])
    const nav = useDocumentNavigation(pages)

    expect(nav.at.value).toBe(0)

    await nav.next()
    expect(nav.at.value).toBe(1)

    await nav.next()
    expect(nav.at.value).toBe(2)

    await nav.next()
    expect(nav.at.value).toBe(2)

    await nav.back()
    expect(nav.at.value).toBe(1)

    await nav.go(0)
    expect(nav.at.value).toBe(0)
  })

  it('does nothing when pages are empty', async () => {
    const pages = shallowRef<readonly Page[]>([])
    const nav = useDocumentNavigation(pages)

    await nav.go(5)
    expect(nav.at.value).toBe(0)
  })
})
