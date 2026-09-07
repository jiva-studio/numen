/**
 * One book as its tab reads it: where the person is standing in its text, the
 * document that offset falls in, and what the book divides into.
 *
 * A place in a book is a byte offset into its one text stream. How many spreads
 * the book comes to is the width of the window and the size of the text, so a
 * spread is no address: every offset here was counted where the bytes are, and
 * none of them is counted from a string.
 *
 * Apart from the template the way `document/open.ts` is: which document is read,
 * which page an offset falls on and what the contents list holds are decisions,
 * and a test asks them without a browser.
 */
import type { BookSpan, ContentsEntry } from '@numen/ui'
import type { Stretch } from '../core'
import { computed, ref, shallowRef } from 'vue'
import { troubleWords } from '@numen/wire'
import { pointedAt } from './markup'

/** One document of a book's spine, and where it stands in the book's text. */
export interface SpineDocument {
  /** What the book calls the document inside itself. */
  readonly path: string
  /** Where its text stands in the book's text, in bytes. */
  readonly span: BookSpan
}

/** One place a book names, and where it begins. */
export interface BookPart {
  readonly title: string
  /** Where the named text begins, in bytes of the book's text. */
  readonly at: number
  /** How deep it sits, and zero for a place the book's own navigation named. */
  readonly level: number
}

/** One page of the printed book this file was made from. */
export interface PrintedPage {
  /** What the printed book calls the page. */
  readonly label: string
  /** Where it begins, in bytes of the book's text. */
  readonly at: number
}

/**
 * What a book is: the documents it is drawn from, the places it names, and the
 * pages it is read in.
 */
export interface Book {
  /** What the book calls itself. */
  readonly title: string
  /** Where the book's text runs between, in bytes of it. */
  readonly span: BookSpan
  /** The documents of the spine, in reading order. */
  readonly documents: readonly SpineDocument[]
  /** What the book names, ascending by offset. */
  readonly parts: readonly BookPart[]
  /** The pages of the printed book it was made from. Most books carry none. */
  readonly printed: readonly PrintedPage[]
  /** How many pages the book is read in, counted over its text. */
  readonly pages: number
  /** How many bytes of that text stand on one page, in the book's own script. */
  readonly pageBytes: number
  /** The file these were read from, as the one string the window carries. */
  readonly at: string
}

/** Everything a book tab asks of the application. */
export interface Books {
  /** What the book is: its documents, the places it names, and its pages. */
  shape(path: string): Promise<Book>
  /**
   * One document of the spine as it is drawn: markup in which every run of text
   * carries the byte offset it begins at in the book's text, and every picture
   * is named as the archive names it.
   *
   * The file the shape came out of is named in the ask, so the markup is one
   * drawing of one book and is answered with that or with nothing.
   */
  markup(path: string, document: string, seen: string): Promise<string>
  /**
   * Where one entry of the book's archive is served, as an address to point a
   * picture at. It names the same file the markup was asked for.
   */
  entry(path: string, name: string, seen: string): string
}

/** What a book tab says of a book that names nothing. */
export interface BookWords {
  /** What a page of the printed book is called, before what it is labelled. */
  readonly page: string
}

/** Where a stretch of a source's text runs, in bytes of it. */
const spanOf = (stretch: Stretch): BookSpan => ({
  begins: stretch.start,
  ends: stretch.start + stretch.length,
})

/**
 * The page an offset falls on, counted from one, and none for a book with no
 * pages. A page is a run of the text apiece, so the page is the offset over the
 * size of one, and the size is the book's own.
 */
export function pageAt(pageBytes: number, pages: number, at: number): number {
  if (pages <= 0 || pageBytes <= 0) return 0
  return Math.min(Math.max(Math.floor(at / pageBytes) + 1, 1), pages)
}

/**
 * The document an offset falls in: the last one beginning at or before it, and
 * none where the book has no document there. The documents are in reading
 * order.
 */
export function documentAt(
  documents: readonly SpineDocument[],
  at: number,
): SpineDocument | undefined {
  let found: SpineDocument | undefined
  for (const one of documents) {
    if (one.span.begins > at) break
    found = one
  }
  return found
}

/** What a document of the spine is called, which is its own name in the book. */
const namedIn = (document: SpineDocument): string => {
  const last = document.path.split('/').pop() ?? document.path
  const dot = last.lastIndexOf('.')
  return dot > 0 ? last.slice(0, dot) : last
}

/**
 * What a book is reached by. Half of the corpus this reads names nothing at
 * all, and a book of five thousand pages with nothing to jump by is one nobody
 * finds their way back into, so a book that names no place is reached by the
 * pages it was printed on, and one printed on none by the documents it is read
 * in.
 */
export function contentsOf(book: Book, words: BookWords): readonly ContentsEntry[] {
  if (book.parts.length !== 0) {
    return book.parts.map((one) => ({ title: one.title, at: one.at, level: one.level }))
  }
  if (book.printed.length !== 0) {
    return book.printed.map((one) => ({ title: `${words.page} ${one.label}`, at: one.at, level: 0 }))
  }
  return book.documents.map((one) => ({ title: namedIn(one), at: one.span.begins, level: 0 }))
}

