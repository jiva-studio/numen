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
    close: () => undefined,
    state: {
      isLoading: ref(false),
      contents: shallowRef([]),
      span: ref({ start: 0, end: 0 }),
    },
    progress: { page: ref(0), chapter: ref('') },
    navigation: {
      offset: ref(0),
      target: ref(null),
      goToOffset: async () => undefined,
      moveTo: async () => undefined,
      followLink: async () => undefined,
    },
    bookDocument: {
      documentSpan: ref({ start: 0, end: 0 }),
      renderedChapter: ref(''),
      markup: ref(''),
    },
    marks: {
      highlights: shallowRef([]),
      otherHighlights: shallowRef([]),
    },
    actions: {
      getCfi: async () => null,
      goToCfi: async () => false,
      focusSpans: reader.focusSpans,
      reportFailure: () => undefined,
    },
  }),
}))

vi.mock('../book', () => ({
  Book: () => h('div', { class: 'book-stub' }),
  BookContents: () => h('nav'),
}))

vi.mock('@numen/ui', () => ({
  LoadingIndicator: () => h('span'),
  win: { addEventListener: () => undefined, removeEventListener: () => undefined },
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
