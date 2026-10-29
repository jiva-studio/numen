/**
 * The values that make up the window appearance: the modes in which a theme's
 * tokens are read, the size ranges, and the percentage typed by the user.
 */
/** Which half of a `light-dark()` pair every token is read as. */
export type Mode = 'system' | 'light' | 'dark'

/** One theme as the list refers to it. */
export interface Theme {
  /** The origin and the filename, which together identify the theme when it is requested again. */
  readonly name: string
  /** What the file is called, without the origin. */
  readonly title: string
  /** Whether it ships inside the application. Every other theme is a user file. */
  readonly isBuiltIn: boolean
  /** It declares light and dark itself, so the mode has nothing left to choose. */
  readonly isPinned: boolean
}

/**
 * One value for each of the two sizes: the interface scale and the text scale.
 */
export interface Scales<T> {
  readonly interfaceScale: T
  readonly textScale: T
}

export interface Bounds {
  readonly min: number
  readonly max: number
}

/** The multiplier that draws everything the size it was designed at. */
export const DESIGNED = 1

/** How many sizes stand between one whole and the next. */
const STEPS = 10

/**
 * A number typed by the user, as a whole percentage. The row it produces is
 * titled with the typed digits, so it matches any typed text and is never
 * filtered out.
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

/** An empty range, used until the application provides the real one. */
export const EMPTY_RANGE: Bounds = { min: 0, max: 0 }

/** Whether a range contains a size. An empty range contains none. */
export const isInBounds = (range: Bounds, size: number): boolean =>
  range.max > range.min && range.min > 0 && size >= range.min && size <= range.max

/**
 * The sizes standing between the ends of a range: every step inside it, with
 * each end itself, so the end is offered wherever it falls.
 */
export const ladder = (range: Bounds): readonly number[] => {
  if (!isInBounds(range, range.min)) return []
  const rungs = [range.min]
  const last = Math.floor(range.max * STEPS)
  for (let step = 1; step <= last; step += 1) {
    const size = step / STEPS
    if (size > range.min && size < range.max) rungs.push(size)
  }
  return [...rungs, range.max]
}

/** The size those digits name, and nothing where what was typed is not digits. */
export const parseSize = (digits: string): number | null => {
  const matched = TYPED.exec(digits.trim())
  return matched ? Number(matched[1]) / 100 : null
}
