import { runInBackground } from '@numen/ui'
import type { VaultFlagsHost } from '@numen/plugin-sdk'
import type { VaultsClient } from '../api/vaults'

export const SYNCED_FLAG = 'isSynced'

const DEFAULT_REFRESH_EVERY_MS = 5000

const ignore = (): void => undefined

export interface SyncedFlags {
  refresh(): Promise<void>
  start(): void
  stop(): void
}

/**
 * Keeps the flag `isSynced` of every vault of the window true while the vault is connected to a synced vault,
 * which the conditions of the menus read as `vault.flags.md.numen.tools.sync.isSynced`. Where the list of the
 * synced vaults cannot be read, as before sign-in, no vault is synced.
 */
export function createSyncedFlags(deps: {
  readonly client: Pick<VaultsClient, 'listVaults'>
  readonly flags: VaultFlagsHost
  readonly getVaultIds: () => readonly string[]
  readonly refreshEveryMs?: number
}): SyncedFlags {
  let timer: ReturnType<typeof setInterval> | undefined

  const refresh = async (): Promise<void> => {
    const connected = new Set<string>()
    try {
      for (const one of (await deps.client.listVaults()).vaults) {
        if (one.state === 'connected') connected.add(one.id)
      }
    } catch {
      connected.clear()
    }
    for (const id of deps.getVaultIds()) deps.flags.setFlag(id, SYNCED_FLAG, connected.has(id))
  }

  return {
    refresh,
    start: () => {
      runInBackground(refresh(), ignore)
      timer = setInterval(
        () => runInBackground(refresh(), ignore),
        deps.refreshEveryMs ?? DEFAULT_REFRESH_EVERY_MS,
      )
    },
    stop: () => {
      clearInterval(timer)
      timer = undefined
    },
  }
}
