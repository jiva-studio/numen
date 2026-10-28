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
  // Stryker disable next-line StringLiteral: Missing query defaults to an empty query string.
  const [escapedPath, queryString = ''] = rest.split('?', 2)
  // Stryker disable next-line ConditionalExpression: Empty path check is also guarded by path length check.
  if (!escapedPath) return null

  let path: string
  try {
    path = decodeURIComponent(escapedPath)
  } catch {
    // An escape a model wrote wrongly names no file, and a link to no file opens nothing.
    return null
  }
  // Stryker disable next-line ConditionalExpression,EqualityOperator: Non-empty written cannot decode to empty path.
  if (path === '') return null

  const searchParams = new URLSearchParams(queryString)
  const start = Number(searchParams.get('start'))
  const length = Number(searchParams.get('length'))
  if (!Number.isSafeInteger(start) || !Number.isSafeInteger(length)) return null
  if (start < 0 || length <= 0) return null
  return { path, start, length }
}

const URI_SCHEME_PATTERN = /^[a-zA-Z][a-zA-Z0-9+.-]*:/

function extractRawVaultPath(href: string): string | null {
  if (!href || href.startsWith('#')) return null
  if (href.startsWith(LINK_SCHEME)) return href.slice(LINK_SCHEME.length)
  if (!URI_SCHEME_PATTERN.test(href)) return href
  return null
}

/**
 * Parses a vault file or note path from a `numen:` link without a span or a relative markdown link.
 */
export function parseVaultFileLink(href: string): string | null {
  const raw = extractRawVaultPath(href.trim())
  if (!raw) return null
  const [withoutHash = ''] = raw.split('#', 2)
  const [escapedPath = ''] = withoutHash.split('?', 2)
  if (!escapedPath) return null
  try {
    const decoded = decodeURIComponent(escapedPath)
      .trim()
      .replace(/^(\.\/+|\/+)/, '')
    return decoded || null
  } catch {
    return null
  }
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
