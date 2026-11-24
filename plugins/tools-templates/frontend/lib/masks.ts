const ANY_FOLDERS = '**'
const WILDCARDS = /[*?[]/
const DEFAULTS = ['Templates/**', '**/*.template.md']

/** A glob over vault paths: its segments, where a segment ** stands for any number of folders. */
export interface Mask {
  readonly segments: readonly string[]
}

const parseEntry = (entry: string): Mask | undefined => {
  const segments: string[] = []
  for (const segment of entry.trim().replaceAll('\\', '/').split('/')) {
    if (segment === '..') return undefined
    if (segment !== '' && segment !== '.') segments.push(segment)
  }
  return segments.length > 0 ? { segments } : undefined
}

/** The masks of the templates setting: entries separated by commas or lines, the defaults where none is left. */
export function parseMasks(raw: string): readonly Mask[] {
  const masks = raw
    .split(/[,\n]/)
    .map(parseEntry)
    .filter((mask): mask is Mask => mask !== undefined)
  if (masks.length > 0) return masks
  return DEFAULTS.flatMap((entry) => parseEntry(entry) ?? [])
}

const escapeLiteral = (text: string): string => text.replace(/[.+^${}()|\\\]]/g, '\\$&')

const toExpression = (segment: string): RegExp => {
  let source = ''
  for (let at = 0; at < segment.length; at += 1) {
    const char = segment.charAt(at)
    const close = char === '[' ? segment.indexOf(']', at + 2) : -1
    if (char === '*') source += '[^/]*'
    else if (char === '?') source += '[^/]'
    else if (close > 0) {
      const body = segment.slice(at + 1, close)
      const negated = body.startsWith('!') || body.startsWith('^')
      source += `[${negated ? '^' : ''}${escapeLiteral(negated ? body.slice(1) : body)}]`
      at = close
    } else source += escapeLiteral(char)
  }
  return new RegExp(`^${source}$`, 'u')
}

const matchSegments = (pattern: readonly string[], parts: readonly string[]): boolean => {
  const [head, ...rest] = pattern
  if (head === undefined) return parts.length === 0
  if (head === ANY_FOLDERS) {
    return Array.from({ length: parts.length + 1 }).some((_, skipped) =>
      matchSegments(rest, parts.slice(skipped)),
    )
  }
  const [part, ...others] = parts
  return part !== undefined && toExpression(head).test(part) && matchSegments(rest, others)
}

const getStaticFolder = (mask: Mask): readonly string[] => {
  const folders = mask.segments.slice(0, -1)
  const wildcard = folders.findIndex((segment) => WILDCARDS.test(segment))
  return wildcard < 0 ? folders : folders.slice(0, wildcard)
}

const getGroup = (mask: Mask, path: string): string => {
  const folder = path.split('/').slice(0, -1)
  return folder.slice(getStaticFolder(mask).length).join('/')
}

/** The group of a path under the first mask that names it, and undefined where none does. */
export function findGroup(masks: readonly Mask[], path: string): string | undefined {
  const parts = path.split('/')
  const found = masks.find((mask) => matchSegments(mask.segments, parts))
  return found ? getGroup(found, path) : undefined
}

const canHoldUnder = (pattern: readonly string[], folder: readonly string[]): boolean => {
  const [head, ...rest] = pattern
  const [part, ...others] = folder
  if (part === undefined) return true
  if (head === ANY_FOLDERS) return true
  if (head === undefined || rest.length === 0) return false
  return toExpression(head).test(part) && canHoldUnder(rest, others)
}

/** Whether a folder could hold a note some mask names, so that a search below it is worth making. */
export function canHold(masks: readonly Mask[], folder: string): boolean {
  const parts = folder.split('/').filter((part) => part !== '')
  return masks.some((mask) => canHoldUnder(mask.segments, parts))
}
