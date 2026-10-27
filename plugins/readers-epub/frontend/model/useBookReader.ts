/**
 * Reader state and controls for an open book.
 */
import { computed, ref, shallowRef, type Ref, type ShallowRef } from 'vue'
import type { ContentsEntry } from '../book'
import { formatErrorMessage } from '@numen/wire'
import type { Span } from '@numen/plugin-sdk'
import type { MessageWriter } from './useBookDocument'
import { getContents, getDocumentAtOffset, getPageNumber } from '../lib/pagination'
import { useBookDocument } from './useBookDocument'
import { useBookHighlights } from './useBookHighlights'
import type {
  Book,
  BookHandle,
  BookPart,
  Books,
  BookWords,
  PrintedPage,
  SpineDocument,
} from '../types'

export type { Book, BookHandle, BookPart, Books, BookWords, PrintedPage, SpineDocument }
export { getContents, getDocumentAtOffset, getPageNumber } from '../lib/pagination'
export { useBookDocument } from './useBookDocument'
export { useBookHighlights } from './useBookHighlights'

export type BookReaderState = ReturnType<typeof useBookReader>

export interface BookReaderOptions {
  words: BookWords
  writeMessage?: MessageWriter
}

const getChapterTitle = (contents: readonly ContentsEntry[], offset: number): string => {
  let found = ''
  for (const entry of contents) {
    if (entry.at > offset) break
    found = entry.title
  }
  return found
}

interface BookNavigationDeps {
  documents: ShallowRef<readonly SpineDocument[]>
  span: Ref<Span>
  bookDocument: ReturnType<typeof useBookDocument>
  getShape: () => Promise<void>
  isOpen: () => boolean
}

function useBookNavigation(deps: BookNavigationDeps) {
  const offset = ref(0)

  const goToOffset = async (targetOffset: number) => {
    await deps.getShape()
    if (!deps.isOpen() || deps.documents.value.length === 0) return
    const last = Math.max(deps.span.value.to - 1, deps.span.value.from)
    offset.value = Math.min(Math.max(Math.trunc(targetOffset), deps.span.value.from), last)
    const doc = getDocumentAtOffset(deps.documents.value, offset.value)
    if (doc) await deps.bookDocument.draw(doc)
  }

  const followLink = async (target: string) => {
    await deps.getShape()
    const targetDoc = deps.documents.value.find((one: SpineDocument) => one.path === target)
    if (!targetDoc) return
    await goToOffset(targetDoc.span.from)
  }

  return { offset, goToOffset, followLink }
}

async function loadBookData(
  target: { books: Books; path: string },
  state: {
    title: Ref<string>
    span: Ref<Span>
    documents: ShallowRef<readonly SpineDocument[]>
    pages: Ref<number>
    pageBytes: Ref<number>
    contents: ShallowRef<readonly ContentsEntry[]>
    stat: Ref<string>
    offset: Ref<number>
  },
  deps: {
    words: BookWords
    writeMessage: (msg: string, level: 'error') => void
    isOpen: () => boolean
    drawDocument: (doc: SpineDocument) => Promise<void>
  },
) {
  try {
    const book = await target.books.getBook(target.path)
    if (!deps.isOpen()) return
    state.title.value = book.title
    state.span.value = book.span
    state.documents.value = book.documents
    state.pages.value = book.pages
    state.pageBytes.value = book.pageBytes
    state.contents.value = getContents(book, deps.words)
    state.stat.value = book.stat
    state.offset.value = book.span.from
    const doc = getDocumentAtOffset(book.documents, book.span.from)
    if (doc) await deps.drawDocument(doc)
  } catch (error) {
    if (!deps.isOpen()) return
    deps.writeMessage(formatErrorMessage(error), 'error')
  }
}

function useBookState() {
  const title = ref('')
  const span = ref<Span>({ from: 0, to: 0 })
  const documents = shallowRef<readonly SpineDocument[]>([])
  const contents = shallowRef<readonly ContentsEntry[]>([])
  const pages = ref(0)
  const pageBytes = ref(0)
  // Stryker disable next-line StringLiteral: the file stat is assigned before the first document is drawn
  const stat = ref('')
  const isLoading = ref(true)

  return { title, span, documents, contents, pages, pageBytes, stat, isLoading }
}

function useBookProgress(
  state: {
    pageBytes: Ref<number>
    pages: Ref<number>
    contents: ShallowRef<readonly ContentsEntry[]>
  },
  offset: Ref<number>,
) {
  const page = computed(() => getPageNumber(state.pageBytes.value, state.pages.value, offset.value))
  const chapter = computed(() => getChapterTitle(state.contents.value, offset.value))
  return { page, chapter }
}

function createBookCloser(
  bookDocument: ReturnType<typeof useBookDocument>,
  state: ReturnType<typeof useBookState>,
  onClose: () => void,
) {
  return () => {
    onClose()
    bookDocument.close()
    state.contents.value = []
    state.pages.value = 0
    state.pageBytes.value = 0
  }
}

export function useBookReader(books: Books, path: string, options: BookReaderOptions) {
  const { words, writeMessage = () => {} } = options
  const state = useBookState()
  let isOpen = true

  const bookDocument = useBookDocument(books, path, {
    stat: state.stat,
    writeMessage,
    isOpen: () => isOpen,
  })

  const getShape = () => shape
  const nav = useBookNavigation({
    documents: state.documents,
    span: state.span,
    bookDocument,
    getShape,
    isOpen: () => isOpen,
  })

  const loadBook = async () => {
    await loadBookData(
      { books, path },
      { ...state, offset: nav.offset },
      { words, writeMessage, isOpen: () => isOpen, drawDocument: (doc) => bookDocument.draw(doc) },
    )
    state.isLoading.value = false
  }
  const shape = loadBook()

  const { page, chapter } = useBookProgress(state, nav.offset)
  const marks = useBookHighlights(nav.goToOffset, () => isOpen)
  const focusSpans = async (...spans: readonly Span[]) => {
    await shape
    await marks.focusSpans(...spans)
  }

  // A saved place and a search hit name the book by EPUB CFI, which the plugin's backend reads.
  const getCfi = (offset: number) => books.getCfi(path, offset)
  const goToCfi = async (cfi: string) => {
    await shape
    const offset = await books.findOffset(path, cfi)
    if (offset === null || !isOpen) return false
    await nav.goToOffset(offset)
    return true
  }

  const close = createBookCloser(bookDocument, state, () => {
    isOpen = false
  })

  const reader = {
    path,
    ...state,
    page,
    chapter,
    offset: nav.offset,
    documentSpan: bookDocument.documentSpan,
    renderedChapter: bookDocument.renderedChapter,
    markup: bookDocument.markup,
    highlights: marks.highlights,
    otherHighlights: marks.otherHighlights,
    goToOffset: nav.goToOffset,
    goToCfi,
    getCfi,
    followLink: nav.followLink,
    focusSpans,
    close,
  }
  return reader
}
