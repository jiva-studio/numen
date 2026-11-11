/**
 * A PDF tab whose text ranges the window highlights: the reader scrolls to the first and
 * highlights all of them.
 */
import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { reactive } from 'vue'
import DocumentTab from './DocumentTab.vue'
import type { Documents, TextRange } from '../types'

const RANGES: readonly TextRange[] = [
  { start: 40_512, end: 40_543 },
  { start: 1_200, end: 1_260 },
]

const createDocuments = () => {
  const getHighlights = vi.fn(async (_path: string, _ranges: readonly TextRange[]) => [
    [{ page: 4, rects: [{ minX: 0.1, minY: 0.2, maxX: 0.8, maxY: 0.3 }] }],
    [{ page: 1, rects: [{ minX: 0.1, minY: 0.5, maxX: 0.6, maxY: 0.55 }] }],
  ])
  const documents: Documents = {
    getDocumentLayout: async () => ({
      pages: Array.from({ length: 8 }, () => ({ width: 600, height: 800 })),
      stat: 'birds-1',
    }),
    getPageUrl: () => '',
    getHighlights,
  }
  return { documents, getHighlights }
}

const mountTab = (ranges: readonly TextRange[] | null) => {
  const { documents, getHighlights } = createDocuments()
  const state = reactive({
    path: 'library/Birds.pdf',
    title: 'Birds',
    selector: null,
    ranges,
    savePlace: vi.fn(),
    reportPlace: vi.fn(),
  })
  const tab = mount(DocumentTab, { props: { state, documents } })
  return { tab, state, getHighlights }
}

describe('the ranges of a PDF tab', () => {
  it('highlights the ranges the tab opens with, in their order', async () => {
    const { tab, state, getHighlights } = mountTab(RANGES)
    await flushPromises()

    expect(getHighlights).toHaveBeenCalledWith('library/Birds.pdf', RANGES)
    expect(state.savePlace).toHaveBeenCalledWith({ page: 5, pageCount: 8 })
    tab.unmount()
  })

  it('highlights the ranges the window sets later, each time it sets them', async () => {
    const { tab, state, getHighlights } = mountTab(null)
    await flushPromises()
    expect(getHighlights).not.toHaveBeenCalled()

    state.ranges = RANGES
    await flushPromises()
    state.ranges = [...RANGES]
    await flushPromises()

    expect(getHighlights).toHaveBeenCalledTimes(2)
    expect(getHighlights).toHaveBeenLastCalledWith('library/Birds.pdf', RANGES)
    tab.unmount()
  })
})
