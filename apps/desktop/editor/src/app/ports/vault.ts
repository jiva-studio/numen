import type { PathRename } from '@/shared/paths'
import type { Task } from '@/shared/notices/task'
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
  getInitialOpenPath(): Promise<{ path: string } | null>
  state(): Promise<{
    id: string
    name: string
    path: string
    scan: {
      isReady: boolean
      error: string
      unwatchedPath: string
    }
    coverage: {
      chunkCount: bigint
      embeddedCount: bigint
      isEmbedding: boolean
    }
  }>
  /**
   * Why an agent cannot be reached, or an empty string when it can. Which agent answers is
   * determined by the installation, so the check is made on the agent.
   */
  agentUnreachable(): Promise<string>
  watchVaultChanges(signal: AbortSignal): AsyncIterable<{
    paths: string[]
    shouldReload: boolean
    renamed: readonly PathRename[]
    versionPaths?: string[]
  }>
  /**
   * Everything the application is doing in the background, until the caller aborts the signal.
   *
   * The whole list is sent whenever any of it changes, and the first list is sent immediately.
   * It is a stream because work can begin without a request from the window: for example an
   * agent reading a document, which the user sees here.
   */
  watchTasks(signal: AbortSignal): AsyncIterable<readonly Task[]>
  /**
   * Asks the core to stop a task that `watchTasks` lists as cancellable. It resolves once the
   * core has cancelled the work; the task leaves the list when the work has stopped. It rejects
   * for a task that is not listed or cannot be cancelled.
   */
  cancelTask(id: string): Promise<void>
  /**
   * The places another component requested to display: a source and the ranges of its text,
   * counted in bytes. A length of zero names the source and no place inside it.
   */
  watchFocus(signal: AbortSignal): AsyncIterable<{
    path: string
    ranges: readonly { start: number; end: number }[]
    tabId: string
  }>
  /** Reports the currently open tabs; called again whenever they change. */
  writeOpenTabs(open: OpenTabs): Promise<void>
  /** The tabs last written for the vault in the window, kept by the core across restarts. */
  readOpenTabs(): Promise<OpenTabs>
  /**
   * The window quitting, until the client aborts the signal. The stream opens with the token this
   * client replies under.
   */
  watchQuit(signal: AbortSignal): AsyncIterable<{ token: string; flush: boolean }>
  /** Reports that this client has written all pending data. */
  reportFlush(token: string, outcome?: 'written' | 'unsaved'): Promise<void>
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
