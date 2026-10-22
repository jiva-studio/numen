/**
 * How far a number may go, and what one step of it moves.
 *
 * Pure: the same value gives the same answer wherever it is read.
 */

/** The floor and the ceiling a number is held between. */
export interface Ends {
  readonly min: number
  readonly max: number
}

/** The ends and the step that lays places along it, counted from the floor. */
export interface Bounds extends Ends {
  readonly step: number
}

/** The value brought inside the ends; a value past both ends stands at the ceiling. */
export const clamp = (value: number, ends: Ends): number =>
  Math.min(ends.max, Math.max(ends.min, value))

/** The places after the point a number is written to. */
export const getDecimalPlaces = (value: number): number => {
  const str = `${value}`
  const point = str.indexOf('.')
  return point < 0 ? 0 : str.length - point - 1
}

/** The value rounded to so many places after the point. */
export const roundToPlaces = (value: number, places: number): number => {
  const scale = 10 ** places
  return Math.round(value * scale) / scale
}

/**
 * How many steps a value stands above the floor. A value short of a whole step
 * by the width of a rounding error relies on that whole one.
 */
const getStepsAbove = (value: number, bounds: Bounds): number => {
  const steps = (value - bounds.min) / bounds.step
  const whole = Math.round(steps)
  return Math.abs(steps - whole) < 1e-9 ? whole : steps
}

/**
 * Where a walk of so many steps leaves a value: the place the step lays that
 * many along from it, and the end past the last of them. A value between two
 * places is drawn onto the one the walk is heading towards, so a step out and a
 * step back come to where they began. The result is written to as many places
 * as the floor and the step are.
 */
export const stepBy = (value: number, by: number, bounds: Bounds): number => {
  const from = clamp(value, bounds)
  if (bounds.step <= 0 || by === 0) return from
  const at = getStepsAbove(from, bounds)
  const place = (by > 0 ? Math.floor(at) : Math.ceil(at)) + by
  const places = Math.max(getDecimalPlaces(bounds.min), getDecimalPlaces(bounds.step))
  return clamp(roundToPlaces(bounds.min + place * bounds.step, places), bounds)
}
