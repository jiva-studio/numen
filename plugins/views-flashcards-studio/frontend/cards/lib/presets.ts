/** How the decks pointing at one preset are scheduled, and what that evaluates to. */
import { type Result } from '@numen/wire'
import { type ErrorCode } from '../wire/errors'
import {
  DEFAULT_SETTINGS,
  type BudgetUnit,
  type Rule,
  type Load,
  type Settings,
} from '../wire/settings'
import { type Goal } from '../wire/goal'
import { StopReason, type BudgetName } from '../../gen/numen/flashcards/v1/presets_pb.js'
import type { CardsFailure } from '../types'

export type { Goal, BudgetUnit, Rule, Load, Settings }

export { DEFAULT_SETTINGS as DEFAULTS }

export type { StopReason }

/**
 * Every reason there is, `none` last. A verdict the schema gains and this does
 * not is caught where the two are mapped onto each other.
 */
export const STOP_REASONS: readonly StopReason[] = [
  StopReason.NO_MINUTES,
  StopReason.NO_CARDS,
  StopReason.NO_DAY,
  StopReason.PAST_DAY,
  StopReason.NO_LOAD,
  StopReason.NO_WEEK,
  StopReason.NOTHING,
]

export type { BudgetName }

/** The three, in the order they are offered. */
export const GOALS: readonly Goal[] = ['minutes', 'retention', 'date']

/** The two, in the order they are offered. */
export const BUDGET_UNITS: readonly BudgetUnit[] = ['cards', 'shows']

/** The two, in the order they are offered. */
export const RULES: readonly Rule[] = ['interval', 'retention']

/** The whole of a day's load, which a day nothing was said about carries. */
export const WHOLE_LOAD = 100

/** The shares of a day's load a day may be put at, in the order they are offered. */
export const LOADS: readonly number[] = [0, 10, 25, 50, 75, 90, WHOLE_LOAD]

export const getLoadOn = (load: Load, day: string): number => load[day] ?? WHOLE_LOAD

/**
 * The load with one day put at a share.
 */
export const setLoadOn = (load: Load, day: string, share: number): Load => {
  const out: Record<string, number> = { ...load }
  if (share === WHOLE_LOAD) delete out[day]
  else out[day] = share
  return out
}

/** How far one setting goes, at each end. */
export interface Bounds {
  readonly least: number
  readonly most: number
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

/** How far each setting goes until the application has said. */
export const NO_BOUNDS: SettingsBounds = {}

/** One preset as a read hands it over. */
export interface Preset {
  readonly path: string
  readonly title: string
  readonly settings: Settings
  readonly problems: readonly string[]
  readonly stops: StopReason
  readonly stopsOn: StopReason
}

/** A preset as a read found it, and how far each of its settings goes. */
export interface ReadPreset {
  /** Null where nothing names a preset, which a read answers as readily. */
  readonly preset: Preset | null
  readonly at: string
  readonly bounds: SettingsBounds
}

/** What reading a preset came back with. */
export type PresetReadResult = Result<ReadPreset, ErrorCode>

/** What writing a preset came back with. */
export type PresetWriteResult = Result<{ readonly at: string }, CardsFailure>

/** Where a preset was filed, or why none was made. */
export type MakeResult = Result<{ readonly path: string }, ErrorCode>

/** What a preset evaluates to at one place of the grid. */
export interface Point {
  readonly reviews: number
  readonly minutes: number
  readonly retained: number
  readonly owed: number
  readonly through: number
  /** Whether the pace this place sets learns every review card it can. */
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

/**
 * What a preset schedules, as the figures over the picture count it.
 */
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

/** One preset as a person choosing between them sees it. */
export interface PresetChoice {
  readonly path: string
  readonly title: string
}

/** What the window asks about the presets of a vault. */
export interface Presets {
  read(path: string): Promise<PresetReadResult>
  list(): Promise<readonly PresetChoice[]>
  createPreset(title: string, folder: string): Promise<MakeResult>
  scheduleDeck(deck: string, preset: string, observed: string): Promise<PresetWriteResult>
  getDeckPreset(deck: string): Promise<PresetReadResult>
  write(path: string, settings: Settings, observed: string): Promise<PresetWriteResult>
  curve(path: string, settings: Settings): Promise<Curve>
}
