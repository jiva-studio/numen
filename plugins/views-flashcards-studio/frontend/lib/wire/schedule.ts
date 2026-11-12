/** The JSON messages of the presets of this plugin: their settings, projection curves and the preset of a deck. */
import type { BudgetName, StopReason } from '../stop'
import type { Bounds, ErrorCode, FileStat } from './notes'

/** Goal is what a preset schedules toward. */
export const Goal = {
  UNSPECIFIED: '',
  MINUTES_A_DAY: 'minutes_a_day',
  RETENTION: 'retention',
  BY_DATE: 'by_date',
} as const
export type Goal = (typeof Goal)[keyof typeof Goal]

/** Rule is how the interval of a learned card is set. */
export const Rule = {
  UNSPECIFIED: '',
  INTERVAL: 'interval',
  RETENTION: 'retention',
} as const
export type Rule = (typeof Rule)[keyof typeof Rule]

/** BudgetUnit is what the daily budget counts. */
export const BudgetUnit = {
  UNSPECIFIED: '',
  CARDS: 'cards',
  SHOWS: 'shows',
} as const
export type BudgetUnit = (typeof BudgetUnit)[keyof typeof BudgetUnit]

/** Settings is the scheduling settings of a preset. Load maps a weekday name to the cards due that day. */
interface Settings {
  goal: Goal
  byDate: string
  minutesADay: number
  newADay: number
  reviewsADay: number
  retention: number
  hasEvenLoad: boolean
  counts: BudgetUnit
  backlog: number
  load: Record<string, number>
  learned: Rule
  interval: number
}

/** Preset is a preset note with its settings and the reasons it schedules nothing. */
export interface Preset {
  path: string
  title: string
  settings?: Settings
  problems: string[]
  stops: StopReason
  stopsOn: StopReason
}

/** Curve is the projection of a preset over the range of its goal. */
export interface Curve {
  goal: Goal
  grid: number[]
  days: string[]
  at: Point[]
  now?: CurvePoint
  suggested?: CurvePoint
  decks: number
  cards: number
  overdue: number
  unbegun: number
}

/** Point is one sample of a projection curve. Learns is absent when the learned rule has no count. */
export interface Point {
  reviews: number
  minutes: number
  retained: number
  due: number
  through: number
  isEnough: boolean
  closed: BudgetName[]
  clears: number
  backlog: number[]
  learned: number
  learns?: number
  short: number
}

/** CurvePoint is a marked position on a projection curve. */
export interface CurvePoint {
  at: number
  value: number
  day: string
}

export interface GetDeckPresetRequest {
  deck?: string
}

export interface GetDeckPresetResponse {
  preset?: Preset
  error?: ErrorCode
  at?: FileStat
  bounds?: SettingsBounds
}

export type ListPresetsRequest = Record<string, never>

export interface ListPresetsResponse {
  presets: PresetSummary[]
}

/** PresetSummary is the listing entry of a preset. */
interface PresetSummary {
  path: string
  title: string
}

export interface CreatePresetRequest {
  title?: string
  path?: string
}

export interface CreatePresetResponse {
  path: string
  error?: ErrorCode
  isIndexOutdated: boolean
}

export interface ScheduleDeckRequest {
  deck?: string
  preset?: string
  baseline?: FileStat
}

export interface ScheduleDeckResponse {
  error?: ErrorCode
  at?: FileStat
  isIndexOutdated: boolean
}

export interface ReadPresetRequest {
  path?: string
}

export interface ReadPresetResponse {
  preset?: Preset
  error?: ErrorCode
  at?: FileStat
  bounds?: SettingsBounds
}

/** SettingsBounds is the range each numeric setting of a preset may take. */
export interface SettingsBounds {
  minutesADay?: Bounds
  newADay?: Bounds
  reviewsADay?: Bounds
  retention?: Bounds
  backlog?: Bounds
  interval?: Bounds
  load?: Bounds
}

export interface WritePresetRequest {
  path?: string
  settings?: Settings
  baseline?: FileStat
}

export interface WritePresetResponse {
  error?: ErrorCode
  at?: FileStat
  isIndexOutdated: boolean
}

export interface ComputeCurveRequest {
  path?: string
  settings?: Settings
}

export interface ComputeCurveResponse {
  curve?: Curve
}

export interface GetVaultDeckPresetRequest {
  vault?: string
  deck?: string
}

export interface GetVaultDeckPresetResponse {
  preset?: Preset
  error?: ErrorCode
}
