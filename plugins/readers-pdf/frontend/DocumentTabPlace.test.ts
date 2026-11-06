/**
 * The place a PDF tab saves and reports to the window: the page shown and the number of pages of the
 * document, when the reader navigates to another page and when the window moves the tab to a passage. The
 * tab's own reader runs over a made-up port.
 */
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h, shallowRef } from 'vue'
import type { Selector, TabPlace } from '@numen/plugin-sdk'
import DocumentTab from './DocumentTab.vue'
import { useDocumentReader } from './model/useDocumentReader'
import { useDocumentTab } from './model/useDocumentTab'
import type { Documents, TextRange } from './types'

vi.mock('./reader', () => ({
  Reader: defineComponent({
    name: 'Reader',
    setup: () => () => h('div', { class: 'reader-stub' }),
  }),
}))

vi.mock('@numen/ui', () => ({
  LoadingIndicator: defineComponent({ setup: () => () => h('span') }),
}))

const PATH = 'library/Wading Birds.pdf'

/** The passage on the seventh page, and another on the second. */
const RANGES: readonly TextRange[] = [
  { start: 18_204, end: 18_390 },
  { start: 2_310, end: 2_355 },
]

/** Resolves once every pending request has completed. */
const flush = () => new Promise((resolve) => setTimeout(resolve, 0))

/** A document of twelve pages whose layout arrives when `release` is called. */
const createDocuments = () => {
  let release = () => undefined as void
  const layout = new Promise<void>((resolve) => {
    release = resolve
  })
  const documents: Documents = {
    getDocumentLayout: async () => {
      await layout
      return {
        pages: Array.from({ length: 12 }, () => ({ width: 600, height: 800 })),
        stat: 'birds-1',
      }
    },
    getPageUrl: () => '',
    getHighlights: async () => [
      [{ page: 6, rects: [{ minX: 0.1, minY: 0.2, maxX: 0.8, maxY: 0.3 }] }],
      [{ page: 1, rects: [{ minX: 0.1, minY: 0.5, maxX: 0.6, maxY: 0.55 }] }],
    ],
  }
  return { documents, release: () => release() }
}

const mountTab = (documents: Documents) => {
  const reader = useDocumentTab(useDocumentReader(documents, PATH))
  const ranges = shallowRef<readonly TextRange[] | null>(null)
  const savePlace = vi.fn((_place: Selector) => undefined)
  const reportPlace = vi.fn((_place: TabPlace) => undefined)
  const state = Object.defineProperty(
    Object.assign(reader, { title: 'Wading Birds', selector: null, savePlace, reportPlace }),
    'ranges',
    { get: () => ranges.value, enumerable: true },
  )
  const tab = mount(DocumentTab, { props: { state } })
  return { tab, ranges, savePlace, reportPlace, reader }
}

describe('the place a PDF tab saves when the window moves it', () => {
  it('is the page of the first range the window sets on an open tab', async () => {
    const { documents, release } = createDocuments()
    release()
    const { tab, ranges, savePlace } = mountTab(documents)
    await flush()
    savePlace.mockClear()

    ranges.value = [...RANGES]
    await flush()

    expect(savePlace).toHaveBeenLastCalledWith({ page: 7, pageCount: 12 })
    tab.unmount()
  })

  it('is the page of the first range the window sets while the tab is still loading', async () => {
    const { documents, release } = createDocuments()
    const { tab, ranges, savePlace } = mountTab(documents)
    ranges.value = [...RANGES]
    await flush()

    release()
    await flush()

    expect(savePlace).toHaveBeenLastCalledWith({ page: 7, pageCount: 12 })
    tab.unmount()
  })
})

describe('the place a PDF tab saves when the reader navigates to another page', () => {
  it('is the page shown, from 1, and the number of pages of the document', async () => {
    const { documents, release } = createDocuments()
    release()
    const { tab, savePlace, reader } = mountTab(documents)
    await flush()
    savePlace.mockClear()

    await reader.goToPage(3)
    await flush()

    expect(savePlace).toHaveBeenLastCalledWith({ page: 4, pageCount: 12 })
    tab.unmount()
  })
})

describe('the place a PDF tab reports to the window', () => {
  it('is the page shown, from 1, and the page count when the reader navigates to another page', async () => {
    const { documents, release } = createDocuments()
    release()
    const { tab, reportPlace, reader } = mountTab(documents)
    await flush()
    reportPlace.mockClear()

    await reader.goToPage(3)
    await flush()

    expect(reportPlace).toHaveBeenLastCalledWith({ page: 4, pageCount: 12 })
    tab.unmount()
  })

  it('is the page of the first range the window sets on an open tab', async () => {
    const { documents, release } = createDocuments()
    release()
    const { tab, ranges, reportPlace } = mountTab(documents)
    await flush()
    reportPlace.mockClear()

    ranges.value = [...RANGES]
    await flush()

    expect(reportPlace).toHaveBeenLastCalledWith({ page: 7, pageCount: 12 })
    tab.unmount()
  })

  it('is nothing while the document is loading', async () => {
    const { documents } = createDocuments()
    const { tab, reportPlace } = mountTab(documents)
    await flush()

    expect(reportPlace).not.toHaveBeenCalled()
    tab.unmount()
  })
})
