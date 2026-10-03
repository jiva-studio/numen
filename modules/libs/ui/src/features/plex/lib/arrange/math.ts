import type { Extent } from '../frame'

export const lerp = (from: number, to: number, t: number): number => from + (to - from) * t

/**
 * A fraction of the way through, and never anything else.
 *
 * Not a number is caught here. It reaches a transform as `opacity="NaN"` and
 * the node is simply not drawn, with nothing to say why.
 */
export const clamp01 = (value: number): number => {
  if (Number.isNaN(value)) return 1
  return Math.min(Math.max(value, 0), 1)
}

export function lerpExtent(from: Extent, to: Extent, t: number): Extent {
  return {
    minX: lerp(from.minX, to.minX, t),
    minY: lerp(from.minY, to.minY, t),
    maxX: lerp(from.maxX, to.maxX, t),
    maxY: lerp(from.maxY, to.maxY, t),
  }
}

/** Fast to leave, slow to settle: the useful part of a move is the end. */
export const easeOut = (t: number): number => 1 - Math.pow(1 - clamp01(t), 4)
