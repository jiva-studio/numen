import type { Bounds } from '@/shared/lib/math/bounds'
import { snapToBounds } from './snapToBounds'

/** Where an arrow key leaves the number: one step from at its current position. */
export const stepBy = (value: number | null, delta: number, bounds: Bounds): number =>
  snapToBounds((value ?? bounds.min) + delta * bounds.step, bounds)
