/**
 * The runs of a book marked where they stand, drawn by the browser's own
 * highlight registry.
 *
 * The registry is keyed by a name and the name is styled once, so every book on
 * screen puts its ranges into the entry it asks for and takes them out again
 * when it goes.
 */

/** The entries, which `::highlight()` is written against. */
export const HIGHLIGHT = 'numen-book'

/** The other places named, drawn more faintly. */
export const OTHER_HIGHLIGHT = 'numen-book-other-highlight'

/** The ranges each book on screen has in each entry. */
const target = new Map<string, Map<object, readonly Range[]>>()

/** Whether this browser draws a range the page hands it. */
const canHighlight = (): boolean =>
  typeof CSS !== 'undefined' && 'highlights' in CSS && typeof Highlight === 'function'

const redraw = (entry: string, books: ReadonlyMap<object, readonly Range[]>): void => {
  if (!canHighlight()) return
  const all: Range[] = []
  for (const ranges of books.values()) all.push(...ranges)
  if (all.length === 0) CSS.highlights.delete(entry)
  else CSS.highlights.set(entry, new Highlight(...all))
}

/** One book's runs in one entry, in place of whatever it had there before. */
export function highlight(entry: string, who: object, ranges: readonly Range[]): void {
  const books = target.get(entry) ?? new Map<object, readonly Range[]>()
  books.set(who, [...ranges])
  target.set(entry, books)
  redraw(entry, books)
}

/** One book's marked runs taken back, when it goes. */
export function unhighlight(who: object): void {
  for (const [entry, books] of target) {
    if (!books.delete(who)) continue
    redraw(entry, books)
  }
}
