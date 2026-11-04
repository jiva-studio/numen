/**
 * Syncs the path of linked tabs. When a tab navigates, the window moves its own
 * linked tabs and sends the sync to the core, which broadcasts it to all windows.
 * Every window applies the syncs in the order the core broadcasts them, its own
 * included, so all windows end on the path of the last sync.
 */
import type { useWindowTabs } from '@/entities/tab'
import type { TabId, TabLinksState } from '@/features/tab-linking'
import type { TabPathSync } from '../ports/vault'

export interface TabSyncDeps {
  windowTabs: Pick<ReturnType<typeof useWindowTabs>, 'getTab'>
  tabLinks: Pick<TabLinksState, 'getGroupOf' | 'groups'>
  /** Sends a path sync from a tab of this window to the other windows. */
  sendSync: (sync: { sourceTabId: string; path: string; title: string }) => Promise<void>
  /** Receives navigate failures and rejected syncs. */
  onError: (error: unknown) => void
}

export function createTabSync({ windowTabs, tabLinks, sendSync, onError }: TabSyncDeps) {
  let isSyncing = false
  /**
   * Path each tab was moved to by a sync. The next report of that path from the
   * tab is an echo: it consumes the entry and is not synced.
   */
  const echoPaths = new Map<TabId, string>()
  /** Path each tab of this window shows, from its last report or its last move. */
  const shownPaths = new Map<TabId, string>()

  /** Forgets a move to the path, after the tab failed to navigate to it. */
  const forgetMove = (tabId: TabId, path: string) => {
    if (echoPaths.get(tabId) === path) echoPaths.delete(tabId)
    if (shownPaths.get(tabId) === path) shownPaths.delete(tabId)
  }

  /** Navigates one tab of this window; forgets the move when navigate fails. */
  const moveTab = (tabId: TabId, path: string, title: string) => {
    const target = windowTabs.getTab(tabId)
    if (!target?.kind.navigate) return
    echoPaths.set(tabId, path)
    shownPaths.set(tabId, path)
    try {
      const navigation = target.kind.navigate(target.state, path, title)
      if (navigation) {
        navigation.catch((error: unknown) => {
          forgetMove(tabId, path)
          onError(error)
        })
      }
    } catch (error) {
      forgetMove(tabId, path)
      throw error
    }
  }

  /** Navigates each of the tabs of this window that does not show the path. */
  const moveTabs = (tabIds: readonly TabId[], path: string, title: string) => {
    isSyncing = true
    try {
      for (const tabId of tabIds) {
        if (shownPaths.get(tabId) !== path) moveTab(tabId, path, title)
      }
    } finally {
      isSyncing = false
    }
  }

  /**
   * Moves every tab linked to the source to the path. Returns whether the source
   * is linked. An echo of a sync this window applied moves nothing.
   */
  const syncPathFromTab = (sourceTabId: TabId, path: string, title = ''): boolean => {
    if (!sourceTabId || !path) return false
    shownPaths.set(sourceTabId, path)
    const group = tabLinks.getGroupOf(sourceTabId)
    if (!group || group.tabs.length < 2) return false
    const expected = echoPaths.get(sourceTabId)
    echoPaths.delete(sourceTabId)
    if (expected === path) return true
    if (isSyncing) return false
    moveTabs(
      group.tabs.filter((id) => id !== sourceTabId),
      path,
      title,
    )
    sendSync({ sourceTabId, path, title }).catch(onError)
    return true
  }

  /**
   * Applies a sync from the core to the tabs of this window in the group, the
   * source included. A later local move is undone, and every window ends on the
   * path of the last sync of the core.
   */
  const applySync = (sync: TabPathSync): void => {
    const group = tabLinks.groups.get(sync.groupId)
    if (group) moveTabs(group.tabs, sync.path, sync.title)
  }

  return { syncPathFromTab, applySync }
}
