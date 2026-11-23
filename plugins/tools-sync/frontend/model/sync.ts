import { ref, type Ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { VaultsClient } from '../api/vaults'

const DEFAULT_POLL_EVERY_MS = 10_000

const ignore = (): void => undefined

export type SyncPhase = 'hidden' | 'syncing' | 'synced'

export interface SyncState {
  readonly phase: Readonly<Ref<SyncPhase>>
  start(): void
  stop(): void
}

export function createSyncState(
  client: Pick<VaultsClient, 'getStatus'>,
  pollEveryMs: number = DEFAULT_POLL_EVERY_MS,
): SyncState {
  const phase = ref<SyncPhase>('hidden')
  let timer: ReturnType<typeof setTimeout> | undefined
  let isRunning = false

  const update = async (): Promise<void> => {
    try {
      const status = await client.getStatus()
      phase.value = status.hasIdentity ? 'synced' : 'hidden'
    } catch {
      // The status is asked again at the next poll; the badge keeps what it showed.
    }
  }

  const poll = async (): Promise<void> => {
    await update()
    if (isRunning) timer = setTimeout(() => runInBackground(poll(), ignore), pollEveryMs)
  }

  return {
    phase,
    start: () => {
      isRunning = true
      phase.value = 'syncing'
      runInBackground(poll(), ignore)
    },
    stop: () => {
      isRunning = false
      clearTimeout(timer)
      timer = undefined
    },
  }
}
