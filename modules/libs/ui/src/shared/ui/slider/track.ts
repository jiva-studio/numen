/**
 * Where a key leaves the handle.
 *
 * Pure: the same value gives the same answer wherever it is read, so what the
 * handle stands at can be named without drawing it.
 */
import { stepBy, type Bounds } from '@/shared/lib/bounds'

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

/** Options for walking a handle along a slider track. */
export interface SliderStepOptions {
  readonly bounds: Bounds
  readonly isBigStep?: boolean
}

/** Where a key leaves the handle, and nothing for a key it does not answer. */
export const stepForKey = (
  key: string,
  value: number,
  options: SliderStepOptions,
): number | null => {
  const { bounds, isBigStep = false } = options
  const paces = isBigStep ? PACES : 1
  if (key === 'Home') return bounds.min
  if (key === 'End') return bounds.max
  if (key === 'ArrowRight' || key === 'ArrowUp') return stepBy(value, paces, bounds)
  if (key === 'ArrowLeft' || key === 'ArrowDown') return stepBy(value, -paces, bounds)
  if (key === 'PageUp') return stepBy(value, PACES, bounds)
  if (key === 'PageDown') return stepBy(value, -PACES, bounds)
  return null
}
