/**
 * Reader state and controls for an open book.
 */
import { computed, ref, shallowRef } from 'vue'
import type { BookSpan, ContentsEntry } from '@numen/ui'
import { formatErrorMessage } from '@numen/wire'
import type { Span } from '../../../shared/core'
import type { MessageWriter } from '../../../shared/notices/messages'
import { resolveImageUrls } from '../markup'
import { getContents, getDocumentAtOffset, getPageNumber } from '../pagination'
import type {
  Book,
  BookHandle,
  BookPart,
  Books,
  BookWords,
  PrintedPage,
  SpineDocument,
} from '../types'

export type {
  Book,
  BookHandle,
  BookPart,
  Books,
  BookWords,
  PrintedPage,
  SpineDocument,
}
export { getContents, getDocumentAtOffset, getPageNumber } from '../pagination'

const spanOf = (span: Span): BookSpan => ({
  begins: span.from,
  ends: span.to,
})

export type BookReaderState = ReturnType<typeof useBookReader>

export function useBookReader(
  books: Books,
  path: string,
  words: BookWords,
  said: MessageWriter = () => {},
) {
  const title = ref('')
  const span = ref<BookSpan>({ begins: 0, ends: 0 })
  const documents = shallowRef<readonly SpineDocument[]>([])
  const contents = shallowRef<readonly ContentsEntry[]>([])
  const pages = ref(0)
  const pageBytes = ref(0)
  const offset = ref(0)
  const fingerprint = ref('')
  const markup = ref('')
  const activeDocument = shallowRef<SpineDocument | undefined>(undefined)
  const highlights = shallowRef<readonly BookSpan[]>([])
  const elsewhere = shallowRef<readonly BookSpan[]>([])

  const page = computed(() => getPageNumber(pageBytes.value, pages.value, offset.value))

  const chapter = computed(() => {
    let found = ''
    for (const entry of contents.value) {
      if (entry.at > offset.value) break
      found = entry.title
    }
    return found
  })

  const reading = computed<BookSpan>(() => activeDocument.value?.span ?? { begins: 0, ends: 0 })
  const drawn = computed(() => activeDocument.value?.path ?? '')
  let open = true
  let wanted = 0

  const draw = async (document: SpineDocument | undefined) => {
    if (!document || document.path === activeDocument.value?.path) return
    const asked = ++wanted
    try {
      const markupContent = await books.readMarkup(path, document.path, fingerprint.value)
      if (!open || asked !== wanted) return
      activeDocument.value = document
      markup.value = resolveImageUrls(markupContent, (name) => books.getEntryUrl(path, name, fingerprint.value))
    } catch (error) {
      if (!open || asked !== wanted) return
      said(formatErrorMessage(error), 'error')
    }
  }

  const loadBook = async () => {
    try {
      const saidBook = await books.getBook(path)
      if (!open) return
      title.value = saidBook.title
      span.value = saidBook.span
      documents.value = saidBook.documents
      pages.value = saidBook.pages
      pageBytes.value = saidBook.pageBytes
      contents.value = getContents(saidBook, words)
      fingerprint.value = saidBook.fingerprint
      offset.value = saidBook.span.begins
      await draw(getDocumentAtOffset(saidBook.documents, saidBook.span.begins))
    } catch (error) {
      if (!open) return
      said(formatErrorMessage(error), 'error')
    }
  }

  const shape = loadBook()

  const goToOffset = async (targetOffset: number) => {
    await shape
    if (!open || documents.value.length === 0) return
    const last = Math.max(span.value.ends - 1, span.value.begins)
    offset.value = Math.min(Math.max(Math.trunc(targetOffset), span.value.begins), last)
    await draw(getDocumentAtOffset(documents.value, offset.value))
  }

  const followLink = async (target: string) => {
    await shape
    if (!open) return
    const targetDoc = documents.value.find((one) => one.path === target)
    if (!targetDoc) return
    await goToOffset(targetDoc.span.begins)
  }

  const focusSpans = async (...spans: readonly Span[]) => {
    await shape
    if (!open || spans.length === 0) return
    const [front, ...rest] = spans
    if (!front) return
    highlights.value = [spanOf(front)]
    elsewhere.value = rest.map(spanOf)
    await goToOffset(front.from)
  }

  const close = () => {
    open = false
    markup.value = ''
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
    reading,
    drawn,
    markup,
    highlights,
    elsewhere,
    goToOffset,
    followLink,
    focusSpans,
    close,
  }
}
