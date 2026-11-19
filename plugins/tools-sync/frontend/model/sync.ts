import { ref, type Ref } from 'vue'
import type { Disposable, PluginLicenses } from '@numen/plugin-sdk'

export const SYNC_LICENSE_ID = 'sync'
export const DEFAULT_SYNC_DURATION_MS = 2000

export type SyncPhase = 'hidden' | 'syncing' | 'synced'

export interface SyncState {
  readonly phase: Readonly<Ref<SyncPhase>>
  start(): void
  stop(): void
}

export function createSyncState(
  licenses: PluginLicenses,
  syncDurationMs: number = DEFAULT_SYNC_DURATION_MS,
): SyncState {
  const phase = ref<SyncPhase>('hidden')
  let watching: Disposable | undefined
  let timer: ReturnType<typeof setTimeout> | undefined

  const clearTimer = (): void => {
    if (timer !== undefined) {
      clearTimeout(timer)
      timer = undefined
    }
  }

  const applyActive = (isActive: boolean): void => {
    clearTimer()
    if (!isActive) {
      phase.value = 'hidden'
      return
    }
    phase.value = 'syncing'
    timer = setTimeout(() => {
      timer = undefined
      phase.value = 'synced'
    }, syncDurationMs)
  }

  return {
    phase,
    start: () => {
      applyActive(licenses.has(SYNC_LICENSE_ID))
      watching = licenses.onChange(SYNC_LICENSE_ID, (next) => {
        applyActive(next)
      })
    },
    stop: () => {
      clearTimer()
      watching?.dispose()
      watching = undefined
    },
  }
}
