/**
 * Type declarations and domain models for the flashcards preset tab.
 */
import type { Ref } from 'vue'
import type { Field } from '../lib/fields'
import type { SettingValue } from '../lib/settings'
import type {
  Curve,
  Goal,
  PresetCounts,
  Settings,
  SettingsBounds,
  StopReason,
} from '../../lib/presets'

/** The value of one row of the receipt. */

interface PresetStatus {
  readonly problems: Readonly<Ref<readonly string[]>>
  readonly stopped: Readonly<Ref<StopReason>>
  readonly errorMessage: Readonly<Ref<string>>
  readonly hasExternalChange: Readonly<Ref<boolean>>
}

interface PresetCurveData {
  readonly curve: Readonly<Ref<Curve>>
  readonly place: Readonly<Ref<number>>
  readonly isWaiting: Readonly<Ref<boolean>>
  readonly material: Readonly<Ref<PresetCounts | null>>
}

interface PresetTabActions {
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
