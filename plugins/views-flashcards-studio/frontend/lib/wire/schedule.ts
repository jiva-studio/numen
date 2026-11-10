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

/** GetDeckPresetRequest is the request of the GetDeckPreset method. */
export interface GetDeckPresetRequest {
  deck?: string
}

/** GetDeckPresetResponse is the response of the GetDeckPreset method. */
export interface GetDeckPresetResponse {
  preset?: Preset
  error?: ErrorCode
  at?: FileStat
  bounds?: SettingsBounds
}

/** ListPresetsRequest is the request of the ListPresets method. */
export type ListPresetsRequest = Record<string, never>

/** ListPresetsResponse is the response of the ListPresets method. */
export interface ListPresetsResponse {
  presets: PresetSummary[]
}

/** PresetSummary is the listing entry of a preset. */
interface PresetSummary {
  path: string
  title: string
}

/** CreatePresetRequest is the request of the CreatePreset method. */
export interface CreatePresetRequest {
  title?: string
  path?: string
}

/** CreatePresetResponse is the response of the CreatePreset method. */
export interface CreatePresetResponse {
  path: string
  error?: ErrorCode
  isIndexOutdated: boolean
}

/** ScheduleDeckRequest is the request of the ScheduleDeck method. */
export interface ScheduleDeckRequest {
  deck?: string
  preset?: string
  baseline?: FileStat
}

/** ScheduleDeckResponse is the response of the ScheduleDeck method. */
export interface ScheduleDeckResponse {
  error?: ErrorCode
  at?: FileStat
  isIndexOutdated: boolean
}

/** ReadPresetRequest is the request of the ReadPreset method. */
export interface ReadPresetRequest {
  path?: string
}

/** ReadPresetResponse is the response of the ReadPreset method. */
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

/** WritePresetRequest is the request of the WritePreset method. */
export interface WritePresetRequest {
  path?: string
  settings?: Settings
  baseline?: FileStat
}

/** WritePresetResponse is the response of the WritePreset method. */
export interface WritePresetResponse {
  error?: ErrorCode
  at?: FileStat
  isIndexOutdated: boolean
}

/** ComputeCurveRequest is the request of the ComputeCurve method. */
export interface ComputeCurveRequest {
  path?: string
  settings?: Settings
}

/** ComputeCurveResponse is the response of the ComputeCurve method. */
export interface ComputeCurveResponse {
  curve?: Curve
}

/** GetVaultDeckPresetRequest is the request of the GetVaultDeckPreset method. */
export interface GetVaultDeckPresetRequest {
  vault?: string
  deck?: string
}

/** GetVaultDeckPresetResponse is the response of the GetVaultDeckPreset method. */
export interface GetVaultDeckPresetResponse {
  preset?: Preset
  error?: ErrorCode
}
