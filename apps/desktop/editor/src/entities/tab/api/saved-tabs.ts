/**
 * The tabs the window keeps open across restarts.
 */
import type { WindowClients } from '@/shared/api/clients'
import type { OpenTabs } from '../lib/tab'

/** The open tabs the core keeps for the window, over `workspace`. */
export const createSavedTabsApi = (workspace: WindowClients['workspace']) => ({
  /** Reports the currently open tabs; called again whenever they change. */
  writeOpenTabs: async (open: OpenTabs): Promise<void> => {
    await workspace.writeOpenTabs({
      tabs: open.tabs.map((one) => ({ ...one })),
      activeId: open.activeId,
    })
  },

  /** The tabs last written for the vault in the window, kept by the core across restarts. */
  readOpenTabs: async (): Promise<OpenTabs> => {
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
})

export type SavedTabsApi = ReturnType<typeof createSavedTabsApi>
