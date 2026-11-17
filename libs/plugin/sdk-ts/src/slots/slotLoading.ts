/**
 * The readiness of the views of a slot. A view that fetches its data calls `useSlotLoading` in `setup`
 * and the `finish` it returns once the data has arrived; the owner of the slot draws the slot with
 * `createSlotLoading` and `provideSlotLoading`, and waits for `isReady` before it lays the slot out.
 */
import { inject, ref, type InjectionKey, type Ref } from 'vue'
import type { SlotEntryKey } from './slots'

// SLOT_LOADING_CAP_MS is how long a view may stay loading before the host stops waiting for it.
export const SLOT_LOADING_CAP_MS = 1000

// SlotLoadingHost is what the owner of a slot gives the views of one entry.
export interface SlotLoadingHost {
  // Registers a view as loading and answers the call that releases it.
  begin(): () => void
}

export const SLOT_LOADING_KEY: InjectionKey<SlotLoadingHost> = Symbol.for('numen.slotLoading')

// SlotLoadingClock draws on the frames of the window; a test supplies its own.
export interface SlotLoadingClock {
  readonly now: () => number
  readonly schedule: (run: (now: number) => void) => number
  readonly cancel: (handle: number) => void
}

export interface SlotLoadingDeps {
  readonly clock: SlotLoadingClock
  readonly reportIssue?: (pluginId: string, message: string) => void
}

// SlotLoading is the readiness of all the views the owner of a slot has mounted.
export interface SlotLoading {
  readonly isReady: Readonly<Ref<boolean>>
}

// The entry whose views the owner of a slot draws.
export interface SlotLoadingEntry {
  readonly key: SlotEntryKey
  readonly pluginId: string
}

export function useSlotLoading(): () => void {
  inject(SLOT_LOADING_KEY, undefined)
  return () => undefined
}

export function createSlotLoading(_deps: SlotLoadingDeps): SlotLoading {
  return { isReady: ref(true) }
}

// Called in `setup` of the component that draws the views of one entry.
export function provideSlotLoading(_loading: SlotLoading, _entry: SlotLoadingEntry): void {}
