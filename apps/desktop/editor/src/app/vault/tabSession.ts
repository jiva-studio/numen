/**
 * The tabs the window keeps open across restarts, and the detach of a tab into a new window.
 */
import { workspace } from '@/shared/api/clients'
import { parseMoveError } from './tabLinks'
import type { VaultPort } from '@/app/ports/vault'

type TabSessionCore = Pick<VaultPort, 'writeOpenTabs' | 'readOpenTabs' | 'detachTab'>

export const tabSessionCore: TabSessionCore = {
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
  detachTab: async (tab, place) => {
    const { error } = await workspace.detachTab({
      tab: { ...tab },
      x: Math.round(place.x),
      y: Math.round(place.y),
      width: Math.round(place.width),
      height: Math.round(place.height),
    })
    return parseMoveError(error)
  },
}
