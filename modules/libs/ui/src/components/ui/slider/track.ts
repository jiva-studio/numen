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
