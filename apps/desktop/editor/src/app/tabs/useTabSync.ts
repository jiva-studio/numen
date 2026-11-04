/**
 * Syncs the path of linked tabs. When a tab navigates, the window moves its own
 * linked tabs and sends the sync to the core, which broadcasts it to all windows.
 */
import type { useWindowTabs } from '@/entities/tab'
import type { TabId, TabLinkGroup, TabLinksState } from '@/features/tab-linking'
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

  const clearEcho = (tabId: TabId, path: string) => {
    if (echoPaths.get(tabId) === path) echoPaths.delete(tabId)
  }

  /** Navigates one tab of this window; clears the expected echo when navigate fails. */
  const moveTab = (tabId: TabId, path: string, title: string) => {
    const target = windowTabs.getTab(tabId)
    if (!target?.kind.navigate) return
    echoPaths.set(tabId, path)
    try {
      const navigation = target.kind.navigate(target.state, path, title)
      if (navigation) {
        navigation.catch((error: unknown) => {
          clearEcho(tabId, path)
          onError(error)
        })
      }
    } catch (error) {
      clearEcho(tabId, path)
      throw error
    }
  }

  /** Navigates every tab of this window in the group, except the source, to the path. */
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
   * Moves every tab linked to the source to the path. Returns whether the source
   * is linked. An echo of a sync this window applied moves nothing.
   */
  const syncPathFromTab = (sourceTabId: TabId, path: string, title = ''): boolean => {
    if (!sourceTabId || !path) return false
    const group = tabLinks.getGroupOf(sourceTabId)
    if (!group || group.tabs.length < 2) return false
    const expected = echoPaths.get(sourceTabId)
    echoPaths.delete(sourceTabId)
    if (expected === path) return true
    if (isSyncing) return false
    moveOwnTabs(group, sourceTabId, path, title)
    sendSync({ sourceTabId, path, title }).catch(onError)
    return true
  }

  /** Applies a sync from another window to the tabs of this window in the group. */
  const applySync = (sync: TabPathSync): void => {
    if (windowTabs.getTab(sync.sourceTabId)) return
    const group = tabLinks.groups.get(sync.groupId)
    if (group) moveOwnTabs(group, sync.sourceTabId, sync.path, sync.title)
  }

  return { syncPathFromTab, applySync }
}
