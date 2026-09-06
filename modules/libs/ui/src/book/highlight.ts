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

/** Where else the same search stands, drawn more faintly. */
export const ALSO = 'numen-book-also'

/** The ranges each book on screen has in each entry. */
const held = new Map<string, Map<object, readonly Range[]>>()

/** Whether this browser draws a range the page hands it. */
const draws = (): boolean =>
  typeof CSS !== 'undefined' && 'highlights' in CSS && typeof Highlight === 'function'

const redraw = (entry: string): void => {
  if (!draws()) return
  const all: Range[] = []
  for (const ranges of held.get(entry)?.values() ?? []) all.push(...ranges)
  if (all.length === 0) CSS.highlights.delete(entry)
  else CSS.highlights.set(entry, new Highlight(...all))
}

/** One book's runs in one entry, in place of whatever it had there before. */
export function highlight(entry: string, who: object, ranges: readonly Range[]): void {
  const books = held.get(entry) ?? new Map<object, readonly Range[]>()
  books.set(who, [...ranges])
  held.set(entry, books)
  redraw(entry)
}

/** One book's marked runs taken back, when it goes. */
export function unhighlight(who: object): void {
  for (const [entry, books] of held) {
    if (!books.delete(who)) continue
    redraw(entry)
  }
}