/** What one open book holds: where the person is, and what stands there. */
export type OpenBookState = ReturnType<typeof openBook>

export function openBook(books: Books, path: string, words: BookWords) {
  /** What the book calls itself. */
  const title = ref('')
  /** Where the book's text runs between, in bytes of it. */
  const span = shallowRef<BookSpan>({ begins: 0, ends: 0 })
  /** The documents of the spine, in reading order. */
  const documents = shallowRef<readonly SpineDocument[]>([])
  /** What the book is reached by: its places, its printed pages, or its documents. */
  const contents = shallowRef<readonly ContentsEntry[]>([])
  /** How many pages the book is read in, and how many bytes stand on one. */
  const pages = ref(0)
  const pageBytes = ref(0)
  /** Where the person is reading, in bytes of the book's text. */
  const at = ref(0)
  /** The document the person is reading, and none until the book has opened. */
  const standing = shallowRef<SpineDocument | null>(null)
  /** That document as it is drawn. */
  const markup = ref('')
  /** The file the book was read from, which the addresses it is drawn at name. */
  const seen = ref('')
  /** The runs marked where they stand: the place the tab was sent to. */
  const marked = shallowRef<readonly BookSpan[]>([])
  /**
   * The other runs asked about. They are somewhere else to look and not where
   * the person was taken.
   */
  const also = shallowRef<readonly BookSpan[]>([])
  /** What this book could not do, in words the window puts up for it. */
  const trouble = ref('')

  /** The page the person is on, counted from one. */
  const page = computed(() => pageAt(pageBytes.value, pages.value, at.value))

  /**
   * What the book calls the place in front: the last thing it names at or
   * before the offset, and nothing before the first of them.
   */
  const chapter = computed(() => {
    let found = ''
    for (const entry of contents.value) {
      if (entry.at > at.value) break
      found = entry.title
    }
    return found
  })

  /** Where the document being read stands in the book's text. */
  const reading = computed<BookSpan>(() => standing.value?.span ?? { begins: 0, ends: 0 })

  /** The document being read, as the book's archive names it. */
  const drawn = computed(() => standing.value?.path ?? '')

  /** Whether the tab this book stands in is still open. */
  let open = true

  /**
   * Which drawing of a document is wanted. A document asked for while another is
   * on its way is the one that is drawn.
   */
  let wanted = 0

  /** One document of the spine put on screen, its pictures at their addresses. */
  const draw = async (document: SpineDocument | undefined) => {
    if (!document || document.path === standing.value?.path) return
    const asked = ++wanted
    try {
      const drawn = await books.markup(path, document.path, seen.value)
      if (!open || asked !== wanted) return
      standing.value = document
      markup.value = pointedAt(drawn, (name) => books.entry(path, name, seen.value))
    } catch (error) {
      if (!open || asked !== wanted) return
      trouble.value = troubleWords(error)
    }
  }

  /**
   * What the book is, asked for once. One that will not open says so, and its
   * tab stands with nothing in it.
   */
  const shape = (async () => {
    try {
      const said = await books.shape(path)
      if (!open) return
      title.value = said.title
      span.value = said.span
      documents.value = said.documents
      pages.value = said.pages
      pageBytes.value = said.pageBytes
      contents.value = contentsOf(said, words)
      seen.value = said.at
      at.value = said.span.begins
      await draw(documentAt(said.documents, said.span.begins))
    } catch (error) {
      if (!open) return
      trouble.value = troubleWords(error)
    }
  })()

  /**
   * The offset the person went to. Past either end of the book is the end, and
   * an offset in another document of the spine opens that document.
   */
  const go = async (offset: number) => {
    await shape
    if (!open || documents.value.length === 0) return
    const last = Math.max(span.value.ends - 1, span.value.begins)
    at.value = Math.min(Math.max(Math.trunc(offset), span.value.begins), last)
    await draw(documentAt(documents.value, at.value))
  }

  /**
   * A link inside the book followed to another of its documents. That document
   * is drawn from its beginning, and a link naming a document the book does not
   * hold leads nowhere.
   */
  const follow = async (target: string) => {
    await shape
    if (!open) return
    const wanted = documents.value.find((one) => one.path === target)
    if (!wanted) return
    await go(wanted.span.begins)
  }

  /**
   * Stretches of the book's text reached: the person is sent to the first of
   * them and it is marked where it stands, and the rest are marked more faintly
   * wherever they fall.
   */
  const reach = async (...stretches: readonly Stretch[]) => {
    await shape
    if (!open || stretches.length === 0) return
    const [front, ...rest] = stretches
    if (!front) return
    marked.value = [spanOf(front)]
    also.value = rest.map(spanOf)
    await go(front.start)
  }

  /** The tab has closed: nothing is asked for again and nothing is drawn. */
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
    marked,
    also,
    trouble,
    go,
    follow,
    reach,
    close,
  }
}
