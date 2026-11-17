/**
 * The readiness of the views of a slot. A view that fetches its data calls `useSlotLoading` in `setup`
 * and the `finish` it returns once the data has arrived; the owner of the slot draws the slot with
 * `createSlotLoading` and `provideSlotLoading`, and waits for `isReady` before it lays the slot out.
 */
import {
  computed,
  inject,
  onUnmounted,
  provide,
  shallowReactive,
  type InjectionKey,
  type Ref,
} from 'vue'
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
  // Registers a view of an entry as loading and answers the call that releases it.
  begin(entry: SlotLoadingEntry): () => void
}

// The entry whose views the owner of a slot draws.
export interface SlotLoadingEntry {
  readonly key: SlotEntryKey
  readonly pluginId: string
}

// Called in `setup` of a view: the view is loading until the `finish` it gets is called.
// A view outside the host of a slot, or one that never calls this, is never waited for.
export function useSlotLoading(): () => void {
  const release = inject(SLOT_LOADING_KEY, undefined)?.begin()
  let isFinished = false
  return () => {
    if (isFinished) return
    isFinished = true
    release?.()
  }
}

interface Token {
  readonly entry: SlotLoadingEntry
  readonly startedAt: number
}

export function createSlotLoading({ clock, reportIssue }: SlotLoadingDeps): SlotLoading {
  const tokens = shallowReactive(new Set<Token>())
  let isWatching = false

  // A view that hangs is released and reported: the cap is a safety net, not a way to finish.
  const watchCap = () => {
    if (isWatching || tokens.size === 0) return
    isWatching = true
    clock.schedule((now) => {
      isWatching = false
      for (const token of [...tokens]) {
        if (now - token.startedAt < SLOT_LOADING_CAP_MS) continue
        tokens.delete(token)
        reportIssue?.(token.entry.pluginId, `${token.entry.key} did not report ready`)
      }
      watchCap()
    })
  }

  return {
    isReady: computed(() => tokens.size === 0),
    begin: (entry) => {
      const token: Token = { entry, startedAt: clock.now() }
      tokens.add(token)
      watchCap()
      return () => tokens.delete(token)
    },
  }
}

// Called in `setup` of the component that draws the views of one entry.
export function provideSlotLoading(loading: SlotLoading, entry: SlotLoadingEntry): void {
  const releases = new Set<() => void>()
  provide(SLOT_LOADING_KEY, {
    begin: () => {
      const release = loading.begin(entry)
      releases.add(release)
      return () => {
        releases.delete(release)
        release()
      }
    },
  })
  onUnmounted(() => {
    for (const release of releases) release()
  })
}
