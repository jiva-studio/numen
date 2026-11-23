/** How the decks pointing at one preset are scheduled, and what that evaluates to. */
import { type Result } from '@numen/plugin-sdk'
import {
  DEFAULT_SETTINGS,
  type BudgetUnit,
  type Rule,
  type Load,
  type Settings,
} from './wire/settings'
import { type Goal } from './wire/goal'
import type { StopReason, BudgetName } from './stop'
import type { ErrorCode } from '../types'
import type { PresetReadResult, PresetWriteResult } from './presetResults'

export type { Goal, BudgetUnit, Rule, Load, Settings }

export { DEFAULT_SETTINGS as DEFAULTS }

export type { StopReason }

/** The three, in the order they are offered. */
export const GOALS: readonly Goal[] = ['minutes', 'retention', 'date']

/** The two, in the order they are offered. */
export const BUDGET_UNITS: readonly BudgetUnit[] = ['cards', 'shows']

/** The two, in the order they are offered. */
export const RULES: readonly Rule[] = ['interval', 'retention']

/** The full load of a day, which a day with no configured load carries. */
export const WHOLE_LOAD = 100

/** The shares of a day's load a day may be put at, in the order they are offered. */
export const LOADS: readonly number[] = [0, 10, 25, 50, 75, 90, WHOLE_LOAD]

export const getLoadOn = (load: Load, day: string): number => load[day] ?? WHOLE_LOAD

export const setLoadOn = (load: Load, day: string, share: number): Load => {
  const out: Record<string, number> = { ...load }
  if (share === WHOLE_LOAD) delete out[day]
  else out[day] = share
  return out
}

/** How far one setting goes, at each end. */
export interface Bounds {
  readonly min: number
  readonly max: number
}

/**
 * How far each setting goes, as a read answers it.
 */
export interface SettingsBounds {
  readonly minutesADay?: Bounds
  readonly newADay?: Bounds
  readonly reviewsADay?: Bounds
  readonly retention?: Bounds
  readonly backlog?: Bounds
  readonly interval?: Bounds
  readonly load?: Bounds
}

/** The range of each setting until the application reports it. */
export const NO_BOUNDS: SettingsBounds = {}

/** One preset as returned by a read. */
export interface Preset {
  readonly path: string
  readonly title: string
  readonly settings: Settings
  readonly problems: readonly string[]
  readonly stops: StopReason
  readonly stopsOn: StopReason
}

/** Where a preset was filed, or why none was made. */
type MakeResult = Result<{ readonly path: string }, ErrorCode>

/** What a preset evaluates to at one place of the grid. */
export interface Point {
  readonly reviews: number
  readonly minutes: number
  readonly retained: number
  readonly due: number
  readonly through: number
  readonly canLearnEveryCard: boolean
  readonly closed: readonly BudgetName[]
  readonly clears: number
  readonly learned: number
  readonly learns?: number
  readonly short: number
  readonly backlog: readonly number[]
}

/** One point on the curve worth pointing at. */
export interface CurvePoint {
  readonly index: number
  readonly value: number
  readonly day: string
}

/** A curve point that falls outside the grid. */
export const NO_CURVE_POINT: CurvePoint = { index: -1, value: 0, day: '' }

/** What a preset schedules, as the figures on the curve count it. */
export interface PresetCounts {
  readonly decks: number
  readonly cards: number
  readonly overdue: number
  readonly unbegun: number
}

export interface Curve extends PresetCounts {
  readonly goal: Goal
  readonly grid: readonly number[]
  readonly days: readonly string[]
  readonly at: readonly Point[]
  readonly now: CurvePoint
  readonly suggested: CurvePoint
  readonly isValid?: boolean
  readonly isHonest: boolean
}

/** One preset as presented in a list of choices. */
export interface PresetChoice {
  readonly path: string
  readonly title: string
}

/** Reads and writes of the presets of a vault. */
export interface Presets {
  read(path: string): Promise<PresetReadResult>
  list(): Promise<readonly PresetChoice[]>
  createPreset(title: string, folder: string): Promise<MakeResult>
  scheduleDeck(deck: string, preset: string, observed: string): Promise<PresetWriteResult>
  getDeckPreset(deck: string): Promise<PresetReadResult>
  write(path: string, settings: Settings, observed: string): Promise<PresetWriteResult>
  getCurve(path: string, settings: Settings): Promise<Curve>
}
