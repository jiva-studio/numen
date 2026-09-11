/**
 * One document of a book with its pictures pointed at addresses this window
 * loads them from.
 */

/** Where one entry of the archive is served. */
export type EntryAddress = (name: string) => string

/** What a picture written into the markup itself begins with. */
const WRITTEN = 'data:'

export function resolveImageUrls(markup: string, getEntryUrl: EntryAddress): string {
  const held = new DOMParser().parseFromString(markup, 'text/html')
  for (const picture of held.body.querySelectorAll('img[src]')) {
    const named = picture.getAttribute('src') ?? ''
    if (named === '' || named.startsWith(WRITTEN)) continue
    picture.setAttribute('src', getEntryUrl(named))
  }
  return held.body.innerHTML
}
