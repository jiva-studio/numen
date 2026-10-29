/**
 * Synchronizes active path selection across linked tabs in the window.
 */
import type { useWindowTabs } from '@/entities/tab'
import type { TabId, TabLinksState } from '@/features/tab-linking'

export interface TabSyncDeps {
  windowTabs: Pick<ReturnType<typeof useWindowTabs>, 'getTab'>
  tabLinks: Pick<TabLinksState, 'getLinkedTargets'>
  /** A linked tab that failed to move, reported to the window. */
  onError: (error: unknown) => void
}

export function createTabSync({ windowTabs, tabLinks, onError }: TabSyncDeps) {
  let isSyncing = false

  /** Every tab linked to the source moves to the path, through its kind's navigate. */
  const syncPathFromTab = (sourceTabId: TabId, path: string, title = ''): boolean => {
    if (!sourceTabId || !path || isSyncing) return false
    const targets = tabLinks.getLinkedTargets(sourceTabId)
    if (targets.length === 0) return false

    isSyncing = true
    try {
      for (const targetId of targets) {
        const target = windowTabs.getTab(targetId)
        const moved = target?.kind.navigate?.(target.state, path, title)
        if (moved) moved.catch(onError)
      }
      return true
    } finally {
      isSyncing = false
    }
  }

  return { syncPathFromTab }
}
