/**
 * The state of the open vault and the changes made to it.
 */
import { vault } from '@/shared/api/clients'
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

/** The open vault: its identity and how far its scan got. */
export async function getVaultState(): Promise<VaultState> {
  const state = await vault.getVaultState({})
  return {
    id: state.id,
    name: state.name,
    path: state.path,
    scan: parseScan(state.scan),
  }
}

/** The changes made to the files of the open vault, until the caller aborts the signal. */
export async function* watchVaultChanges(signal: AbortSignal): AsyncGenerator<VaultChange> {
  for await (const change of vault.watchVaultChanges({}, { signal })) {
    yield {
      paths: change.paths,
      shouldReload: change.reload,
      renamed: change.renamed.map((went) => ({ from: went.from, to: went.to })),
      versionPaths: change.versionPaths,
    }
  }
}
