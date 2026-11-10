/**
 * The state of the open vault and the changes made to it, as the window core reads them.
 */
import { agentService, vault } from '@/shared/api/clients'
import type { VaultPort } from '@/app/ports/vault'

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

type VaultSessionCore = Pick<VaultPort, 'state' | 'agentUnreachable' | 'watchVaultChanges'>

export const vaultSessionCore: VaultSessionCore = {
  state: async () => {
    const state = await vault.getVaultState({})
    return {
      id: state.id,
      name: state.name,
      path: state.path,
      scan: parseScan(state.scan),
      coverage: parseCoverage(state.coverage),
    }
  },
  agentUnreachable: async () => (await agentService.getAgentState({})).unreachable,
  watchVaultChanges: async function* (signal) {
    for await (const change of vault.watchVaultChanges({}, { signal })) {
      yield {
        paths: change.paths,
        shouldReload: change.reload,
        renamed: change.renamed.map((went) => ({ from: went.from, to: went.to })),
        versionPaths: change.versionPaths,
      }
    }
  },
}
