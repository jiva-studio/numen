/**
 * Where a link inside a book points.
 *
 * The markup reaches the window with every href already resolved against the
 * archive, so what a link carries is an entry of the archive and a place inside
 * it. Both are relative to the window's own address, and the reader turns one
 * into a move within the book.
 */

/** Where a link inside a book points. */
export interface BookLink {
  /** The document, as the archive names it, and empty for the one being read. */
  readonly path: string
  /** The place inside that document, and empty for its beginning. */
  readonly fragment: string
}

/** Where an href points inside the book. */
export function placeIn(href: string): BookLink {
  const cut = href.indexOf('#')
  if (cut < 0) return { path: href, fragment: '' }
  return { path: href.slice(0, cut), fragment: href.slice(cut + 1) }
}
