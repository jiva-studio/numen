/** A line of a row split into the runs the words stand in, as plain values. */
import { isLeading, isTrailing } from '@/shared/lib/surrogate'
import type { Span } from '@/shared/lib/span'

/** One run of a line, and whether it is why the item is here. */
export interface PalettePart {
  readonly text: string
  readonly isHit: boolean
}

/**
 * A line split into the runs that are why the item is here and the runs that
 *
 * The spans are counted in UTF-16 code units, which is how the text arrives and
 * how a string is sliced. A boundary landing inside a character is moved off it,
 * outwards, so no run ends on half of one.
 */
// Stryker disable next-line ArrayDeclaration: the mutated default is ["Stryker was here"], whose one entry has no span bounds and is dropped as empty.
export const getParts = (text: string, at: readonly Span[] = []): readonly PalettePart[] => {
  const runs = mergeSpans(text, at)
  const out: PalettePart[] = []
  let readItem = 0
  for (const run of runs) {
    if (run.from > readItem) out.push({ text: text.slice(readItem, run.from), isHit: false })
    out.push({ text: text.slice(run.from, run.to), isHit: true })
    readItem = run.to
  }
  if (readItem < text.length) out.push({ text: text.slice(readItem), isHit: false })
  return out
}

/**
 * The spans as runs of this text: inside it, in order, none of them empty, and
 * no two of them touching.
 */
const mergeSpans = (text: string, at: readonly Span[]): Span[] => {
  const retainedAt = at
    .map((span) => ({
      from: whole(text, clampToText(text, Math.min(span.from, span.to)), -1),
      to: whole(text, clampToText(text, Math.max(span.from, span.to)), 1),
    }))
    .filter((span) => span.from < span.to)
    .sort((one, other) => one.from - other.from)

  const out: Span[] = []
  for (const span of retainedAt) {
    const last = out[out.length - 1]
    if (last && span.from <= last.to) {
      out[out.length - 1] = { from: last.from, to: Math.max(last.to, span.to) }
      continue
    }
    out.push(span)
  }
  return out
}

const clampToText = (text: string, pos: number): number =>
  Math.max(0, Math.min(Math.trunc(pos) || 0, text.length))

/**
 * The nearest boundary that does not fall inside a character, from `at` in the
 * direction given. A character outside the basic plane is two code units, and a
 * span that names one of them names half a character.
 */
const whole = (text: string, pos: number, delta: 1 | -1): number => {
  let here = pos
  while (isInsideCharacter(text, here)) here += delta
  return here
}

const isInsideCharacter = (text: string, pos: number): boolean =>
  isTrailing(text.charCodeAt(pos)) && isLeading(text.charCodeAt(pos - 1))
