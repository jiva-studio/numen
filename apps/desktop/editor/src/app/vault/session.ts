/**
 * Session, lifecycle, and workspace domain methods for the window core.
 */
import { WINDOW } from './clients'
import { agentService, vault, windowService, workspace } from '@/shared/clients'
import { counted, flushResults } from './words'
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

/** How far the scan of the vault has got. */
const parseScan = (scan: ScanState | undefined) => ({
  isReady: scan?.isReady ?? false,
  error: scan?.error ?? '',
  unwatchedPath: scan?.watchError ?? '',
})

/** How much of the vault the index holds. */
const parseCoverage = (coverage: CoverageState | undefined) => ({
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
  | 'readOpenTabs'
  | 'watchTasks'
  | 'watchQuit'
  | 'reportFlush'
  | 'detachTab'
>

export const sessionCore: SessionCore = {
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
      }
    }
  },
  watchFocus: (signal) => workspace.watchFocus({}, { signal }),
  writeOpenTabs: async (open) => {
    await workspace.writeOpenTabs({
      tabs: open.tabs.map((one) => ({ ...one })),
      activeId: open.activeId,
    })
  },
  readOpenTabs: async () => {
    const answer = await workspace.readOpenTabs({})
    return {
      activeId: answer.activeId,
      tabs: answer.tabs.map(({ id, kind, path, title, state }) => ({
        id,
        kind,
        path,
        title,
        state,
      })),
    }
  },
  async *watchTasks(signal) {
    for await (const update of windowService.watchTasks({ window: WINDOW }, { signal })) {
      yield update.tasks.map((task) => ({
        id: task.id,
        label: task.title,
        about: task.about,
        completedCount: Number(task.done),
        total: Number(task.total),
        counting: counted[task.unit] ?? 'things',
        error: task.error,
        isUserStarted: task.isUserStarted,
        pluginId: task.pluginId,
        pluginName: task.pluginName,
      }))
    }
  },
  watchQuit: (signal) => windowService.watchQuit({ window: WINDOW }, { signal }),
  reportFlush: async (token, outcome) => {
    await windowService.reportFlush({
      window: WINDOW,
      token,
      result: flushResults[outcome ?? 'nothing'],
    })
  },
  detachTab: async (tabId, position) => {
    await workspace.detachTab({ tabId, x: position.x, y: position.y })
  },
}
