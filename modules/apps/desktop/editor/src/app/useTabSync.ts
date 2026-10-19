/**
 * Synchronizes active path selection across linked tabs in the window.
 */
import type { useWindowTabs } from '@/entities/tab'
import type { TabId, TabLinksState } from '@/features/tab-linking'

export interface TabSyncDeps {
  held: Pick<ReturnType<typeof useWindowTabs>, 'getTab'>
  tabLinks: Pick<TabLinksState, 'getLinkedTargets'>
}

export function isNotePath(path: string): boolean {
  return path.endsWith('.md') || !path.includes('.')
}

export function useTabSync({ held, tabLinks }: TabSyncDeps) {
  let isSyncing = false

  /** Every tab linked to the source moves to the path, through its kind's navigate. */
  const syncPathFromTab = (sourceTabId: TabId, path: string, title = ''): boolean => {
    if (!sourceTabId || !path || isSyncing) return false
    const targets = tabLinks.getLinkedTargets(sourceTabId)
    if (targets.length === 0) return false

    isSyncing = true
    try {
      for (const targetId of targets) {
        const target = held.getTab(targetId)
        if (target) void target.kind.navigate?.(target.state, path, title)
      }
      return true
    } finally {
      isSyncing = false
    }
  }

  return { syncPathFromTab }
}
