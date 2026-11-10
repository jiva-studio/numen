/**
 * A book tab whose text ranges the window highlights: the tab passes them to its reader.
 */
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { h, nextTick, reactive, ref, shallowRef } from 'vue'
import BookTab from './BookTab.vue'
import type { TextRange } from '../types'

const reader = vi.hoisted(() => ({
  focusSpans: vi.fn(async (..._spans: readonly TextRange[]) => undefined),
}))

vi.mock('../model/useBookReader', () => ({
  useBookReader: () => ({
    path: '',
    reportFailure: () => undefined,
    close: () => undefined,
    state: {
      isLoading: ref(false),
      contents: shallowRef([]),
      span: ref(null),
    },
    place: {
      offset: ref(0),
      documentSpan: ref(null),
      target: ref(null),
      chapter: ref(null),
    },
    rendering: {
      markup: ref(''),
      renderedChapter: ref(''),
      highlights: shallowRef([]),
      otherHighlights: shallowRef([]),
    },
    actions: {
      getCfi: async () => null,
      goToCfi: async () => false,
      goToOffset: async () => undefined,
      moveTo: async () => undefined,
      followLink: async () => undefined,
      focusSpans: reader.focusSpans,
    },
  }),
}))

vi.mock('./BookContent.vue', () => ({
  default: () => h('div', { class: 'book-content-stub' }),
}))

vi.mock('./book-contents', () => ({
  BookListing: () => h('nav'),
}))

vi.mock('@numen/ui', () => ({
  LoadingIndicator: () => h('span'),
}))

const RANGES: readonly TextRange[] = [
  { start: 6_500, end: 6_520 },
  { start: 100, end: 110 },
]

describe('the ranges of a book tab component', () => {
  it('passes the ranges the window sets to the reader of the book', async () => {
    reader.focusSpans.mockClear()
    const state = reactive({
      path: 'library/Marsh Book.epub',
      selector: null,
      ranges: null as readonly TextRange[] | null,
      savePlace: vi.fn(),
    })
    const tab = mount(BookTab, { props: { state } })
    await nextTick()
    expect(reader.focusSpans).not.toHaveBeenCalled()

    state.ranges = RANGES
    await nextTick()

    expect(reader.focusSpans).toHaveBeenCalledWith(...RANGES)
    tab.unmount()
  })
})
