/**
 * Type declarations and domain models for the flashcards preset tab.
 */
import type { Ref } from 'vue'
import type { Field } from './lib/fields'
import type {
  Curve,
  Goal,
  Load,
  PresetCounts,
  Settings,
  SettingsBounds,
  StopReason,
} from '../cards'

export type {
  Goal,
  BudgetUnit,
  Rule,
  Settings,
  Load,
  Bounds,
  SettingsBounds,
  Preset,
  PresetReadResult,
  PresetWriteResult,
  ReadPreset,
  MakeResult,
  Point,
  CurvePoint,
  PresetCounts,
  Curve,
  PresetChoice,
  StopReason,
  Presets,
} from '../cards'
export {
  GOALS,
  BUDGET_UNITS,
  RULES,
  STOP_REASONS,
  WHOLE_LOAD,
  LOADS,
  getLoadOn,
  setLoadOn,
  DEFAULTS,
  NO_BOUNDS,
  NO_CURVE_POINT,
} from '../cards'

export type { Field }

/** The value of one row of the receipt. */
export type SettingValue = number | string | boolean | Load

export interface PresetStatus {
  readonly problems: Readonly<Ref<readonly string[]>>
  readonly stopped: Readonly<Ref<StopReason>>
  readonly errorMessage: Readonly<Ref<string>>
  readonly hasExternalChange: Readonly<Ref<boolean>>
}

export interface PresetCurveData {
  readonly curve: Readonly<Ref<Curve>>
  readonly place: Readonly<Ref<number>>
  readonly isWaiting: Readonly<Ref<boolean>>
  readonly material: Readonly<Ref<PresetCounts | null>>
}

export interface PresetTabActions {
  reload(): void
  chooseGoal(goal: Goal): void
  moveSlider(place: number): void
  flushPending(): void
  updateSetting(field: Field, value: SettingValue): void
  close(id: string): void
}

/** What one preset tab holds. */
export interface PresetTabState {
  readonly id: string
  readonly settings: Readonly<Ref<Settings>>
  readonly bounds: Readonly<Ref<SettingsBounds>>
  readonly curves: PresetCurveData
  readonly status: PresetStatus
  readonly actions: PresetTabActions
}
