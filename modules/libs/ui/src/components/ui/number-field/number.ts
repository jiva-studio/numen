/**
 * What a line of typing comes to as a number, and where the bounds leave it.
 *
 * Pure: the same text gives the same answer wherever it is read, so what the
 * field hands on can be named without drawing it.
 */

/** How far a number may go, and what an arrow key moves it by. */
export interface Bounds {
  readonly min: number
  readonly max: number
  readonly step: number
}

export const DEFAULT_BOUNDS: Bounds = { min: 0, max: 999, step: 1 }

/** Text a number could still be typed out of: a lone sign, a trailing point. */
const ON_ITS_WAY = /^[+-]?(\d+([.,]\d*)?|[.,]\d*)?$/

/** Whether more typing could still make a number of the text. */
export const onItsWay = (typed: string): boolean => ON_ITS_WAY.test(typed.trim())

/**
 * The number the text stands for; nothing where it stands for none. The shape
 * is what a person types: no exponent, no hexadecimal, no word for infinity.
 */
export const numberOf = (typed: string): number | null => {
  const said = typed.trim()
  if (said === '' || !onItsWay(said)) return null
  const value = Number(said.replace(',', '.'))
  return Number.isFinite(value) ? value : null
}

/** The number brought inside the bounds. */
export const clamped = (value: number, bounds: Bounds): number =>
  Math.min(bounds.max, Math.max(bounds.min, value))

/** Whether the text is a number the bounds allow. An empty field is neither. */
export const allowed = (typed: string, bounds: Bounds): boolean => {
  const value = numberOf(typed)
  return value !== null && value === clamped(value, bounds)
}

/** Where an arrow key leaves the number: one step from where it stands. */
export const stepped = (value: number | null, by: number, bounds: Bounds): number =>
  clamped((value ?? bounds.min) + by * bounds.step, bounds)

/** How a number is written into the field. */
export const written = (value: number | null): string => (value === null ? '' : String(value))
