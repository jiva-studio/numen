import type { PathRename } from '@/shared/paths'
import type { Task } from '@/shared/notices/task'
import type { OpenTabs } from '@/entities/tab'
import type { TabLinksChange, TabLinksSnapshot } from '@/features/tab-linking'

export interface VaultPort {
  getInitialOpenPath(): Promise<{ path: string } | null>
  state(): Promise<{
    /** The folder's identity, which looks up the vault. */
    id: string
    /** The user-visible vault name. */
    name: string
    path: string
    /** How far indexing the vault has got. */
    scan: {
      isReady: boolean
      error: string
      unwatchedPath: string
    }
    /** How far searching it by meaning has got. */
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
  reportFlush(token: string, outcome?: 'written' | 'asking'): Promise<void>
  /**
   * Opens a tab of a kind with an optional vault path in a new window at a screen point. The new
   * window reuses the tab id, so the tab keeps its links.
   */
  detachTab(
    tab: { readonly id: string; readonly kind: string; readonly path: string },
    position: { x: number; y: number },
  ): Promise<void>
  /**
   * Streams the link groups and the linkable tabs of all windows, then every change and every
   * path sync, until the signal aborts. When the stream ends, the core unlinks the tabs of this
   * window.
   */
  watchTabLinks(windowId: string, signal: AbortSignal): AsyncIterable<TabLinksEvent>
  /** Cancels the tab links stream of a window. */
  cancelTabLinksWatch(windowId: string): Promise<void>
  /** Links two tabs, unlinks a tab, or reports the linkable tabs of the window. */
  writeTabLinks(windowId: string, change: TabLinksChange | WindowTabsReport): Promise<void>
  /** Sends the new path of a linked tab to all windows, which navigate their linked tabs to it. */
  syncTabPath(sync: { sourceTabId: string; path: string; title: string }): Promise<void>
}

/** Linkable tabs open in a window. */
export interface WindowTabsReport {
  readonly tabs: readonly { readonly id: string; readonly kind: string; readonly title: string }[]
}

/** New path of a linked tab. */
export interface TabPathSync {
  readonly groupId: string
  readonly sourceTabId: string
  readonly path: string
  readonly title: string
}

/** Tab links stream message: a snapshot or a sync; a keepalive has neither. */
export interface TabLinksEvent {
  readonly links?: TabLinksSnapshot
  readonly sync?: TabPathSync
}
