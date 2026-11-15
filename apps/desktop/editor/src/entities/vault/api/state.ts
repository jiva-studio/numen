/**
 * The state of the open vault and the changes made to it.
 */
import type { WindowClients } from '@/shared/api/clients'
import type { VaultChange, VaultState } from '../lib/vault'

interface ScanState {
  readonly isReady: boolean
  readonly error: string
  readonly watchError: string
}

const parseScan = (scan: ScanState | undefined) => ({
  isReady: scan?.isReady ?? false,
  error: scan?.error ?? '',
  unwatchedPath: scan?.watchError ?? '',
})

/** The state of the open vault over `vault`. */
export const createVaultStateApi = (vault: WindowClients['vault']) => {
  /** The open vault: its identity and how far its scan got. */
  const getVaultState = async (): Promise<VaultState> => {
    const state = await vault.getVaultState({})
    return {
      id: state.id,
      name: state.name,
      path: state.path,
      scan: parseScan(state.scan),
    }
  }

  /** The changes made to the files of the open vault, until the caller aborts the signal. */
  async function* watchVaultChanges(signal: AbortSignal): AsyncGenerator<VaultChange> {
    for await (const change of vault.watchVaultChanges({}, { signal })) {
      yield {
        paths: change.paths,
        shouldReload: change.reload,
        renamed: change.renamed.map((went) => ({ from: went.from, to: went.to })),
        versionPaths: change.versionPaths,
      }
    }
  }

  return { getVaultState, watchVaultChanges }
}

export type VaultStateApi = ReturnType<typeof createVaultStateApi>
