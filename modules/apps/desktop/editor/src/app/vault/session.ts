/**
 * Session, lifecycle, and workspace domain methods for the window core.
 */
import { WINDOW } from './clients'
import { agentService, vault, windowService, workspace } from '@/shared/clients'
import { counted, owing } from './words'
import type { VaultPort } from '@/app/ports/vault'

export type SessionCore = Pick<
  VaultPort,
  'state' | 'agentUnreachable' | 'changes' | 'focus' | 'writeOpenTabs' | 'tasks' | 'quitting' | 'flushed'
>

export const sessionCore: SessionCore = {
  state: async () => {
    const said = await vault.getVaultState({})
    return {
      id: said.id,
      name: said.name,
      path: said.path,
      scan: {
        isReady: said.scan?.ready ?? false,
        ready: said.scan?.ready ?? false,
        failureReason: said.scan?.failed ?? '',
        failed: said.scan?.failed ?? '',
        unwatchedPath: said.scan?.unwatched ?? '',
        unwatched: said.scan?.unwatched ?? '',
      },
      coverage: {
        chunkCount: said.coverage?.chunkCount ?? 0n,
        embeddedCount: said.coverage?.embeddedCount ?? 0n,
        isEmbedding: said.coverage?.embedding ?? false,
        embedding: said.coverage?.embedding ?? false,
      },
    }
  },
  agentUnreachable: async () => (await agentService.getAgentState({})).unreachable,
  changes: async function* (signal) {
    for await (const change of vault.watchVaultChanges({}, { signal })) {
      yield {
        paths: change.paths,
        shouldReload: change.reload,
        reload: change.reload,
        renamed: change.renamed.map((went) => ({ from: went.from, to: went.to })),
      }
    }
  },
  focus: (signal) => workspace.watchFocus({}, { signal }),
  writeOpenTabs: async (open) => {
    await workspace.writeOpenTabs({ tabs: open.tabs.map((one) => ({ ...one })), front: open.front })
  },
  async *tasks(signal) {
    for await (const said of windowService.watchTasks({ window: WINDOW }, { signal })) {
      yield said.tasks.map((at) => ({
        id: at.id,
        doing: at.doing,
        about: at.about,
        done: Number(at.done),
        total: Number(at.total),
        counting: counted[at.unit] ?? 'things',
        failed: at.failed,
        asked: at.asked,
      }))
    }
  },
  quitting: (signal) => windowService.watchQuit({ window: WINDOW }, { signal }),
  flushed: async (token, owed) => {
    await windowService.reportFlush({ window: WINDOW, token, result: owing[owed ?? 'nothing'] })
  },
}
