import { StateEffect } from '@codemirror/state'

/** How much of a change's text is shown, and which part arrived most recently. */
export interface Reveal {
  readonly visibleLength: number
  readonly fading: number
}

/**
 * How long one word is shown arriving for, in milliseconds. It is what the
 * fade takes, so a word has finished arriving as the next one starts.
 */
const PACE = 220

const findBoundaries = (text: string): number[] => {
  const words = /\S+\s*/g
  const ends: number[] = []
  for (let found = words.exec(text); found; found = words.exec(text)) {
    ends.push(found.index + found[0].length)
  }
  return ends
}

/**
 * How much of `text` is shown after `part` of the reveal has completed.
 *
 * Each word is one step, and the step after the last word shows all of the text
 * with none of it still arriving.
 */
export const createReveal = (text: string, part: number): Reveal => {
  const ends = findBoundaries(text)
  const steps = ends.length + 1
  const stepIndex = Math.max(0, Math.min(steps, Math.floor(part * steps)))
  if (stepIndex === 0) return { visibleLength: 0, fading: 0 }
  if (stepIndex === steps) return { visibleLength: text.length, fading: text.length }
  return {
    visibleLength: ends[stepIndex - 1] ?? text.length,
    fading: stepIndex > 1 ? (ends[stepIndex - 2] ?? 0) : 0,
  }
}

export const getDuration = (text: string) => (findBoundaries(text).length + 1) * PACE

/** How far the showing of the text has got. */
export const setReveal = StateEffect.define<Reveal>()
