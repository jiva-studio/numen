import type { Extent } from '../frame'

export const lerp = (from: number, to: number, t: number): number => from + (to - from) * t

/**
 * A fraction clamped to the range 0 to 1.
 *
 * NaN returns 1. A NaN passed on becomes `opacity="NaN"` in a transform and
 * the node is not drawn, with no error.
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
