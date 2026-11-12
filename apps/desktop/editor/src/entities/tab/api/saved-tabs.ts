/**
 * The tabs the window keeps open across restarts.
 */
import { workspace } from '@/shared/api/clients'
import type { OpenTabs } from '../lib/tab'

/** Reports the currently open tabs; called again whenever they change. */
export const writeOpenTabs = async (open: OpenTabs): Promise<void> => {
  await workspace.writeOpenTabs({
    tabs: open.tabs.map((one) => ({ ...one })),
    activeId: open.activeId,
  })
}

/** The tabs last written for the vault in the window, kept by the core across restarts. */
export const readOpenTabs = async (): Promise<OpenTabs> => {
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
}
