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

export function useBookReader(
  books: Books,
  path: string,
  words: BookWords,
  writeMessage: MessageWriter = () => {},
) {
  const title = ref('')
  const span = ref<Span>({ from: 0, to: 0 })
  const documents = shallowRef<readonly SpineDocument[]>([])
  const contents = shallowRef<readonly ContentsEntry[]>([])
  const pages = ref(0)
  const pageBytes = ref(0)
  const offset = ref(0)
  const fingerprint = ref('')

  /**
   * Whether the book and its first document are still on their way. No text is
   * not the same as no text yet, and a blank page says the wrong one of the two.
   */
  const isLoading = ref(true)
  let open = true

  const bookDocument = useBookDocument(books, path, fingerprint, writeMessage, () => open)

  const page = computed(() => getPageNumber(pageBytes.value, pages.value, offset.value))

  const chapter = computed(() => {
    let found = ''
    for (const entry of contents.value) {
      if (entry.at > offset.value) break
      found = entry.title
    }
    return found
  })

  const loadBook = async () => {
    try {
      const book = await books.getBook(path)
      if (!open) return
      title.value = book.title
      span.value = book.span
      documents.value = book.documents
      pages.value = book.pages
      pageBytes.value = book.pageBytes
      contents.value = getContents(book, words)
      fingerprint.value = book.fingerprint
      offset.value = book.span.from
      await bookDocument.draw(getDocumentAtOffset(book.documents, book.span.from))
    } catch (error) {
      if (!open) return
      writeMessage(formatErrorMessage(error), 'error')
    } finally {
      isLoading.value = false
    }
  }

  const shape = loadBook()

  const goToOffset = async (targetOffset: number) => {
    await shape
    if (!open || documents.value.length === 0) return
    const last = Math.max(span.value.to - 1, span.value.from)
    offset.value = Math.min(Math.max(Math.trunc(targetOffset), span.value.from), last)
    await bookDocument.draw(getDocumentAtOffset(documents.value, offset.value))
  }

  const followLink = async (target: string) => {
    await shape
    if (!open) return
    const targetDoc = documents.value.find((one) => one.path === target)
    if (!targetDoc) return
    await goToOffset(targetDoc.span.from)
  }

  const marks = useBookHighlights(goToOffset, () => open)

  const focusSpans = async (...spans: readonly Span[]) => {
    await shape
    if (!open) return
    await marks.focusSpans(...spans)
  }

  const close = () => {
    open = false
    bookDocument.close()
    documents.value = []
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
    offset,
    isLoading,
    reading: bookDocument.reading,
    drawn: bookDocument.drawn,
    markup: bookDocument.markup,
    highlights: marks.highlights,
    otherHighlights: marks.otherHighlights,
    goToOffset,
    followLink,
    focusSpans,
    close,
  }
}
