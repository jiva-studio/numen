/**
 * Type declarations and domain models for the flashcards preset tab.
 */
import type { Ref } from 'vue'
import {
  type BudgetName,
  type StopReason,
} from '@numen/protocol'
import type { Goal } from '@numen/wire'
import type { ErrorCode } from '../shared/core'
import type { Field } from './curve'

export type { Goal, Field }

/** The three, in the order they are offered. */
export const GOALS: readonly Goal[] = ['minutes', 'retention', 'date']

/**
 * The unit a day's budget is spent in.
 */
export type BudgetUnit = 'cards' | 'shows'

/** The two, in the order they are offered. */
export const BUDGET_UNITS: readonly BudgetUnit[] = ['cards', 'shows']

/**
 * What a preset counts as learned.
 */
export type Rule = 'interval' | 'retention'

/** The two, in the order they are offered. */
export const RULES: readonly Rule[] = ['interval', 'retention']

/**
 * How the decks pointing at one preset are scheduled.
 */
export interface Settings {
  readonly goal: Goal
  /** The day the material is to be in the head, as `2026-09-30`. Empty for none. */
  readonly byDate: string
  readonly minutesADay: number
  readonly newADay: number
  readonly reviewsADay: number
  readonly retention: number
  /** The unit a day's budget is spent in. */
  readonly counts: BudgetUnit
  readonly backlog: number
  readonly load: Load
  readonly hasEvenLoad?: boolean
  readonly evenLoad: boolean
  readonly learned: Rule
  readonly interval: number
}

/** A share of a day's load for each day of the week that is not at the whole. */
export type Load = Readonly<Record<string, number>>

/** The whole of a day's load, which a day nothing was said about carries. */
export const WHOLE_LOAD = 100

/** The shares of a day's load a day may be put at, in the order they are offered. */
export const LOADS: readonly number[] = [0, 10, 25, 50, 75, 90, WHOLE_LOAD]

export const loadOn = (load: Load, day: string): number => load[day] ?? WHOLE_LOAD

/**
 * The load with one day put at a share.
 */
export const loaded = (load: Load, day: string, share: number): Load => {
  const out: Record<string, number> = { ...load }
  if (share === WHOLE_LOAD) delete out[day]
  else out[day] = share
  return out
}

/** A preset naming nothing, and how a deck pointing at none is scheduled. */
export const DEFAULTS: Settings = {
  goal: 'minutes',
  byDate: '',
  minutesADay: 20,
  newADay: 10,
  reviewsADay: 200,
  retention: 0.9,
  counts: 'cards',
  backlog: 100,
  load: {},
  hasEvenLoad: true,
  evenLoad: true,
  learned: 'interval',
  interval: 21,
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

/** What reading a preset came back with. */
export interface ReadResult {
  readonly preset: Preset | null
  readonly error: ErrorCode | null
  readonly at: string
  readonly bounds: SettingsBounds
}

/** What writing a preset came back with. */
export interface WriteResult {
  readonly error: ErrorCode | null
  readonly changed: boolean
  readonly at: string
}

/** What making a preset came back with. */
export interface MakeResult {
  readonly path: string
  readonly error: ErrorCode | null
}

/** What a preset comes to at one place of the grid. */
export interface Point {
  readonly reviews: number
  readonly minutes: number
  readonly retained: number
  readonly owed: number
  readonly through: number
  readonly isSufficient?: boolean
  readonly enough: boolean
  readonly closed: readonly BudgetName[]
  readonly clears: number
  readonly learned: number
  readonly learns?: number
  readonly short: number
  readonly backlog: readonly number[]
}

/** One place on the curve worth pointing at. */
export interface Place {
  readonly at: number
  readonly value: number
  readonly day: string
}

/** A place that falls outside the grid. */
export const NOWHERE: Place = { at: -1, value: 0, day: '' }

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
  readonly now: Place
  readonly suggested: Place
  readonly isValid?: boolean
  readonly honest: boolean
}

/** One preset as a person choosing between them sees it. */
export interface PresetChoice {
  readonly path: string
  readonly title: string
}

/** What the window asks about the presets of a vault. */
export interface Presets {
  read(path: string): Promise<ReadResult>
  list(): Promise<readonly PresetChoice[]>
  makes(title: string, folder: string): Promise<MakeResult>
  schedules(deck: string, preset: string, seen: string): Promise<WriteResult>
  scheduling(deck: string): Promise<ReadResult>
  write(path: string, settings: Settings, seen: string): Promise<WriteResult>
  curve(path: string, settings: Settings): Promise<Curve>
}

/** What a person can put into one row of the receipt. */
export type SettingValue = number | string | boolean | Load

/** What one preset tab holds. */
export interface PresetTabState {
  readonly id: string
  readonly settings: Readonly<Ref<Settings>>
  readonly curve: Readonly<Ref<Curve>>
  readonly counts: Readonly<Ref<PresetCounts | null>>
  readonly material: Readonly<Ref<PresetCounts | null>>
  readonly sliderPosition: Readonly<Ref<number>>
  readonly place: Readonly<Ref<number>>
  readonly isWaiting: Readonly<Ref<boolean>>
  readonly waiting: Readonly<Ref<boolean>>
  readonly bounds: Readonly<Ref<SettingsBounds>>
  readonly problems: Readonly<Ref<readonly string[]>>
  readonly stopped: Readonly<Ref<StopReason>>
  readonly errorMessage: Readonly<Ref<string>>
  readonly saying: Readonly<Ref<string>>
  readonly hasChanged: Readonly<Ref<boolean>>
  readonly changed: Readonly<Ref<boolean>>
  again(): void
  reload(): void
  chooses(goal: Goal): void
  chooseGoal(goal: Goal): void
  moves(place: number): void
  move(place: number): void
  moveSlider(place: number): void
  settles(): void
  save(): void
  types(field: Field, value: SettingValue): void
  updateSetting(field: Field, value: SettingValue): void
  shuts(id: string): void
  close(id: string): void
}
