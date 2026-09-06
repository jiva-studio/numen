/**
 * One document of a book with its pictures pointed at addresses this window
 * loads them from.
 *
 * A picture in a book is an entry of the archive the book was read out of, named
 * as the archive names it, and the window is drawn from a scheme a browser loads
 * nothing through. A picture already written into the markup is left where it
 * stands.
 */

/** Where one entry of the archive is served. */
export type EntryAddress = (name: string) => string

/** What a picture written into the markup itself begins with. */
const WRITTEN = 'data:'

export function pointedAt(markup: string, entry: EntryAddress): string {
  const held = new DOMParser().parseFromString(markup, 'text/html')
  for (const picture of held.body.querySelectorAll('img[src]')) {
    const named = picture.getAttribute('src') ?? ''
    if (named === '' || named.startsWith(WRITTEN)) continue
    picture.setAttribute('src', entry(named))
  }
  return held.body.innerHTML
}
