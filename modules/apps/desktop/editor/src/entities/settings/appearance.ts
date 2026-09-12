/**
 * The values the window's appearance is made of: the modes a theme's tokens are
 * read as, the sizes a range reaches, and the number a person typed.
 */
import type { Bounds, Mode } from './theme'

/** The multiplier that draws everything the size it was designed at. */
export const DESIGNED = 1

/** How many sizes stand between one whole and the next. */
const STEPS = 10

/**
 * A number a person typed, which is a whole number of percent. The row it makes
 * is titled in the digits that were typed, so the words typed always leave it
 * standing.
 */
const TYPED = /^(\d+)\s*%?$/

/** The modes, in the order they are offered. */
export const MODES: readonly Mode[] = ['system', 'light', 'dark']

/** What `color-scheme` is written as for each mode. */
export const SCHEMES: Record<Mode, string> = {
  system: 'light dark',
  light: 'light',
  dark: 'dark',
}

/** A range until the application has said what one is. */
export const NOWHERE: Bounds = { least: 0, most: 0 }

/** Whether a range reaches a size. A range holding nothing reaches none. */
export const reaches = (range: Bounds, size: number): boolean =>
  range.most > range.least && range.least > 0 && size >= range.least && size <= range.most

/**
 * The sizes standing between the ends of a range: every step inside it, with
 * each end itself, so the end is offered wherever it falls.
 */
export const ladder = (range: Bounds): readonly number[] => {
  if (!reaches(range, range.least)) return []
  const rungs = [range.least]
  const first = Math.ceil(range.least * STEPS)
  const last = Math.floor(range.most * STEPS)
  for (let step = first; step <= last; step += 1) {
    const size = step / STEPS
    if (size > range.least && size < range.most) rungs.push(size)
  }
  return [...rungs, range.most]
}

/** The size those digits name, and nothing where what was typed is not digits. */
export const typedSize = (typed: string): number | null => {
  const said = TYPED.exec(typed.trim())
  return said ? Number(said[1]) / 100 : null
}
