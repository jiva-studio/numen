/** A line of a row split into the runs the words stand in, as plain values. */
import type { Span } from '@/shared/lib/span'

/** One run of a line, and whether it is why the item is here. */
export interface PalettePart {
  readonly text: string
  readonly hit: boolean
}

/**
 * A line split into the runs that are why the item is here and the runs that
 * are not. A line nothing stands in is one run.
 *
 * The spans are counted in UTF-16 code units, which is how the text arrives and
 * how a string is sliced. A boundary landing inside a character is moved off it,
 * outwards, so no run ends on half of one.
 */
export const partsOf = (text: string, at: readonly Span[] = []): readonly PalettePart[] => {
  const runs = merged(text, at)
  if (runs.length === 0) return text === '' ? [] : [{ text, hit: false }]

  const out: PalettePart[] = []
  let read = 0
  for (const run of runs) {
    if (run.from > read) out.push({ text: text.slice(read, run.from), hit: false })
    out.push({ text: text.slice(run.from, run.to), hit: true })
    read = run.to
  }
  if (read < text.length) out.push({ text: text.slice(read), hit: false })
  return out
}

/**
 * The spans as runs of this text: inside it, in order, none of them empty, and
 * no two of them touching.
 */
const merged = (text: string, at: readonly Span[]): Span[] => {
  const kept = at
    .map((span) => ({
      from: whole(text, bounded(text, Math.min(span.from, span.to)), -1),
      to: whole(text, bounded(text, Math.max(span.from, span.to)), 1),
    }))
    .filter((span) => span.from < span.to)
    .sort((one, other) => one.from - other.from)

  const out: Span[] = []
  for (const span of kept) {
    const last = out[out.length - 1]
    if (last && span.from <= last.to) {
      out[out.length - 1] = { from: last.from, to: Math.max(last.to, span.to) }
      continue
    }
    out.push(span)
  }
  return out
}

const bounded = (text: string, at: number): number =>
  Math.max(0, Math.min(Math.trunc(at) || 0, text.length))

/**
 * The nearest boundary that does not fall inside a character, from `at` in the
 * direction given. A character outside the basic plane is two code units, and a
 * span that names one of them names half a character.
 */
const whole = (text: string, at: number, by: 1 | -1): number => {
  let here = at
  while (here > 0 && here < text.length && inside(text, here)) here += by
  return here
}

const inside = (text: string, at: number): boolean =>
  isTrailing(text.charCodeAt(at)) && isLeading(text.charCodeAt(at - 1))

const isLeading = (unit: number): boolean => unit >= 0xd800 && unit <= 0xdbff
const isTrailing = (unit: number): boolean => unit >= 0xdc00 && unit <= 0xdfff
