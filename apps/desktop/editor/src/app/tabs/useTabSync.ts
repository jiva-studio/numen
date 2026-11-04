/**
 * Moves linked tabs to the path one of them moved to. A window moves its own
 * tabs and sends the move to the core, which passes it to every window.
 */
import type { useWindowTabs } from '@/entities/tab'
import type { TabId, TabLinkGroup, TabLinksState } from '@/features/tab-linking'
import type { TabPathSync } from '../ports/vault'

export interface TabSyncDeps {
  windowTabs: Pick<ReturnType<typeof useWindowTabs>, 'getTab'>
  tabLinks: Pick<TabLinksState, 'getGroupOf' | 'groups'>
  /** Sends a move of a tab of this window to the other windows. */
  sendSync: (sync: { sourceTabId: string; path: string; title: string }) => Promise<void>
  /** A linked tab that failed to move, or a move the core did not take. */
  onError: (error: unknown) => void
}

export function createTabSync({ windowTabs, tabLinks, sendSync, onError }: TabSyncDeps) {
  let isSyncing = false
  /** The path each group last moved to; a repeat of it is an echo and moves nothing. */
  const lastPaths = new Map<string, string>()

  /** Every tab of this window in the group, other than the source, moved through its kind's navigate. */
  const moveOwnTabs = (group: TabLinkGroup, sourceTabId: TabId, path: string, title: string) => {
    lastPaths.set(group.id, path)
    isSyncing = true
    try {
      for (const targetId of group.tabs) {
        if (targetId === sourceTabId) continue
        const target = windowTabs.getTab(targetId)
        const moved = target?.kind.navigate?.(target.state, path, title)
        if (moved) moved.catch(onError)
      }
    } finally {
      isSyncing = false
    }
  }

  /**
   * Moves every tab linked to the source to the path. It answers whether the
   * source is linked; a repeat of the path the group last moved to moves nothing.
   */
  const syncPathFromTab = (sourceTabId: TabId, path: string, title = ''): boolean => {
    if (!sourceTabId || !path || isSyncing) return false
    const group = tabLinks.getGroupOf(sourceTabId)
    if (!group || group.tabs.length < 2) return false
    if (lastPaths.get(group.id) === path) return true
    moveOwnTabs(group, sourceTabId, path, title)
    sendSync({ sourceTabId, path, title }).catch(onError)
    return true
  }

  /** A move another window sent: this window's tabs of the group move, unless it is an echo. */
  const applySync = (sync: TabPathSync): void => {
    if (windowTabs.getTab(sync.sourceTabId)) return
    const group = tabLinks.groups.get(sync.groupId)
    if (!group || lastPaths.get(group.id) === sync.path) return
    moveOwnTabs(group, sync.sourceTabId, sync.path, sync.title)
  }

  return { syncPathFromTab, applySync }
}
