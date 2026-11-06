/**
 * The place a PDF tab saves when the window moves it to a passage: the page the passage is on,
 * saved as soon as the tab is there. The tab's own reader runs over a made-up port.
 */
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h, shallowRef } from 'vue'
import type { Selector } from '@numen/plugin-sdk'
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

/** Resolves once every pending request has been answered. */
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
  const state = Object.defineProperty(
    Object.assign(reader, { title: 'Wading Birds', selector: null, savePlace }),
    'ranges',
    { get: () => ranges.value, enumerable: true },
  )
  const tab = mount(DocumentTab, { props: { state } })
  return { tab, ranges, savePlace }
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

    expect(savePlace).toHaveBeenLastCalledWith({ page: 7 })
    tab.unmount()
  })

  it('is the page of the first range the window sets while the tab is still loading', async () => {
    const { documents, release } = createDocuments()
    const { tab, ranges, savePlace } = mountTab(documents)
    ranges.value = [...RANGES]
    await flush()

    release()
    await flush()

    expect(savePlace).toHaveBeenLastCalledWith({ page: 7 })
    tab.unmount()
  })
})
