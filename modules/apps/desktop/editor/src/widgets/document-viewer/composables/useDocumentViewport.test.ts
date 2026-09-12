import { describe, expect, it } from 'vitest'
import { ref, shallowRef } from 'vue'
import { useDocumentViewport } from './useDocumentViewport'
import type { Documents, Page } from '../types'

describe('useDocumentViewport', () => {
  const documents: Documents = {
    getDocumentLayout: async () => ({ pages: [], fingerprint: '' }),
    getPageUrl: (path, page, width, fingerprint) => `${path}/p${page}?w=${width}&s=${fingerprint ?? ''}`,
    getHighlights: async () => [],
  }

  it('clamps width within range and constructs page URL', () => {
    const pages = shallowRef<readonly Page[]>([{ width: 100, height: 200 }])
    const pageNumber = ref(0)
    const fingerprint = ref('v1')
    const viewport = useDocumentViewport(documents, 'doc.pdf', pages, pageNumber, fingerprint, () => true)

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
