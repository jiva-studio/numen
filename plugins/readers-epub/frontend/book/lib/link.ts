/**
 * Where a link inside a book points.
 *
 * The markup arrives in the window with every href already resolved against the
 * archive, so a link carries an archive entry and a position within it. Both
 * are relative to the window's own address, and the reader converts one into a
 * move within the book.
 */
import { isExternalLink, win } from '@numen/ui'

/** Where a link inside a book points. */
export interface BookLink {
  readonly path: string
  readonly fragment: string
}

/** Where an href points inside the book. */
export function parseBookLink(href: string): BookLink {
  const hashIndex = href.indexOf('#')
  if (hashIndex < 0) return { path: href, fragment: '' }
  return { path: href.slice(0, hashIndex), fragment: href.slice(hashIndex + 1) }
}

/** Whether an address names somewhere the window is not served from. */
export function isExternalHref(href: string): boolean {
  const here = new URL(win.location.href)
  try {
    return isExternalLink(new URL(href, here), here)
  } catch {
    // An href that is not a valid address is not external, and the book is
    // queried for the position it names.
    return false
  }
}
