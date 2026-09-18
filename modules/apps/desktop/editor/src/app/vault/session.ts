/**
 * Session, lifecycle, and workspace domain methods for the window core.
 */
import type { IndexCoverage, Scan } from '@numen/protocol'
import { WINDOW } from './clients'
import { agentService, vault, windowService, workspace } from '@/shared/clients'
import { counted, flushResults } from './words'
import type { VaultPort } from '@/app/ports/vault'

/** How far the scan of the vault has got. */
const parseScan = (scan: Scan | undefined) => ({
  isReady: scan?.isReady ?? false,
  error: scan?.error ?? '',
  unwatchedPath: scan?.unwatched ?? '',
})

/** How much of the vault the index holds. */
const parseCoverage = (coverage: IndexCoverage | undefined) => ({
  chunkCount: coverage?.chunkCount ?? 0n,
  embeddedCount: coverage?.embeddedCount ?? 0n,
  isEmbedding: coverage?.isEmbedding ?? false,
})

export type SessionCore = Pick<
  VaultPort,
  | 'state'
  | 'agentUnreachable'
  | 'watchVaultChanges'
  | 'watchFocus'
  | 'writeOpenTabs'
  | 'watchTasks'
  | 'watchQuit'
  | 'reportFlush'
>

export const sessionCore: SessionCore = {
  state: async () => {
    const said = await vault.getVaultState({})
    return {
      id: said.id,
      name: said.name,
      path: said.path,
      scan: parseScan(said.scan),
      coverage: parseCoverage(said.coverage),
    }
  },
  agentUnreachable: async () => (await agentService.getAgentState({})).unreachable,
  watchVaultChanges: async function* (signal) {
    for await (const change of vault.watchVaultChanges({}, { signal })) {
      yield {
        paths: change.paths,
        shouldReload: change.reload,
        renamed: change.renamed.map((went) => ({ from: went.from, to: went.to })),
      }
    }
  },
  watchFocus: (signal) => workspace.watchFocus({}, { signal }),
  writeOpenTabs: async (open) => {
    await workspace.writeOpenTabs({ tabs: open.tabs.map((one) => ({ ...one })), front: open.front })
  },
  async *watchTasks(signal) {
    for await (const said of windowService.watchTasks({ window: WINDOW }, { signal })) {
      yield said.tasks.map((at) => ({
        id: at.id,
        label: at.doing,
        about: at.about,
        done: Number(at.done),
        total: Number(at.total),
        counting: counted[at.unit] ?? 'things',
        error: at.error,
        isAsked: at.isAsked,
      }))
    }
  },
  watchQuit: (signal) => windowService.watchQuit({ window: WINDOW }, { signal }),
  reportFlush: async (token, owed) => {
    await windowService.reportFlush({
      window: WINDOW,
      token,
      result: flushResults[owed ?? 'nothing'],
    })
  },
}
