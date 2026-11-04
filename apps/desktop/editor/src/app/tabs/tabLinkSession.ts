/**
 * Watches the tab links stream for this window, reports its linkable tabs, and
 * cancels the watch on close.
 */
import { createFollower } from '@numen/ui'
import type { useWindowTabs } from '@/entities/tab'
import type { TabLinkGroup, TabLinksState } from '@/features/tab-linking'
import type { TabPathSync, VaultPort, WindowTabsReport } from '../ports/vault'

/** How long a failed tab report waits before it is sent again. */
export const REPORT_RETRY_MS = 2000

export interface TabLinkSessionDeps {
  readonly core: Pick<VaultPort, 'watchTabLinks' | 'cancelTabLinksWatch' | 'writeTabLinks'>
  /** Window id used for the watch and for writes. */
  readonly windowId: string
  readonly tabLinks: Pick<TabLinksState, 'applySnapshot'>
  readonly applySync: (sync: TabPathSync) => void
  /** Moves the tabs of this window to the path of each group of a snapshot. */
  readonly applyGroupPaths: (groups: readonly TabLinkGroup[]) => void
  /** Returns the linkable tabs open in the window. */
  readonly getOpenTabs: () => WindowTabsReport['tabs']
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

  /** The latest tab report not yet written to the core. */
  let pendingTabs: WindowTabsReport['tabs'] | null = null
  let isReporting = false

  /**
   * Writes the pending report until none is left. A failed report is reported
   * once and sent again after REPORT_RETRY_MS, unless a newer report replaced
   * it. Stops while the session is closed.
   */
  const writeReports = async (): Promise<void> => {
    isReporting = true
    let hasFailed = false
    while (pendingTabs && isOpen) {
      const tabs = pendingTabs
      pendingTabs = null
      try {
        await deps.core.writeTabLinks(deps.windowId, { tabs })
        hasFailed = false
      } catch (error) {
        if (!hasFailed) deps.onError(error)
        hasFailed = true
        pendingTabs ??= tabs
        await deps.wait(REPORT_RETRY_MS)
      }
    }
    isReporting = false
  }

  /** Writes the current linkable tabs to the core, after any pending report. */
  const reportTabs = (): void => {
    pendingTabs = deps.getOpenTabs()
    if (!isReporting) writeReports().catch(deps.onError)
  }

  /** Reads the stream until close; reports the tabs again on each new stream. */
  const start = (): Promise<void> =>
    followStream(
      () => {
        isNewStream = true
        return deps.core.watchTabLinks(deps.windowId, abortController.signal)
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
    deps.core.cancelTabLinksWatch(deps.windowId).catch(deps.onError)
  }

  /** Starts the stream again after close; does nothing while the session is open. */
  const reopen = (): Promise<void> => {
    if (isOpen) return Promise.resolve()
    isOpen = true
    abortController = new AbortController()
    return start()
  }

  return { start, close, reopen, reportTabs }
}

/** Returns the tabs whose kind supports navigate, in tab order. */
export function getLinkableTabs(
  windowTabs: Pick<ReturnType<typeof useWindowTabs>, 'tabs' | 'getTab'>,
): WindowTabsReport['tabs'] {
  return windowTabs.tabs.value.flatMap((tab) => {
    const kind = windowTabs.getTab(tab.id)?.kind
    return kind?.navigate ? [{ id: tab.id, kind: kind.kind, title: tab.title }] : []
  })
}
