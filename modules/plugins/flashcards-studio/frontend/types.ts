/**
 * Types for Flashcards Studio plugin.
 */
import type { Ref, ComputedRef } from 'vue'
import type { DeckCard, DeckSection, Stencil, InsertionPoint, Half } from '@numen/ui'

export type { DeckCard, DeckSection, Stencil, InsertionPoint, Half }

export interface StencilFace {
  readonly id: string
  readonly name: string
  readonly front: string
  readonly back: string
}

export interface DeckDataState {
  readonly id?: string
  readonly note?: ComputedRef<unknown>
  readonly deck?: ComputedRef<unknown>
  readonly drawn?: ComputedRef<readonly DeckCard[]>
  readonly sections?: ComputedRef<readonly DeckSection[]>
  readonly stencils?: ComputedRef<readonly Stencil[]>
  readonly marks?: ComputedRef<unknown>
  readonly errorMessage?: ComputedRef<string>
}

export interface DeckScheduleState {
  readonly scheduled?: ComputedRef<unknown>
  readonly choices?: ComputedRef<readonly unknown[]>
  setSchedule?(preset: string): void
}

export interface DeckActions {
  addCard?(stencil: string, values: unknown, section: string | null): void
  removeCard?(card: string): void
  moveCard?(card: string, at: unknown): void
  writeCardField?(card: string, field: string, nth: number, text: string): void
  addSection?(name: string): void
  renameSection?(section: string, name: string): void
  removeSection?(section: string): void
  keepMine?(): void
  takeFile?(): void
  close?(id: string): void
}

export type DeckTabState = DeckDataState & DeckScheduleState & DeckActions

export interface StencilTabState {
  readonly id?: string
  readonly note?: ComputedRef<unknown>
  readonly stencil?: Readonly<Ref<{ fields: readonly string[]; faces: readonly StencilFace[] }>>
  readonly marks?: Readonly<Ref<unknown>>
  readonly errorMessage?: Readonly<Ref<string>>
  addField?(name: string): void
  renameField?(field: string, name: string): void
  removeField?(field: string): void
  moveField?(field: string, at: InsertionPoint): void
  addFace?(name: string): void
  renameFace?(id: string, name: string): void
  removeFace?(id: string): void
  moveFace?(face: string, at: InsertionPoint): void
  writeFront?(face: string, prose: string): void
  writeBack?(face: string, prose: string): void
  writeFaceHalf?(face: string, half: Half, text: string): void
  keepMine?(): void
  takeFile?(): void
  close?(id: string): void
  [key: string]: unknown
}

export interface Point {
  readonly x?: number
  readonly y?: number
  readonly reviews?: number
  readonly minutes?: number
  [key: string]: unknown
}

export interface Curve {
  readonly points?: readonly Point[]
  readonly min?: number
  readonly max?: number
  [key: string]: unknown
}

export interface PresetCounts {
  readonly dueToday?: number
  readonly dueThisWeek?: number
  readonly totalCards?: number
  [key: string]: unknown
}

export interface SettingsBounds {
  readonly min?: number
  readonly max?: number
  readonly step?: number
  [key: string]: unknown
}

export interface Scheduling {
  readonly intervalModifier?: number
  readonly maximumIntervalDays?: number
  readonly requestRetention?: number
  readonly targetRetention?: number
  [key: string]: unknown
}

export type Settings = Scheduling

export interface PresetTabState {
  readonly id?: string
  readonly settings?: Readonly<Ref<Settings>>
  readonly bounds?: Readonly<Ref<SettingsBounds>>
  readonly problems?: Readonly<Ref<readonly string[]>>
  readonly stopped?: Readonly<Ref<unknown>>
  readonly errorMessage?: Readonly<Ref<string>> | string
  readonly hasExternalChange?: Readonly<Ref<boolean>> | boolean
  readonly material?: Readonly<Ref<PresetCounts | null>>
  readonly curve?: Readonly<Ref<Curve>>
  readonly place?: Readonly<Ref<number>>
  readonly isWaiting?: Readonly<Ref<boolean>>
  reload?(): void
  chooseGoal?(goal: string): void
  moveSlider?(place: number): void
  flushPending?(): void
  updateSetting?(field: string, value: unknown): void
  setRetention?(retention: number): void
  setGoal?(goal: string): void
  close?(id: string): void
}

export const DEFAULTS = {
  goal: 'retention',
  retention: 0.9,
  minutesADay: 15,
}
