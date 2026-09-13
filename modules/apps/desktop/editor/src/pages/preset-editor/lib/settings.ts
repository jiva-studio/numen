/**
 * Settings calculation, bound clamping, and reconciliation for preset tabs.
 */
import { dayAfter, daysBetween, isDay } from '@numen/ui'
import type { Bounds, BudgetUnit, Goal, Load, Rule, Settings, SettingsBounds } from '../types'
import { clamp, nearest } from './curve'
import type { Field } from './fields'
import type { SettingValue } from '../types'

/** How far off the day a goal of a date opens on, where the file names none. */
export const AHEAD = 30

/** Whether the value is a valid load record with daily numeric shares. */
export const isLoad = (value: SettingValue): value is Load =>
  typeof value === 'object' && Object.values(value).every((share) => typeof share === 'number')

/** Clamps every day of the week held inside load to within bounds. */
export const clampShares = (load: Load, within: Bounds | undefined): Load => {
  if (within === undefined) return load
  const out: Record<string, number> = {}
  for (const [day, share] of Object.entries(load)) out[day] = clamp(Math.round(share), within)
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
  for (const field of theirs) {
    ;(out as Record<keyof Settings, unknown>)[field] = was[field]
  }
  return out
}

/** The fields holding a number, and whether the field counts in whole ones. */
const NUMBER_FIELDS = {
  retention: false,
  newADay: false,
  reviewsADay: false,
  minutesADay: false,
  backlog: true,
  interval: true,
} as const

type NumberField = keyof typeof NUMBER_FIELDS

const isNumberField = (field: Field): field is NumberField => field in NUMBER_FIELDS

const isBudgetUnit = (value: SettingValue): value is BudgetUnit =>
  value === 'cards' || value === 'shows'

const isRule = (value: SettingValue): value is Rule =>
  value === 'interval' || value === 'retention'

/** Clamps a number to the field's bounds, rounding where the field is whole. */
const applyNumberSetting = (
  settings: Settings,
  field: NumberField,
  value: number,
  bounds: SettingsBounds,
): Settings => {
  const out: MutableSettings = { ...settings }
  out[field] = clamp(NUMBER_FIELDS[field] ? Math.round(value) : value, bounds[field])
  return out
}

/** Sets a field that takes no bounds, or nothing where the value is not its own. */
const applyChoiceSetting = (
  settings: Settings,
  field: Field,
  value: SettingValue,
): Settings | undefined => {
  if (field === 'byDate' && typeof value === 'string') return { ...settings, byDate: value }
  if (field === 'counts' && isBudgetUnit(value)) return { ...settings, counts: value }
  if (field === 'learned' && isRule(value)) return { ...settings, learned: value }
  if (field === 'evenLoad' && typeof value === 'boolean') return { ...settings, evenLoad: value }
  return undefined
}

/** Clamps a typed value to bounds and updates the settings. */
export const applyTypedSetting = (
  settings: Settings,
  field: Field,
  value: SettingValue,
  bounds: SettingsBounds,
): Settings => {
  const chosen = applyChoiceSetting(settings, field, value)
  if (chosen !== undefined) return chosen
  if (field === 'load' && isLoad(value)) {
    return { ...settings, load: clampShares(value, bounds.load) }
  }
  if (isNumberField(field) && typeof value === 'number') {
    return applyNumberSetting(settings, field, value, bounds)
  }
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
