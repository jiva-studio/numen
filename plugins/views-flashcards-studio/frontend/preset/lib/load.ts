import type { Bounds, Load } from '../../lib/presets'
import { clamp } from './curve'
import type { SettingValue } from './settings'

/** Whether the value is a valid load record with daily numeric shares. */
export const isLoad = (value: SettingValue): value is Load =>
  value !== null &&
  typeof value === 'object' &&
  Object.values(value).every((share) => typeof share === 'number')

/** Clamps every day of the week held inside load to within bounds. */
export const clampShares = (load: Load, within: Bounds | undefined): Load => {
  if (within === undefined) return load
  const out: Record<string, number> = {}
  for (const [day, share] of Object.entries(load)) out[day] = clamp(Math.round(share), within)
  return out
}
