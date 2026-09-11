/**
 * Reader state and controls for an open book.
 */
import { computed, ref, shallowRef } from 'vue'
import type { BookSpan, ContentsEntry } from '@numen/ui'
import { formatErrorMessage } from '@numen/wire'
import type { Span } from '../../shared/core'
import type { MessageWriter } from '../../shared/notices/messages'
import { pointedAt } from './markup'
import { contentsOf, documentAt, pageAt } from './pagination'
import type {
  Book,
  BookHandle,
  BookPart,
  Books,
  BookWords,
  PrintedPage,
  SpineDocument,
} from './types'

export type {
  Book,
  BookHandle,
  BookPart,
  Books,
  BookWords,
  PrintedPage,
  SpineDocument,
}
export { contentsOf, documentAt, pageAt } from './pagination'

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
  const at = ref(0)
  const seen = ref('')
  const markup = ref('')
  const standing = shallowRef<SpineDocument | undefined>(undefined)
  const highlights = shallowRef<readonly BookSpan[]>([])
  const elsewhere = shallowRef<readonly BookSpan[]>([])

  const page = computed(() => pageAt(pageBytes.value, pages.value, at.value))

  const chapter = computed(() => {
    let found = ''
    for (const entry of contents.value) {
      if (entry.at > at.value) break
      found = entry.title
    }
    return found
  })

  const reading = computed<BookSpan>(() => standing.value?.span ?? { begins: 0, ends: 0 })
  const drawn = computed(() => standing.value?.path ?? '')
  let open = true
  let wanted = 0

  const draw = async (document: SpineDocument | undefined) => {
    if (!document || document.path === standing.value?.path) return
    const asked = ++wanted
    try {
      const markupContent = await books.readMarkup(path, document.path, seen.value)
      if (!open || asked !== wanted) return
      standing.value = document
      markup.value = pointedAt(markupContent, (name) => books.getEntryUrl(path, name, seen.value))
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
      contents.value = contentsOf(saidBook, words)
      seen.value = saidBook.fingerprint
      at.value = saidBook.span.begins
      await draw(documentAt(saidBook.documents, saidBook.span.begins))
    } catch (error) {
      if (!open) return
      said(formatErrorMessage(error), 'error')
    }
  }

  const shape = loadBook()

  const go = async (offset: number) => {
    await shape
    if (!open || documents.value.length === 0) return
    const last = Math.max(span.value.ends - 1, span.value.begins)
    at.value = Math.min(Math.max(Math.trunc(offset), span.value.begins), last)
    await draw(documentAt(documents.value, at.value))
  }

  const follow = async (target: string) => {
    await shape
    if (!open) return
    const targetDoc = documents.value.find((one) => one.path === target)
    if (!targetDoc) return
    await go(targetDoc.span.begins)
  }

  const reach = async (...spans: readonly Span[]) => {
    await shape
    if (!open || spans.length === 0) return
    const [front, ...rest] = spans
    if (!front) return
    highlights.value = [spanOf(front)]
    elsewhere.value = rest.map(spanOf)
    await go(front.from)
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
    at,
    reading,
    drawn,
    markup,
    highlights,
    elsewhere,
    go,
    follow,
    reach,
    close,
  }
}
