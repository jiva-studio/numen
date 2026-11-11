import type { Notes } from '@/shared/model/command'
import type { WindowTabs } from '@/entities/tab'

type TabsOfWindow = Pick<WindowTabs, 'tabs' | 'panes'>

function tabMatchesPath(
  windowTab: ReturnType<TabsOfWindow['tabs']['getTab']>,
  path: string,
): boolean {
  if (!windowTab) return false
  const openPath = windowTab.kind.address?.getOpenTab?.(windowTab.state).path
  if (openPath === path) return true
  const target = windowTab.kind.address?.getTarget?.(windowTab.state)
  return target?.file === path || target?.path === path
}

function findActiveTabAt(windowTabs: TabsOfWindow, path: string): string | null {
  if (!path) return null
  for (const tab of windowTabs.tabs.tabs.value) {
    if (tabMatchesPath(windowTabs.tabs.getTab(tab.id), path)) return tab.id
  }
  return null
}

/**
 * The open files a command can access: those held by the note editors, and those
 * a tab kind serves.
 */
export function createCommandNotes(reached: Notes, windowTabs: TabsOfWindow): Notes {
  const getTab = (id: string) => windowTabs.tabs.getTab(id)
  return {
    openFile: (path, title, placement, place) => reached.openFile(path, title, placement, place),
    getTabAt: (path: string) => reached.getTabAt(path) ?? findActiveTabAt(windowTabs, path),
    getPath: (id: string) => {
      const path = reached.getPath(id)
      if (path !== id) return path
      const tab = getTab(id)
      return tab?.kind.address?.getOpenTab?.(tab.state).path ?? id
    },
    asking: (id: string) => {
      const tab = getTab(id)
      return reached.asking(id) || (tab?.kind.events?.isAsking?.(tab.state) ?? false)
    },
    flushPending: async (id: string) => {
      await reached.flushPending(id)
      const tab = getTab(id)
      await tab?.kind.events?.flushPending?.(tab.state)
    },
    close: (id: string) => {
      reached.close(id)
      windowTabs.panes.requestClose(id)
    },
  }
}
