/**
 * The runs of a book marked where they stand, drawn by the browser's own
 * highlight registry.
 *
 * The registry is keyed by a name and the name is styled once, so every book on
 * screen puts its ranges into the one entry and takes them out again when it
 * goes.
 */

/** What the entry is called, which `::highlight()` is written against. */
export const HIGHLIGHT = 'numen-book'

/** The ranges each book on screen has marked. */
const held = new Map<object, readonly Range[]>()

/** Whether this browser draws a range the page hands it. */
const draws = (): boolean =>
  typeof CSS !== 'undefined' && 'highlights' in CSS && typeof Highlight === 'function'

const redraw = (): void => {
  if (!draws()) return
  const all: Range[] = []
  for (const ranges of held.values()) all.push(...ranges)
  if (all.length === 0) CSS.highlights.delete(HIGHLIGHT)
  else CSS.highlights.set(HIGHLIGHT, new Highlight(...all))
}

/** One book's marked runs, in place of whatever it had marked before. */
export function highlight(who: object, ranges: readonly Range[]): void {
  held.set(who, [...ranges])
  redraw()
}

/** One book's marked runs taken back, when it goes. */
export function unhighlight(who: object): void {
  held.delete(who)
  redraw()
}
