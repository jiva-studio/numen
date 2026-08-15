import type { Extent } from '../model'

export const lerp = (from: number, to: number, t: number): number =>
  from + (to - from) * t

export const clamp01 = (value: number): number =>
  value < 0 ? 0 : value > 1 ? 1 : value

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
