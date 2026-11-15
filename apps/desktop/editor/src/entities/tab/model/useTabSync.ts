/**
 * Syncs the path of linked tabs. When a tab navigates, the window sends the sync
 * to the core, which broadcasts it to all windows, and moves its own linked tabs.
 * Every window applies the syncs in the order the core broadcasts them, its own
 * included, so all windows end on the path of the last sync. Each snapshot
 * carries the path of each group, and a window moves its tabs that missed a
 * sync to it.
 */
import type { TabId, TabLinkGroup, TabPathSync } from '../lib/links'
import type { WindowTabs } from './windowTabs'
import type { Caught } from '@/shared/api/errors'

export interface TabSyncDeps {
  windowTabs: Pick<WindowTabs, 'tabs'>
  tabLinks: {
    getGroupOf(this: void, tabId: TabId): TabLinkGroup | null
    readonly groups: ReadonlyMap<string, TabLinkGroup>
  }
  sendSync: (sync: { sourceTabId: string; path: string; title: string }) => Promise<void>
  onError: (error: Caught) => void
}

function createTabMover({ windowTabs, onError }: Pick<TabSyncDeps, 'windowTabs' | 'onError'>) {
  /**
   * Path each tab was moved to by a sync. A report of that path from the tab
   * before its navigate settles is an echo: it consumes the entry and is not
   * synced.
   */
  const echoPaths = new Map<TabId, string>()
  /** Path each tab of this window shows, from its last report or its last move. */
  const shownPaths = new Map<TabId, string>()

  /** Returns the path a tab of this window shows, or '' when it is not known. */
  const getShownPath = (tabId: TabId): string => {
    const shownPath = shownPaths.get(tabId)
    if (shownPath !== undefined) return shownPath
    const tab = windowTabs.tabs.getTab(tabId)
    if (!tab) return ''
    return (
      tab.kind.address?.getOpenTab?.(tab.state).path ||
      tab.kind.address?.getTarget?.(tab.state).path ||
      ''
    )
  }

  /** Removes the echo of a tab and returns the path it expected, if any. */
  const takeEcho = (tabId: TabId): string | undefined => {
    const expected = echoPaths.get(tabId)
    echoPaths.delete(tabId)
    return expected
  }

  /** Forgets the echo of a move to the path, once the navigate has settled. */
  const forgetEcho = (tabId: TabId, path: string) => {
    if (echoPaths.get(tabId) === path) echoPaths.delete(tabId)
  }

  /** Forgets a move to the path, after the tab failed to navigate to it. */
  const forgetMove = (tabId: TabId, path: string, error: Caught) => {
    forgetEcho(tabId, path)
    if (shownPaths.get(tabId) === path) shownPaths.delete(tabId)
    onError(error)
  }

  /**
   * Navigates one tab of this window; forgets the move when navigate fails. The
   * echo is expected until the navigate settles.
   */
  const moveTab = (tabId: TabId, path: string, title: string) => {
    const target = windowTabs.tabs.getTab(tabId)
    if (!target?.kind.address?.navigate) return
    echoPaths.set(tabId, path)
    shownPaths.set(tabId, path)
    try {
      const navigation = target.kind.address.navigate(target.state, path, title)
      if (!navigation) {
        forgetEcho(tabId, path)
        return
      }
      navigation.then(
        () => forgetEcho(tabId, path),
        (error: Caught) => forgetMove(tabId, path, error),
      )
    } catch (error) {
      forgetMove(tabId, path, error)
    }
  }

  const setShownPath = (tabId: TabId, path: string): void => {
    shownPaths.set(tabId, path)
  }

  return { getShownPath, takeEcho, moveTab, setShownPath }
}

export function createTabSync({ windowTabs, tabLinks, sendSync, onError }: TabSyncDeps) {
  const mover = createTabMover({ windowTabs, onError })
  let isSyncing = false
  /**
   * Path each moved tab reported that differs from its target path. Each one is
   * synced after the moves end, so the group follows the redirect.
   */
  const redirects = new Map<TabId, { path: string; title: string }>()
  /**
   * Path of the last sync this window sent in each group, until the core sends
   * it back. A snapshot taken before the core received the sync carries an
   * older path, which does not move the tabs.
   */
  const sentPaths = new Map<string, string>()

  /**
   * Navigates each of the tabs of this window that does not show the path, then
   * syncs the paths the moved tabs reported instead.
   */
  const moveTabs = (tabIds: readonly TabId[], path: string, title: string) => {
    isSyncing = true
    try {
      for (const tabId of tabIds) {
        if (mover.getShownPath(tabId) !== path) mover.moveTab(tabId, path, title)
      }
    } finally {
      isSyncing = false
    }
    const reported = [...redirects]
    redirects.clear()
    for (const [tabId, redirect] of reported) syncPathFromTab(tabId, redirect.path, redirect.title)
  }

  /**
   * Sends the path to the core and moves every tab linked to the source to it.
   * Returns whether the source is linked. An echo of a sync this window applied
   * moves nothing.
   */
  const syncPathFromTab = (sourceTabId: TabId, path: string, title = ''): boolean => {
    if (!sourceTabId || !path) return false
    mover.setShownPath(sourceTabId, path)
    const group = tabLinks.getGroupOf(sourceTabId)
    if (!group || group.tabs.length < 2) return false
    const expected = mover.takeEcho(sourceTabId)
    if (expected === path) return true
    if (isSyncing) {
      if (expected === undefined) return false
      redirects.set(sourceTabId, { path, title })
      return true
    }
    sentPaths.set(group.id, path)
    sendSync({ sourceTabId, path, title }).catch((error: Caught) => {
      if (sentPaths.get(group.id) === path) sentPaths.delete(group.id)
      onError(error)
    })
    moveTabs(
      group.tabs.filter((id) => id !== sourceTabId),
      path,
      title,
    )
    return true
  }

  /**
   * Applies a sync from the core to the tabs of this window in the group, the
   * source included. A later local move is undone, and every window ends on the
   * path of the last sync of the core.
   */
  const applySync = (sync: TabPathSync): void => {
    sentPaths.delete(sync.groupId)
    const group = tabLinks.groups.get(sync.groupId)
    if (group) moveTabs(group.tabs, sync.path, sync.title)
  }

  /**
   * Moves the tabs of this window in each group of a snapshot to the path of the
   * group. A group whose sync from this window the core has not sent back yet
   * is skipped.
   */
  const applyGroupPaths = (groups: readonly TabLinkGroup[]): void => {
    for (const groupId of sentPaths.keys()) {
      if (!groups.some((group) => group.id === groupId)) sentPaths.delete(groupId)
    }
    for (const group of groups) {
      const sentPath = sentPaths.get(group.id)
      if (sentPath !== undefined) {
        if (sentPath !== group.path) continue
        sentPaths.delete(group.id)
      }
      if (group.path) moveTabs(group.tabs, group.path, group.title)
    }
  }

  return {
    syncPathFromTab,
    applySync,
    applyGroupPaths,
    /** Records the path a tab of this window was moved to outside a sync. */
    setShownPath: mover.setShownPath,
  }
}
