/**
 * The place a PDF tab reports to the window as soon as its document has loaded, before the reader
 * does anything: the first page, or the page of the selector it opens at. The tab's own reader runs
 * over a made-up port.
 */
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import type { ReaderTabState, Selector, TabPlace } from '@numen/plugin-sdk'
import DocumentTab from './DocumentTab.vue'
import type { Documents } from '../types'

const PATH = 'library/Marsh Birds.pdf'

const flush = () => new Promise((resolve) => setTimeout(resolve, 0))

/** A document of nine pages. */
const documents: Documents = {
  getDocumentLayout: async () => ({
    pages: Array.from({ length: 9 }, () => ({ width: 600, height: 800 })),
    stat: 'marsh-1',
  }),
  getPageUrl: () => '',
  getHighlights: async () => [],
}

const mountTab = (selector: Selector | null) => {
  const savePlace = vi.fn((_place: Selector) => undefined)
  const reportPlace = vi.fn((_place: TabPlace) => undefined)
  const state: ReaderTabState = {
    path: PATH,
    title: 'Marsh Birds',
    selector,
    ranges: null,
    savePlace,
    reportPlace,
  }
  const tab = mount(DocumentTab, { props: { state, documents } })
  return { tab, reportPlace }
}

describe('the place a PDF tab reports when it opens', () => {
  it('is the first page and the number of pages once the document has loaded', async () => {
    const { tab, reportPlace } = mountTab(null)
    await flush()
    await flush()

    expect(reportPlace).toHaveBeenLastCalledWith({ page: 1, pageCount: 9 })
    tab.unmount()
  })

  it('is the page of the selector it opens at', async () => {
    const { tab, reportPlace } = mountTab({ page: 5, pageCount: 9 })
    await flush()
    await flush()

    expect(reportPlace).toHaveBeenLastCalledWith({ page: 5, pageCount: 9 })
    tab.unmount()
  })
})
