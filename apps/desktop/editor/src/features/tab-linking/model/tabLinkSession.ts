/**
 * Watches the tab links stream for this window, reports its linkable tabs,
 * opens tabs sent from other windows, and cancels the watch on close.
 */
import { createFollower } from '@/shared/api/stream'
import type {
  WindowTabs,
  TabLinkGroup,
  MovedTab,
  TabPathSync,
  WindowTabsReport,
} from '@/entities/tab'
import type { TabLinksApi } from '../api/tab-links'
import type { TabLinksState } from '../types'
import type { TabReports } from './tabReports'
import { wait } from '@/shared/model/wait'

/** What the stream of the tab links applies to the tabs of this window. */
export interface TabPathApplier {
  readonly applySync: (sync: TabPathSync) => void
  readonly applyGroupPaths: (groups: readonly TabLinkGroup[]) => void
}

export interface TabLinkSessionDeps {
  readonly api: Pick<TabLinksApi, 'watchTabLinks' | 'cancelTabLinksWatch'>
  readonly windowId: string
  readonly tabLinks: Pick<TabLinksState, 'applySnapshot'>
  readonly tabSync: TabPathApplier
  readonly openMovedTab: (moved: MovedTab) => void
  readonly setHighlighted: (isOn: boolean) => void
  readonly reports: Pick<TabReports, 'reportTabs' | 'waitForReports' | 'stop' | 'resume'>
  readonly wait?: (ms: number) => Promise<void>
  readonly onError: (error: unknown) => void
}

export function createTabLinkSession(deps: TabLinkSessionDeps) {
  let abortController = new AbortController()
  let isOpen = true
  let isNewStream = false

  const followStream = createFollower({
    isOpen: () => isOpen,
    setLost: () => undefined,
    wait: deps.wait ?? wait,
  })

  const { reportTabs, waitForReports } = deps.reports

  /** Reads the stream until close; reports the tabs again on each new stream. */
  const start = (): Promise<void> =>
    followStream(
      () => {
        isNewStream = true
        return deps.api.watchTabLinks(deps.windowId, abortController.signal)
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
            deps.tabSync.applyGroupPaths(event.links.groups)
          } else if (event.sync) {
            deps.tabSync.applySync(event.sync)
          } else if (event.tabMove) {
            deps.openMovedTab(event.tabMove)
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
    deps.reports.stop()
    abortController.abort()
    deps.api.cancelTabLinksWatch(deps.windowId).catch(deps.onError)
  }

  /** Starts the stream again after close; does nothing while the session is open. */
  const reopen = (): Promise<void> => {
    if (isOpen) return Promise.resolve()
    isOpen = true
    deps.reports.resume()
    abortController = new AbortController()
    return start()
  }

  return { start, close, reopen, reportTabs, waitForReports }
}

type ReportedTabs = Pick<WindowTabs, 'tabs' | 'handle'>

function getLinkableTabs(windowTabs: Pick<ReportedTabs, 'tabs'>): WindowTabsReport['tabs'] {
  return windowTabs.tabs.tabs.value.flatMap((tab) => {
    const kind = windowTabs.tabs.getTab(tab.id)?.kind
    return kind?.address?.navigate ? [{ id: tab.id, kind: kind.kind, title: tab.title }] : []
  })
}

/** Returns the linkable tabs, the active tab and the tab count of the window. */
export function getWindowReport(windowTabs: ReportedTabs, isMain: boolean): WindowTabsReport {
  const front = windowTabs.handle.getFrontTab()
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
