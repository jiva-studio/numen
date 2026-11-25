import { ref, shallowRef, type Ref } from 'vue'
import { getCurrentDate, runInBackground } from '@numen/ui'
import type { VaultsClient } from '../api/vaults'
import { getRow, getSyncPercent, type VaultRow } from './vaults'

const DEFAULT_POLL_EVERY_MS = 10_000

const ignore = (): void => undefined

export type SyncPhase = 'hidden' | 'syncing' | 'synced' | 'waiting'

export interface SyncState {
  readonly phase: Readonly<Ref<SyncPhase>>
  readonly percent: Readonly<Ref<number | null>>
  readonly rows: Readonly<Ref<readonly VaultRow[]>>
  start(): void
  stop(): void
}

export function createSyncState(
  client: Pick<VaultsClient, 'getStatus' | 'listVaults'>,
  pollEveryMs: number = DEFAULT_POLL_EVERY_MS,
): SyncState {
  const phase = ref<SyncPhase>('hidden')
  const percent = ref<number | null>(null)
  const rows = shallowRef<readonly VaultRow[]>([])
  let timer: ReturnType<typeof setTimeout> | undefined
  let isRunning = false

  const clear = (): void => {
    percent.value = null
    rows.value = []
  }

  const update = async (): Promise<boolean> => {
    try {
      const status = await client.getStatus()
      if (!status.hasIdentity) {
        phase.value = 'hidden'
        clear()
        return true
      }
      const { vaults } = await client.listVaults()
      const connected = vaults.filter((vault) => vault.state === 'connected')
      rows.value = connected.map((vault) =>
        getRow(vault, getCurrentDate(), status.isReachable, []),
      )
      const isSyncing = status.isReachable && connected.some((vault) => vault.syncing)
      percent.value = isSyncing ? getSyncPercent(connected) : null
      if (!status.isReachable) phase.value = 'waiting'
      else phase.value = isSyncing ? 'syncing' : 'synced'
      return true
    } catch {
      phase.value = 'hidden'
      clear()
      return false
    }
  }

  const poll = async (): Promise<void> => {
    await update()
    if (isRunning) timer = setTimeout(() => runInBackground(poll(), ignore), pollEveryMs)
  }

  return {
    phase,
    percent,
    rows,
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
