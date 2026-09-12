/**
 * Where the ends of a track leave a value, and where a key leaves the handle.
 *
 * Pure: the same value gives the same answer wherever it is read, so what the
 * handle stands at can be named without drawing it.
 */

/** How far the track runs, and what one step of it moves. */
export interface Bounds {
  readonly min: number
  readonly max: number
  readonly step: number
}

/** The value brought inside the ends. */
export const clamp = (value: number, bounds: Bounds): number =>
  Math.min(bounds.max, Math.max(bounds.min, value))

/** The keys that walk the handle along the track. */
const WALKING: readonly string[] = [
  'ArrowUp',
  'ArrowDown',
  'ArrowLeft',
  'ArrowRight',
  'PageUp',
  'PageDown',
  'Home',
  'End',
]

/** Whether a key is one the handle walks under. */
export const isWalkingKey = (key: string): boolean => WALKING.includes(key)

/** How many steps a page key covers, which is what a key held with shift covers. */
const PACES = 10

/** The places a number is written to. */
const places = (value: number): number => {
  const said = `${value}`
  const point = said.indexOf('.')
  return point < 0 ? 0 : said.length - point - 1
}

/** A value written to the places the floor and the step are written to. */
const roundToPlaces = (value: number, bounds: Bounds): number => {
  const scale = 10 ** Math.max(places(bounds.min), places(bounds.step))
  return Math.round(value * scale) / scale
}

/**
 * How many steps a value stands above the floor. A value one step short of a
 * whole one by the width of a rounding error stands on that whole one.
 */
const above = (value: number, bounds: Bounds): number => {
  const steps = (value - bounds.min) / bounds.step
  const whole = Math.round(steps)
  return Math.abs(steps - whole) < 1e-9 ? whole : steps
}

/**
 * Where a walk of so many steps leaves a value: the place the step lays that
 * many along from it, and the end of the track past the last of them. A value
 * between two places is drawn onto the one the walk is heading towards, so a
 * step out and a step back come to where they began.
 */
export const stepBy = (value: number, by: number, bounds: Bounds): number => {
  const from = clamp(value, bounds)
  if (bounds.step <= 0 || by === 0) return from
  const at = above(from, bounds)
  const place = (by > 0 ? Math.floor(at) : Math.ceil(at)) + by
  return clamp(roundToPlaces(bounds.min + place * bounds.step, bounds), bounds)
}

/** Where a key leaves the handle, and nothing for a key it does not answer. */
export const stepForKey = (
  key: string,
  value: number,
  bounds: Bounds,
  far: boolean,
): number | null => {
  const paces = far ? PACES : 1
  if (key === 'Home') return bounds.min
  if (key === 'End') return bounds.max
  if (key === 'ArrowRight' || key === 'ArrowUp') return stepBy(value, paces, bounds)
  if (key === 'ArrowLeft' || key === 'ArrowDown') return stepBy(value, -paces, bounds)
  if (key === 'PageUp') return stepBy(value, PACES, bounds)
  if (key === 'PageDown') return stepBy(value, -PACES, bounds)
  return null
}
