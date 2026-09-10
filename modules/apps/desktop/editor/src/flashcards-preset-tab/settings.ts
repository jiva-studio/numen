/**
 * Settings calculation, bound clamping, and reconciliation for preset tabs.
 */
import { dayAfter, daysBetween, isDay } from '@numen/ui'
import type { Bounds, Goal, Load, Settings, SettingsBounds } from './core'
import { held, nearest, type Field } from './curve'
import type { SettingValue } from './types'

/** How far off the day a goal of a date opens on, where the file names none. */
export const AHEAD = 30

/** Whether the value is a valid load record with daily numeric shares. */
export const isLoad = (value: SettingValue): value is Load =>
  typeof value === 'object' && Object.values(value).every((share) => typeof share === 'number')

/** Clamps every day of the week held inside load to within bounds. */
export const clampShares = (load: Load, within: Bounds | undefined): Load => {
  if (within === undefined) return load
  const out: Record<string, number> = {}
  for (const [day, share] of Object.entries(load)) out[day] = held(Math.round(share), within)
  return out
}

type MutableSettings = { -readonly [field in keyof Settings]: Settings[field] }

/** Reconciles newly read settings with unwritten local edits. */
export const reconcileSettings = (
  read: Settings,
  was: Settings,
  theirs: ReadonlySet<keyof Settings>,
): Settings => {
  if (theirs.size === 0) return read
  const out: MutableSettings = { ...read }
  for (const field of theirs) out[field] = was[field]
  return out
}

/** Clamps a typed value to bounds and updates the settings. */
export const applyTypedSetting = (
  settings: Settings,
  field: Field,
  value: SettingValue,
  bounds: SettingsBounds,
): Settings => {
  if (field === 'byDate' && typeof value === 'string') return { ...settings, byDate: value }
  if (field === 'counts' && (value === 'cards' || value === 'shows')) {
    return { ...settings, counts: value }
  }
  if (field === 'learned' && (value === 'interval' || value === 'retention')) {
    return { ...settings, learned: value }
  }
  if (field === 'load' && isLoad(value)) {
    return { ...settings, load: clampShares(value, bounds.load) }
  }
  if (field === 'evenLoad' && typeof value === 'boolean') return { ...settings, evenLoad: value }
  if (typeof value !== 'number') return settings
  if (field === 'retention') return { ...settings, retention: held(value, bounds.retention) }
  if (field === 'newADay') return { ...settings, newADay: held(value, bounds.newADay) }
  if (field === 'reviewsADay') return { ...settings, reviewsADay: held(value, bounds.reviewsADay) }
  if (field === 'minutesADay') return { ...settings, minutesADay: held(value, bounds.minutesADay) }
  if (field === 'backlog') return { ...settings, backlog: held(Math.round(value), bounds.backlog) }
  if (field === 'interval') return { ...settings, interval: held(Math.round(value), bounds.interval) }
  return settings
}

/** Where a value typed into the field the goal steers falls on the grid. */
export const findGridIndex = (
  grid: readonly number[],
  value: SettingValue,
  today: string,
): number => {
  if (typeof value === 'number') return nearest(grid, value)
  if (typeof value === 'string' && isDay(value)) {
    return nearest(grid, daysBetween(today, value))
  }
  return -1
}

/** Sets a date goal and defaults byDate when empty. */
export const aimGoal = (settings: Settings, goal: Goal, today: string): Settings =>
  goal === 'date' && settings.byDate === ''
    ? { ...settings, goal, byDate: dayAfter(today, AHEAD) }
    : { ...settings, goal }
