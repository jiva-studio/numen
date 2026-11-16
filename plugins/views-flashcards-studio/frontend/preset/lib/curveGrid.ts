import type { Curve, Settings } from '../../lib/presets'
import { getGoalValue, findNearestGridValue } from './curve'

/** Whether two curves are drawn over the same range, place for place. */
export const isSameGrid = (one: readonly number[], two: readonly number[]): boolean =>
  one.length === two.length && one.every((value, index) => value === two[index])

/** Where the knob relies on a curve: the preset's own place, or the nearest. */
export const getKnobPosition = (curve: Curve, settings: Settings, today: string): number =>
  curve.now.index >= 0
    ? curve.now.index
    : Math.max(findNearestGridValue(curve.grid, getGoalValue(settings, today)), 0)
