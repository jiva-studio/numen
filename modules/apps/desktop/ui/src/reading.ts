/**
 * One document as its tab reads it: the page in front, what is lit on it, and
 * the address that page is drawn at.
 *
 * Apart from the template the way `holding.ts` is: which page is shown, how
 * wide it is drawn, and what is lit over it are decisions, and a test asks them
 * without a browser.
 */
import type { Run } from './core'
import { computed, ref } from 'vue'

/** Where something sits on a page, in fractions of it. */
export interface Rect {
  readonly minX: number
  readonly minY: number
  readonly maxX: number
  readonly maxY: number
}

/** One page and what is lit on it. */
export interface Marked {
  readonly page: number
  readonly rects: readonly Rect[]
}

/** One page's size, in the page's own units. */
export interface Sheet {
  readonly wide: number
  readonly high: number
}

/**
 * What a document is: how many pages it has and how big each one is.
 *
 * A page has no name but where it stands in the file. That is the number the
 * viewer opens at and the number a location says, and one page with two numbers
 * is a person working out which is meant.
 *
 * The sizes are here because a strip of pages is laid out before any of them is
 * drawn. Built on one guessed shape it would move under the hand as the real
 * ones arrived.
 */
export interface Shape {
  readonly pages: number
  readonly sheets: readonly Sheet[]
}

/** Everything a document tab asks of the application. */
export interface Documents {
  /** How many pages the document has, and how big each one is. */
  shape(path: string): Promise<Shape>
  /**
   * Where one page is drawn `wide` device pixels across, as an address to point
   * a picture at.
   */
  page(path: string, at: number, wide: number): string
  /**
   * Where stretches of the document's own text stand on its pages, one answer
   * per stretch and in the order they were asked about. A stretch nothing was
   * recorded for stands nowhere.
   */
  marks(path: string, runs: readonly Run[]): Promise<readonly (readonly Marked[])[]>
}

/** The widest a page is drawn, in device pixels, which is as wide as one is drawn. */
const WIDEST = 4096

export type Reading = ReturnType<typeof reading>

export function reading(documents: Documents, path: string) {
  const pages = ref(0)
  /** How big each page is, in its own units. */
  const sheets = ref<readonly Sheet[]>([])
  /** Which page is in front, counted from the first. */
  const at = ref(0)
  /** How wide the page is drawn, in device pixels. */
  const wide = ref(0)
  /** What is lit, page by page: the place the tab turned to. */
  const marks = ref<readonly Marked[]>([])
  /**
   * The other places asked for, page by page. They are somewhere else to look
   * and not where the person was taken.
   */
  const others = ref<readonly (readonly Marked[])[]>([])
  /** What this document could not do, in words the window puts up for it. */
  const trouble = ref('')

  /**
   * What is lit on one page, in fractions of it. A rectangle is multiplied by
   * the page as it is drawn, so the zoom changes nothing here.
   */
  const litOn = (page: number): readonly Rect[] =>
    marks.value.find((one) => one.page === page)?.rects ?? []

  /** The other places on one page, each of them somewhere else to look. */
  const alsoOn = (page: number): readonly Rect[] =>
    others.value.flatMap((where) => where.find((one) => one.page === page)?.rects ?? [])

  /** What is lit on the page in front. */
  const lit = computed<readonly Rect[]>(() => litOn(at.value))

  /** The other places on the page in front. */
  const also = computed<readonly Rect[]>(() => alsoOn(at.value))

  /**
   * Where one page is drawn, at the width the strip wants it. It is empty until
   * the document has been read and the room it is read in has been measured.
   */
  const pictureOf = (page: number): string =>
    pages.value > 0 && wide.value > 0 ? documents.page(path, page, wide.value) : ''

  /** Where the page in front is drawn. */
  const picture = computed(() => pictureOf(at.value))

  /** Whether the tab this document stands in is still open. */
  let open = true

  /**
   * What the document is, asked for once. One that will not open says so, and
   * its tab stands with nothing in it.
   */
  const shape = (async () => {
    try {
      const said = await documents.shape(path)
      if (!open) return
      pages.value = said.pages
      sheets.value = said.sheets
    } catch (error) {
      if (!open) return
      trouble.value = String(error)
    }
  })()

  /** The page the person turned to. Past either end is the end. */
  const go = async (page: number) => {
    await shape
    if (!open || pages.value === 0) return
    at.value = Math.min(Math.max(Math.trunc(page), 0), pages.value - 1)
  }

  const next = () => go(at.value + 1)
  const back = () => go(at.value - 1)

  /** How wide the page is drawn, in device pixels. */
  const widen = (pixels: number) => {
    if (!open) return
    wide.value = Math.min(Math.max(Math.round(pixels), 0), WIDEST)
  }

  /**
   * Where the runs of the document's text sit. The first of them is the place
   * the person was sent to: the tab turns to its first page, and the rest are
   * lit where they fall.
   */
  const light = async (where: readonly (readonly Marked[])[]) => {
    const [front = [], ...rest] = where
    marks.value = front
    others.value = rest
    const first = front[0] ?? rest.flat()[0]
    if (first) await go(first.page)
  }

  /**
   * Stretches of the document's text reached: where they stand is asked for and
   * lit, and the tab turns to the first page of the first of them. A stretch
   * standing nowhere leaves the document on the page it is on with nothing lit.
   */
  const reach = async (...runs: readonly Run[]) => {
    await shape
    if (!open || runs.length === 0) return
    try {
      const where = await documents.marks(path, runs)
      if (!open) return
      await light(where)
    } catch (error) {
      if (!open) return
      trouble.value = String(error)
    }
  }

  /** The tab has closed: nothing is asked for again and nothing is drawn. */
  const close = () => {
    open = false
    pages.value = 0
    sheets.value = []
  }

  return {
    path,
    pages,
    sheets,
    at,
    picture,
    pictureOf,
    lit,
    litOn,
    also,
    alsoOn,
    trouble,
    go,
    next,
    back,
    widen,
    light,
    reach,
    close,
  }
}
