/** A line of a row split into the runs the words stand in, as plain values. */
import { isLeading, isTrailing } from '@numen/ui'
import type { TextRange } from '@numen/ui'

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
export const getParts = (
  text: string,
  spans: readonly TextRange[] = [],
): readonly PalettePart[] => {
  const runs = mergeRanges(text, spans)
  const out: PalettePart[] = []
  let readItem = 0
  for (const run of runs) {
    if (run.start > readItem) out.push({ text: text.slice(readItem, run.start), isHit: false })
    out.push({ text: text.slice(run.start, run.end), isHit: true })
    readItem = run.end
  }
  if (readItem < text.length) out.push({ text: text.slice(readItem), isHit: false })
  return out
}

/**
 * The spans as runs of this text: inside it, in order, none of them empty, and
 * no two of them touching.
 */
const mergeRanges = (text: string, at: readonly TextRange[]): TextRange[] => {
  const retainedAt = at
    .map((span) => ({
      start: whole(text, clampToText(text, Math.min(span.start, span.end)), -1),
      end: whole(text, clampToText(text, Math.max(span.start, span.end)), 1),
    }))
    .filter((span) => span.start < span.end)
    .sort((one, other) => one.start - other.start)

  const out: TextRange[] = []
  for (const span of retainedAt) {
    const last = out[out.length - 1]
    if (last && span.start <= last.end) {
      out[out.length - 1] = { start: last.start, end: Math.max(last.end, span.end) }
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
