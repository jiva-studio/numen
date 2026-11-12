/** The two sizes the window is drawn at, and how long the keyboard rests on one before it is applied. */
import type { Scales } from './appearance'

/** The names the two sizes are declared under. */
export const INTERFACE_SCALE = 'interfaceScale'
export const TEXT_SCALE = 'textScale'

/** Which of the two sizes. */
export type ScaleKind = typeof INTERFACE_SCALE | typeof TEXT_SCALE

/**
 * How long the keyboard must rest on a size before the window is rendered at
 * it. A held arrow key crosses a row every 40 milliseconds, and applying a row
 * re-lays out every open document and empties its pages.
 */
export const HELD = 150

/** Returns the pair with the value of one of the two sizes replaced. */
export const onto = <T>(both: Scales<T>, which: ScaleKind, one: T): Scales<T> => ({
  ...both,
  [which]: one,
})
