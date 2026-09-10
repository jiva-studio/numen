/**
 * Type declarations for the flashcards preset tab domain.
 */
import type { Ref } from 'vue'
import { StopReason } from '@numen/protocol'
import type { Curve, Goal, Load, PresetCounts, Settings, SettingsBounds } from './core'
import type { Field } from './curve'

/** What a person can put into one row of the receipt. */
export type SettingValue = number | string | boolean | Load

/** What one preset tab holds. */
export interface PresetTabState {
  /** The identity this preset opened under, which its tab keeps wherever it goes. */
  readonly id: string
  /** The settings as they now stand, whether or not they have been written. */
  readonly settings: Readonly<Ref<Settings>>
  /** The curve of the goal, which is the control the person moves. */
  readonly curve: Readonly<Ref<Curve>>
  /** What the preset schedules, as the last answer counted it. */
  readonly counts: Readonly<Ref<PresetCounts | null>>
  readonly material: Readonly<Ref<PresetCounts | null>>
  /** Where the knob stands on that curve. */
  readonly sliderPosition: Readonly<Ref<number>>
  readonly place: Readonly<Ref<number>>
  /** Whether an answer to the curve calculation is pending. */
  readonly isWaiting: Readonly<Ref<boolean>>
  readonly waiting: Readonly<Ref<boolean>>
  /** How far each setting goes, as the application answers it. */
  readonly bounds: Readonly<Ref<SettingsBounds>>
  /** What is wrong with the file, in the words to show. */
  readonly problems: Readonly<Ref<readonly string[]>>
  /** Why it schedules nothing on the day it was read in. */
  readonly stopped: Readonly<Ref<StopReason>>
  /** What the file was refused for, in words a person reads, or empty string. */
  readonly errorMessage: Readonly<Ref<string>>
  readonly saying: Readonly<Ref<string>>
  /** Whether the file moved externally on disk and was not overwritten. */
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
