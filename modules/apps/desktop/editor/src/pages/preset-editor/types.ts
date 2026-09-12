/**
 * Type declarations and domain models for the flashcards preset tab.
 */
import type { Ref } from 'vue'
import type { StopReason } from '@numen/protocol'
import type { Field } from './curve'
import type {
  Curve,
  Goal,
  Load,
  PresetCounts,
  Settings,
  SettingsBounds,
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
  ReadResult,
  WriteResult,
  MakeResult,
  Point,
  Place,
  PresetCounts,
  Curve,
  PresetChoice,
  Presets,
} from '@/entities/deck'
export {
  GOALS,
  BUDGET_UNITS,
  RULES,
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
  readonly changed: Readonly<Ref<boolean>>
  readonly material: Readonly<Ref<PresetCounts | null>>
}

export interface PresetCurveData {
  readonly curve: Readonly<Ref<Curve>>
  readonly place: Readonly<Ref<number>>
  readonly waiting: Readonly<Ref<boolean>>
}

export interface PresetTabActions {
  again(): void
  chooses(goal: Goal): void
  moves(place: number): void
  settles(): void
  types(field: Field, value: SettingValue): void
  shuts(id: string): void
}

/** What one preset tab holds. */
export type PresetTabState = PresetSettingsData & PresetCurveData & PresetTabActions
