/**
 * Where the ends of a track leave a value.
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
export const clamped = (value: number, bounds: Bounds): number =>
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
export const walks = (key: string): boolean => WALKING.includes(key)
