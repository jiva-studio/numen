import type { Result } from '@/shared/lib/result'
import type {
  MovedTab,
  OpenTabs,
  Tab,
  TabLinksEvent,
  TabMoveError,
  WindowTabsReport,
} from '@/entities/tab'
import type { TabLinksChange } from '@/features/tab-linking'

export interface VaultPort {
  /** Reports the currently open tabs; called again whenever they change. */
  writeOpenTabs(open: OpenTabs): Promise<void>
  /** The tabs last written for the vault in the window, kept by the core across restarts. */
  readOpenTabs(): Promise<OpenTabs>
  /**
   * Opens a tab in a new window at a screen point, with a size in CSS pixels; a size of 0 keeps
   * the default size. The new window reads the tab and its saved state by id and reuses the id,
   * so the tab keeps its links. Resolves after the new window accepts the tab.
   */
  detachTab(
    tab: Tab,
    place: { x: number; y: number; width: number; height: number },
  ): Promise<Result<void, TabMoveError>>
  /**
   * Returns the tab of a pending detach to the first window that reads it, or null when no detach
   * of the tab is pending for the window.
   */
  readDetachedTab(windowId: string, tabId: string): Promise<MovedTab | null>
  /** Sends a tab to another open window. Resolves after the target window accepts the tab. */
  moveTab(windowId: string, targetWindowId: string, tab: Tab): Promise<Result<void, TabMoveError>>
  /** Reports that this window opened a moved tab. On an error the window closes the tab. */
  acceptTabMove(
    windowId: string,
    tabId: string,
    moveId: string,
  ): Promise<Result<void, TabMoveError>>
  /** Turns the outline of a window on or off. */
  highlightWindow(windowId: string, isOn: boolean): Promise<void>
  /**
   * Streams the link groups and the linkable tabs of all windows, then every change and every
   * path sync, until the signal aborts. When the last stream of this window ends, the core
   * unlinks the tabs of this window unless a new stream starts within a grace period.
   */
  watchTabLinks(windowId: string, signal: AbortSignal): AsyncIterable<TabLinksEvent>
  /** Cancels the tab links stream of a window. */
  cancelTabLinksWatch(windowId: string): Promise<void>
  /** Links two tabs, unlinks a tab, or reports the linkable tabs of the window. */
  writeTabLinks(windowId: string, change: TabLinksChange | WindowTabsReport): Promise<void>
  /** Sends the new path of a linked tab to all windows, which navigate their linked tabs to it. */
  syncTabPath(sync: { sourceTabId: string; path: string; title: string }): Promise<void>
}
