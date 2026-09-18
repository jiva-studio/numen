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
} from '@/entities/deck'

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
  Place,
  PresetCounts,
  Curve,
  PresetChoice,
  StopReason,
  Presets,
} from '@/entities/deck'
export {
  GOALS,
  BUDGET_UNITS,
  RULES,
  STOP_REASONS,
  WHOLE_LOAD,
  LOADS,
  loadOn,
  setLoadOn,
  DEFAULTS,
  NO_BOUNDS,
  NOWHERE,
} from '@/entities/deck'

export type { Field }

/** What a person can put into one row of the receipt. */
export type SettingValue = number | string | boolean | Load

export interface PresetSettingsData {
  readonly id: string
  readonly settings: Readonly<Ref<Settings>>
  readonly bounds: Readonly<Ref<SettingsBounds>>
  readonly problems: Readonly<Ref<readonly string[]>>
  readonly stopped: Readonly<Ref<StopReason>>
  readonly errorMessage: Readonly<Ref<string>>
  readonly hasChanged: Readonly<Ref<boolean>>
  readonly material: Readonly<Ref<PresetCounts | null>>
}

export interface PresetCurveData {
  readonly curve: Readonly<Ref<Curve>>
  readonly place: Readonly<Ref<number>>
  readonly isWaiting: Readonly<Ref<boolean>>
}

export interface PresetTabActions {
  reload(): void
  chooseGoal(goal: Goal): void
  moveSlider(place: number): void
  settle(): void
  updateSetting(field: Field, value: SettingValue): void
  close(id: string): void
}

/** What one preset tab holds. */
export type PresetTabState = PresetSettingsData & PresetCurveData & PresetTabActions
