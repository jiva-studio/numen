import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useDocumentHighlights } from './useDocumentHighlights'
import type { Documents, PageHighlight, Rect } from './types'

describe('useDocumentHighlights', () => {
  const rect: Rect = { minX: 0.1, minY: 0.2, maxX: 0.3, maxY: 0.4 }
  const pageHighlights: PageHighlight[] = [{ page: 1, rects: [rect] }]

  it('sets and looks up page highlights', async () => {
    const documents: Documents = {
      getDocumentLayout: async () => ({ pages: [], fingerprint: '' }),
      getPageUrl: () => '',
      getHighlights: async () => [pageHighlights],
    }
    const pageNumber = ref(0)
    const error = ref('')
    const onGoToPage = vi.fn(async (page: number) => {
      pageNumber.value = page
    })

    const hl = useDocumentHighlights(documents, 'doc.pdf', pageNumber, error, onGoToPage, () => true)

    await hl.applyHighlights([pageHighlights])
    expect(onGoToPage).toHaveBeenCalledWith(1)
    expect(pageNumber.value).toBe(1)
    expect(hl.highlightedOn(1)).toEqual([rect])
    expect(hl.highlighted.value).toEqual([rect])
  })
})
