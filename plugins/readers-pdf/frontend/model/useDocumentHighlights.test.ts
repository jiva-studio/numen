import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useDocumentHighlights as useDocumentHighlightsOriginal } from './useDocumentHighlights'
import type { Documents, PageHighlight, Rect } from '../types'

type HighlightsOptions = Parameters<typeof useDocumentHighlightsOriginal>[2]

const useDocumentHighlights = (
  documents: Documents,
  path: string,
  pageNumber: HighlightsOptions['pageNumber'],
  error: HighlightsOptions['error'],
  onGoToPage: HighlightsOptions['onGoToPage'],
  isOpen: HighlightsOptions['isOpen'],
) => useDocumentHighlightsOriginal(documents, path, { pageNumber, error, onGoToPage, isOpen })

describe('useDocumentHighlights', () => {
  const rect: Rect = { minX: 0.1, minY: 0.2, maxX: 0.3, maxY: 0.4 }
  const pageHighlights: PageHighlight[] = [{ page: 1, rects: [rect] }]

  it('sets and looks up page highlights', async () => {
    const documents: Documents = {
      getDocumentLayout: async () => ({ pages: [], stat: '' }),
      getPageUrl: () => '',
      getHighlights: async () => [pageHighlights],
    }
    const pageNumber = ref(0)
    const error = ref('')
    const onGoToPage = vi.fn(async (page: number) => {
      pageNumber.value = page
    })

    const hl = useDocumentHighlights(
      documents,
      'doc.pdf',
      pageNumber,
      error,
      onGoToPage,
      () => true,
    )

    await hl.applyHighlights([pageHighlights])
    expect(onGoToPage).toHaveBeenCalledWith(1)
    expect(pageNumber.value).toBe(1)
    expect(hl.getHighlightsOn(1)).toEqual([rect])
    expect(hl.highlighted.value).toEqual([rect])
  })

  const createDocumentsOf = (getHighlights: Documents['getHighlights']): Documents => ({
    getDocumentLayout: async () => ({ pages: [], stat: '' }),
    getPageUrl: () => '',
    getHighlights,
  })

  const createHighlightsOf = (documents: Documents, isOpen: () => boolean = () => true) => {
    const pageNumber = ref(0)
    const error = ref('')
    const onGoToPage = vi.fn(async (page: number) => {
      pageNumber.value = page
    })
    const hl = useDocumentHighlights(documents, 'doc.pdf', pageNumber, error, onGoToPage, isOpen)
    return { hl, error, onGoToPage }
  }

  it('holds no highlights before any are applied', () => {
    const { hl } = createHighlightsOf(createDocumentsOf(async () => []))

    expect(hl.highlights.value).toStrictEqual([])
    expect(hl.otherHighlights.value).toStrictEqual([])
    expect(hl.getHighlightsOn(0)).toStrictEqual([])
    expect(hl.getOtherHighlightsOn(0)).toStrictEqual([])
  })

  it('holds no highlights and turns to no page where none are applied', async () => {
    const { hl, onGoToPage } = createHighlightsOf(createDocumentsOf(async () => []))
    await hl.applyHighlights([[{ page: 1, rects: [rect] }]])

    await hl.applyHighlights([])

    expect(hl.highlights.value).toStrictEqual([])
    expect(hl.otherHighlights.value).toStrictEqual([])
    expect(onGoToPage).toHaveBeenCalledTimes(1)
  })

  it('asks nothing of the document for no spans', async () => {
    const getHighlights = vi.fn<Documents['getHighlights']>(async () => [pageHighlights])
    const { hl, onGoToPage } = createHighlightsOf(createDocumentsOf(getHighlights))

    await hl.focusSpans()

    expect(getHighlights).not.toHaveBeenCalled()
    expect(onGoToPage).not.toHaveBeenCalled()
  })

  it('asks nothing of the document once it is closed', async () => {
    const getHighlights = vi.fn<Documents['getHighlights']>(async () => [pageHighlights])
    const { hl, onGoToPage } = createHighlightsOf(createDocumentsOf(getHighlights), () => false)

    await hl.focusSpans({ from: 0, to: 4 })

    expect(getHighlights).not.toHaveBeenCalled()
    expect(onGoToPage).not.toHaveBeenCalled()
  })

  it('applies nothing that arrives after the document is closed', async () => {
    let open = true
    const { hl, onGoToPage } = createHighlightsOf(
      createDocumentsOf(async () => {
        open = false
        return [pageHighlights]
      }),
      () => open,
    )

    await hl.focusSpans({ from: 0, to: 4 })

    expect(hl.highlights.value).toStrictEqual([])
    expect(onGoToPage).not.toHaveBeenCalled()
  })

  it('says nothing of a failure that arrives after the document is closed', async () => {
    let open = true
    const { hl, error } = createHighlightsOf(
      createDocumentsOf(async () => {
        open = false
        throw new Error('the layer is being written')
      }),
      () => open,
    )

    await hl.focusSpans({ from: 0, to: 4 })

    expect(error.value).toBe('')
  })
})
