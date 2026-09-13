/**
 * Links in the vault, as they are written into prose.
 *
 * The address here is the one the core writes: `adapter/mcp` hands a passage
 * over already addressed, and names this module as what reads it back. Both
 * ends of that contract are findable from either side.
 */

/** The scheme a link to somewhere in the vault carries. */
export const LINK_SCHEME = 'numen:'

/** Somewhere in the vault: a file, and the span of its text meant. */
export interface LinkTarget {
  readonly path: string
  readonly start: number
  readonly length: number
}

/**
 * Parses a link href into a LinkTarget, or returns null if href is not a valid vault link.
 */
export function parseLinkTarget(href: string): LinkTarget | null {
  if (!href.startsWith(LINK_SCHEME)) return null
  const rest = href.slice(LINK_SCHEME.length)
  const [written, query = ''] = rest.split('?', 2)
  if (!written) return null

  let path: string
  try {
    path = decodeURIComponent(written)
  } catch {
    // An escape a model wrote wrongly names no file, and a link to no file opens nothing.
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
 * Whether two link targets name the same span of the same file.
 */
export function areLinkTargetsEqual(one: LinkTarget, other: LinkTarget): boolean {
  return one.path === other.path && one.start === other.start && one.length === other.length
}

/**
 * Extracts all unique link targets mentioned in markdown prose, preserving appearance order.
 */
export function extractLinkTargets(text: string): readonly LinkTarget[] {
  const found: LinkTarget[] = []
  for (const [, href] of text.matchAll(/]\(\s*(numen:[^\s)]+)\s*\)/g)) {
    const target = parseLinkTarget(href ?? '')
    if (!target) continue
    if (found.some((one) => areLinkTargetsEqual(one, target))) continue
    found.push(target)
  }
  return found
}
