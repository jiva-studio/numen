/**
 * A PDF tab the window asks to light ranges of the document's text: the reader goes to the first and
 * lights all of them.
 */
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h, nextTick, reactive, ref, shallowRef } from 'vue'
import DocumentTab from './DocumentTab.vue'
import type { TextRange } from './types'

const reader = vi.hoisted(() => ({
  focusSpans: vi.fn(async (..._spans: readonly TextRange[]) => undefined),
}))

vi.mock('./model/useDocumentReader', () => ({
  useDocumentReader: (path: string) => ({
    path,
    pages: shallowRef([{ width: 600, height: 800 }]),
    pageNumber: ref(0),
    isLoading: ref(false),
    error: ref(''),
    getPageImageUrl: () => '',
    getHighlightsOn: () => [],
    getOtherHighlightsOn: () => [],
    goToPage: async () => undefined,
    applyHighlights: async () => undefined,
    widen: () => undefined,
    reportFailure: () => undefined,
    close: () => undefined,
    focusSpans: reader.focusSpans,
  }),
}))

vi.mock('./reader', () => ({
  Reader: defineComponent({
    name: 'Reader',
    setup: () => () => h('div', { class: 'reader-stub' }),
  }),
}))

vi.mock('@numen/ui', () => ({
  LoadingIndicator: defineComponent({ setup: () => () => h('span') }),
}))

const RANGES: readonly TextRange[] = [
  { start: 40_512, end: 40_543 },
  { start: 1_200, end: 1_260 },
]

const mountTab = (ranges: readonly TextRange[] | null) => {
  reader.focusSpans.mockClear()
  const state = reactive({
    path: 'library/Birds.pdf',
    title: 'Birds',
    selector: null,
    ranges,
    savePlace: vi.fn(),
  })
  const tab = mount(DocumentTab, { props: { state } })
  return { tab, state }
}

describe('the ranges of a PDF tab', () => {
  it('lights the ranges the tab opens with, in their order', async () => {
    const { tab } = mountTab(RANGES)
    await nextTick()

    expect(reader.focusSpans).toHaveBeenCalledWith(...RANGES)
    tab.unmount()
  })

  it('lights the ranges the window sets later, each time it sets them', async () => {
    const { tab, state } = mountTab(null)
    await nextTick()
    expect(reader.focusSpans).not.toHaveBeenCalled()

    state.ranges = RANGES
    await nextTick()
    state.ranges = [...RANGES]
    await nextTick()

    expect(reader.focusSpans).toHaveBeenCalledTimes(2)
    expect(reader.focusSpans).toHaveBeenLastCalledWith(...RANGES)
    tab.unmount()
  })
})
