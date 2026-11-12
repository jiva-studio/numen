/**
 * The state of the open vault and the changes made to it.
 */
import { agentService, vault } from '@/shared/api/clients'
import type { VaultChange, VaultState } from '../lib/vault'

interface ScanState {
  readonly isReady: boolean
  readonly error: string
  readonly watchError: string
}

interface CoverageState {
  readonly chunkCount: bigint
  readonly embeddedCount: bigint
  readonly isEmbedding: boolean
}

const parseScan = (scan: ScanState | undefined) => ({
  isReady: scan?.isReady ?? false,
  error: scan?.error ?? '',
  unwatchedPath: scan?.watchError ?? '',
})

const parseCoverage = (coverage: CoverageState | undefined) => ({
  chunkCount: coverage?.chunkCount ?? 0n,
  embeddedCount: coverage?.embeddedCount ?? 0n,
  isEmbedding: coverage?.isEmbedding ?? false,
})

/** The open vault: its identity, how far its scan got and how much of it is indexed. */
export async function getVaultState(): Promise<VaultState> {
  const state = await vault.getVaultState({})
  return {
    id: state.id,
    name: state.name,
    path: state.path,
    scan: parseScan(state.scan),
    coverage: parseCoverage(state.coverage),
  }
}

/**
 * Why an agent cannot be reached, or an empty string when it can. Which agent answers is
 * determined by the installation, so the check is made on the agent.
 */
export async function getAgentUnreachable(): Promise<string> {
  return (await agentService.getAgentState({})).unreachable
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
