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
  /**
   * The path each tab was moved to by a sync. The tab reporting that path next
   * is the echo of the move, and the entry is used up by it.
   */
  const movedTo = new Map<TabId, string>()

  const forgetMove = (tabId: TabId, path: string) => {
    if (movedTo.get(tabId) === path) movedTo.delete(tabId)
  }

  /** One tab of this window moved through its kind's navigate; a failed move is forgotten. */
  const moveTab = (tabId: TabId, path: string, title: string) => {
    const target = windowTabs.getTab(tabId)
    if (!target?.kind.navigate) return
    movedTo.set(tabId, path)
    try {
      const moved = target.kind.navigate(target.state, path, title)
      if (moved) {
        moved.catch((error: unknown) => {
          forgetMove(tabId, path)
          onError(error)
        })
      }
    } catch (error) {
      forgetMove(tabId, path)
      throw error
    }
  }

  /** Every tab of this window in the group, other than the source, moved to the path. */
  const moveOwnTabs = (group: TabLinkGroup, sourceTabId: TabId, path: string, title: string) => {
    isSyncing = true
    try {
      for (const targetId of group.tabs) {
        if (targetId !== sourceTabId) moveTab(targetId, path, title)
      }
    } finally {
      isSyncing = false
    }
  }

  /**
   * Moves every tab linked to the source to the path. It answers whether the
   * source is linked; the echo of a move this window made moves nothing.
   */
  const syncPathFromTab = (sourceTabId: TabId, path: string, title = ''): boolean => {
    if (!sourceTabId || !path) return false
    const group = tabLinks.getGroupOf(sourceTabId)
    if (!group || group.tabs.length < 2) return false
    const expected = movedTo.get(sourceTabId)
    movedTo.delete(sourceTabId)
    if (expected === path) return true
    if (isSyncing) return false
    moveOwnTabs(group, sourceTabId, path, title)
    sendSync({ sourceTabId, path, title }).catch(onError)
    return true
  }

  /** A move another window sent: this window's tabs of the group move. */
  const applySync = (sync: TabPathSync): void => {
    if (windowTabs.getTab(sync.sourceTabId)) return
    const group = tabLinks.groups.get(sync.groupId)
    if (group) moveOwnTabs(group, sync.sourceTabId, sync.path, sync.title)
  }

  return { syncPathFromTab, applySync }
}
