/**
 * Watches the tab links stream for this window, reports its linkable tabs,
 * opens tabs sent from other windows, and cancels the watch on close.
 */
import { createFollower } from '@numen/ui'
import type { Result } from '@/shared/lib/result'
import type {
  WindowTabs,
  TabLinkGroup,
  MovedTab,
  TabMoveError,
  TabPathSync,
  WindowTabsReport,
} from '@/entities/tab'
import { cancelTabLinksWatch, watchTabLinks, writeTabLinks } from '../api/tab-links'
import type { TabLinksState } from '../types'

/** How long a failed tab report waits before it is sent again. */
const REPORT_RETRY_MS = 2000

export interface TabLinkSessionDeps {
  readonly windowId: string
  readonly tabLinks: Pick<TabLinksState, 'applySnapshot'>
  readonly acceptTabMove: (
    windowId: string,
    tabId: string,
    moveId: string,
  ) => Promise<Result<void, TabMoveError>>
  readonly applySync: (sync: TabPathSync) => void
  readonly applyGroupPaths: (groups: readonly TabLinkGroup[]) => void
  readonly openMovedTab: (moved: MovedTab) => void
  readonly setHighlighted: (isOn: boolean) => void
  readonly getReport: () => WindowTabsReport
  readonly wait: (ms: number) => Promise<unknown>
  readonly onError: (error: unknown) => void
}

export function createTabLinkSession(deps: TabLinkSessionDeps) {
  let abortController = new AbortController()
  let isOpen = true
  let isNewStream = false

  const followStream = createFollower({
    isOpen: () => isOpen,
    setLost: () => undefined,
    wait: deps.wait,
  })

  /** The latest report not yet written to the core. */
  let pendingReport: WindowTabsReport | null = null
  let isReporting = false
  /** Settles when the running writeReports loop ends. */
  let reporting: Promise<void> = Promise.resolve()

  /**
   * Writes the pending report until none is left. A failed report is reported
   * once and sent again after REPORT_RETRY_MS, unless a newer report replaced
   * it. Stops while the session is closed.
   */
  const writeReports = async (): Promise<void> => {
    isReporting = true
    let hasFailed = false
    while (pendingReport && isOpen) {
      const report = pendingReport
      pendingReport = null
      try {
        await writeTabLinks(deps.windowId, report)
        hasFailed = false
      } catch (error) {
        if (!hasFailed) deps.onError(error)
        hasFailed = true
        pendingReport ??= report
        await deps.wait(REPORT_RETRY_MS)
      }
    }
    isReporting = false
  }

  /** Writes the current report to the core, after any pending report. */
  const reportTabs = (): void => {
    pendingReport = deps.getReport()
    if (!isReporting) reporting = writeReports().catch(deps.onError)
  }

  /** Settles when the pending tab reports are written. */
  const waitForReports = (): Promise<void> => reporting

  /**
   * Sends AcceptTabMove once the pending reports are written, so that a report
   * made before the tab opened does not reach the core after the accept.
   */
  const acceptMove = (tabId: string, moveId: string): Promise<Result<void, TabMoveError>> =>
    waitForReports().then(() => deps.acceptTabMove(deps.windowId, tabId, moveId))

  /** Reads the stream until close; reports the tabs again on each new stream. */
  const start = (): Promise<void> =>
    followStream(
      () => {
        isNewStream = true
        return watchTabLinks(deps.windowId, abortController.signal)
      },
      (event) => {
        if (isNewStream) {
          isNewStream = false
          reportTabs()
        }
        // An event that fails to apply is reported, and the stream stays open.
        try {
          if (event.links) {
            deps.tabLinks.applySnapshot(event.links)
            deps.applyGroupPaths(event.links.groups)
          } else if (event.sync) {
            deps.applySync(event.sync)
          } else if (event.movedTab) {
            deps.openMovedTab(event.movedTab)
          } else if (event.isHighlighted !== undefined) {
            deps.setHighlighted(event.isHighlighted)
          }
        } catch (error) {
          deps.onError(error)
        }
      },
    )

  /**
   * Aborts the stream and cancels the watch; the core unlinks the tabs of the
   * window when no watch of the window starts within its grace period.
   */
  const close = (): void => {
    if (!isOpen) return
    isOpen = false
    abortController.abort()
    cancelTabLinksWatch(deps.windowId).catch(deps.onError)
  }

  /** Starts the stream again after close; does nothing while the session is open. */
  const reopen = (): Promise<void> => {
    if (isOpen) return Promise.resolve()
    isOpen = true
    abortController = new AbortController()
    return start()
  }

  return { start, close, reopen, reportTabs, acceptMove, waitForReports }
}

type ReportedTabs = Pick<WindowTabs, 'tabs' | 'groups'>

/** Returns the tabs whose kind supports navigate, in tab order. */
function getLinkableTabs(windowTabs: Pick<ReportedTabs, 'tabs'>): WindowTabsReport['tabs'] {
  return windowTabs.tabs.tabs.value.flatMap((tab) => {
    const kind = windowTabs.tabs.getTab(tab.id)?.kind
    return kind?.navigate ? [{ id: tab.id, kind: kind.kind, title: tab.title }] : []
  })
}

/** Returns the linkable tabs, the active tab and the tab count of the window. */
export function getWindowReport(windowTabs: ReportedTabs, isMain: boolean): WindowTabsReport {
  const front = windowTabs.groups.handle.front()
  const activeTitle = windowTabs.tabs.tabs.value.find((tab) => tab.id === front?.id)?.title ?? ''
  return {
    tabs: getLinkableTabs(windowTabs),
    window: {
      isMain,
      activeTitle,
      activeKind: front?.kind ?? '',
      tabCount: windowTabs.tabs.tabs.value.length,
    },
  }
}
