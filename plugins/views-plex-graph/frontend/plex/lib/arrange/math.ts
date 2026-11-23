import type { Extent } from '../frame'

export const interpolate = (from: number, to: number, t: number): number => from + (to - from) * t

/**
 * A fraction clamped to the range 0 to 1.
 *
 * NaN returns 1. A NaN passed on becomes `opacity="NaN"` in a transform and
 * the node is not drawn, with no error.
 */
export const clampToUnit = (value: number): number => {
  if (Number.isNaN(value)) return 1
  return Math.min(Math.max(value, 0), 1)
}

export function interpolateExtent(from: Extent, to: Extent, t: number): Extent {
  return {
    minX: interpolate(from.minX, to.minX, t),
    minY: interpolate(from.minY, to.minY, t),
    maxX: interpolate(from.maxX, to.maxX, t),
    maxY: interpolate(from.maxY, to.maxY, t),
  }
}

/** Fast to leave, slow to settle: the useful part of a move is the end. */
export const computeEaseOut = (t: number): number => 1 - Math.pow(1 - clampToUnit(t), 4)
