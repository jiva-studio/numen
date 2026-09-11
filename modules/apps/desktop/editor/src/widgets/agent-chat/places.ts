/**
 * Places in the vault, as they are written into prose.
 *
 * An answer that names a passage of a book names it as a link, and this is the
 * shape of that link. The window reads the shape back and opens what it names,
 * so what the agent wrote is what the person presses.
 */

/** The scheme a link to somewhere in the vault carries. */
export const SCHEME = 'numen:'

/** Somewhere in the vault: a file, and the span of its text meant. */
export interface LinkTarget {
  readonly path: string
  readonly start: number
  readonly length: number
}

/**
 * The place a link names, and nothing for a link that names none.
 *
 * A link of ours with no span in it is a link to a file and not to a place
 * inside it, which nothing here opens.
 */
export const spotOf = (href: string): LinkTarget | null => {
  if (!href.startsWith(SCHEME)) return null
  const rest = href.slice(SCHEME.length)
  const [written, query = ''] = rest.split('?', 2)
  if (!written) return null

  let path: string
  try {
    path = decodeURIComponent(written)
  } catch {
    // An escape a model wrote wrongly names no file, and a link to no file
    // opens nothing.
    return null
  }
  if (path === '') return null

  const asked = new URLSearchParams(query)
  const start = Number(asked.get('start'))
  const length = Number(asked.get('length'))
  if (!Number.isSafeInteger(start) || !Number.isSafeInteger(length)) return null
  if (start < 0 || length <= 0) return null
  return { path, start, length }
}

/**
 * Every place a piece of prose links to, in the order they are written and each
 * of them once. It is read off the marks and not off the screen, so an answer
 * still arriving names what it has named so far.
 */
export const spotsIn = (text: string): readonly LinkTarget[] => {
  const found: LinkTarget[] = []
  for (const [, href] of text.matchAll(/]\(\s*(numen:[^\s)]+)\s*\)/g)) {
    const spot = spotOf(href ?? '')
    if (!spot) continue
    if (found.some((one) => same(one, spot))) continue
    found.push(spot)
  }
  return found
}

/** Whether two places are the same span of the same file. */
export const same = (one: LinkTarget, other: LinkTarget): boolean =>
  one.path === other.path && one.start === other.start && one.length === other.length
