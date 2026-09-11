import { describe, expect, it } from 'vitest'
import { ref, shallowRef } from 'vue'
import { useDocumentViewport } from './viewport'
import type { Documents, Page } from './types'

describe('useDocumentViewport', () => {
  const documents: Documents = {
    getDocumentLayout: async () => ({ pages: [], fingerprint: '' }),
    getPageUrl: (path, page, width, seen) => `${path}/p${page}?w=${width}&s=${seen ?? ''}`,
    getHighlights: async () => [],
  }

  it('clamps width within range and constructs page URL', () => {
    const pages = shallowRef<readonly Page[]>([{ width: 100, height: 200 }])
    const at = ref(0)
    const seen = ref('v1')
    const viewport = useDocumentViewport(documents, 'doc.pdf', pages, at, seen, () => true)

    expect(viewport.wide.value).toBe(0)
    expect(viewport.getPageImageUrl(0)).toBe('')

    viewport.widen(800)
    expect(viewport.wide.value).toBe(800)
    expect(viewport.getPageImageUrl(0)).toBe('doc.pdf/p0?w=800&s=v1')
    expect(viewport.picture.value).toBe('doc.pdf/p0?w=800&s=v1')

    viewport.widen(5000)
    expect(viewport.wide.value).toBe(4096)
  })
})
