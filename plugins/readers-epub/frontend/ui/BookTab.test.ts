/**
 * A book tab whose text ranges the window highlights: the tab passes them to its reader.
 */
import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { reactive } from 'vue'
import BookTab from './BookTab.vue'
import type { Book, Books, TextRange } from '../types'

const BOOK: Book = {
  title: 'Marsh Book',
  span: { start: 0, end: 9_000 },
  documents: [
    { path: 'text/part0001.xhtml', span: { start: 0, end: 3_000 } },
    { path: 'text/part0002.xhtml', span: { start: 3_000, end: 6_000 } },
    { path: 'text/part0003.xhtml', span: { start: 6_000, end: 9_000 } },
  ],
  parts: [{ title: 'Chapter One', offset: 0, level: 0 }],
  printedPages: [],
  pages: 3,
  pageBytes: 3_000,
  stat: '20480 1700000000000000000 marsh.epub',
}

function createFakeBooks(readChapters: string[]): Books {
  return {
    getBook: async () => BOOK,
    readMarkup: async (_path, document) => {
      readChapters.push(document)
      return `<p data-offset="6500">${document}</p>`
    },
    getEntryUrl: (path, name) => `/assets/${path}/${name}`,
    getCfi: async (_path, offset) => `epubcfi(/6/2!/4/2/1:${offset})`,
    findOffset: async () => null,
  }
}

const RANGES: readonly TextRange[] = [
  { start: 6_500, end: 6_520 },
  { start: 100, end: 110 },
]

describe('the ranges of a book tab component', () => {
  it('passes the ranges the window sets to the reader of the book', async () => {
    const readChapters: string[] = []
    const books = createFakeBooks(readChapters)
    const state = reactive({
      path: 'library/Marsh Book.epub',
      selector: null,
      ranges: null as readonly TextRange[] | null,
      savePlace: vi.fn(),
    })
    const tab = mount(BookTab, { props: { state, books } })
    await flushPromises()
    expect(readChapters).toStrictEqual(['text/part0001.xhtml'])

    state.ranges = RANGES
    await flushPromises()

    expect(readChapters).toStrictEqual(['text/part0001.xhtml', 'text/part0003.xhtml'])
    expect(tab.find('.book__paper').html()).toContain('text/part0003.xhtml')
    tab.unmount()
  })
})
