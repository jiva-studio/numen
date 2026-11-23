import { clamp, getDecimalPlaces, roundToPlaces, type Bounds } from '@/shared/lib/math/bounds'

const onStep = (value: number, bounds: Bounds): number => {
  if (bounds.step <= 0) return value
  const steps = Math.round((value - bounds.min) / bounds.step)
  return roundToPlaces(bounds.min + steps * bounds.step, getDecimalPlaces(bounds.step))
}

/** Where the bounds leave a number: on a place the step lays, inside the ends. */
export const fitToBounds = (value: number, bounds: Bounds): number =>
  clamp(onStep(value, bounds), bounds)
