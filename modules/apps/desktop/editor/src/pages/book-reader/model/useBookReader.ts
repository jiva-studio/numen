/**
 * Reader state and controls for an open book.
 */
import { computed, ref, shallowRef } from 'vue'
import type { ContentsEntry } from '@numen/ui'
import { formatErrorMessage } from '@numen/wire'
import type { Span } from '@/shared/span'
import type { MessageWriter } from '@/shared/notices/messages'
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
    await deps.bookDocument.draw(getDocumentAtOffset(deps.documents.value, offset.value))
  }

  const followLink = async (target: string) => {
    await deps.getShape()
    const targetDoc = deps.documents.value.find((one) => one.path === target)
    if (!targetDoc) return
    await goToOffset(targetDoc.span.from)
  }

  return { offset, goToOffset, followLink }
}

export function useBookReader(books: Books, path: string, options: BookReaderOptions) {
  const { words, writeMessage = () => {} } = options
  const title = ref('')
  const span = ref<Span>({ from: 0, to: 0 })
  const documents = shallowRef<readonly SpineDocument[]>([])
  const contents = shallowRef<readonly ContentsEntry[]>([])
  const pages = ref(0)
  const pageBytes = ref(0)
  // Stryker disable next-line StringLiteral: the fingerprint is assigned before the first document is drawn
  const fingerprint = ref('')
  const isLoading = ref(true)
  let isOpen = true

  const bookDocument = useBookDocument(books, path, {
    fingerprint,
    writeMessage,
    isOpen: () => isOpen,
  })

  let shape: Promise<void>
  const nav = useBookNavigation({
    documents,
    span,
    bookDocument,
    getShape: () => shape,
    isOpen: () => isOpen,
  })

  const page = computed(() => getPageNumber(pageBytes.value, pages.value, nav.offset.value))
  const chapter = computed(() => getChapterTitle(contents.value, nav.offset.value))

  const loadBook = async () => {
    try {
      const book = await books.getBook(path)
      if (!isOpen) return
      title.value = book.title
      span.value = book.span
      documents.value = book.documents
      pages.value = book.pages
      pageBytes.value = book.pageBytes
      contents.value = getContents(book, words)
      fingerprint.value = book.fingerprint
      nav.offset.value = book.span.from
      await bookDocument.draw(getDocumentAtOffset(book.documents, book.span.from))
    } catch (error) {
      if (!isOpen) return
      writeMessage(formatErrorMessage(error), 'error')
    } finally {
      isLoading.value = false
    }
  }

  shape = loadBook()

  const marks = useBookHighlights(nav.goToOffset, () => isOpen)
  const focusSpans = async (...spans: readonly Span[]) => {
    await shape
    await marks.focusSpans(...spans)
  }

  const close = () => {
    isOpen = false
    bookDocument.close()
    contents.value = []
    pages.value = 0
    pageBytes.value = 0
  }

  return {
    path,
    title,
    span,
    contents,
    pages,
    page,
    chapter,
    pageBytes,
    offset: nav.offset,
    isLoading,
    reading: bookDocument.reading,
    drawn: bookDocument.drawn,
    markup: bookDocument.markup,
    highlights: marks.highlights,
    otherHighlights: marks.otherHighlights,
    goToOffset: nav.goToOffset,
    followLink: nav.followLink,
    focusSpans,
    close,
  }
}
