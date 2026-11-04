/**
 * Watches the tab links stream for this window, reports its linkable tabs, and
 * cancels the watch on close.
 */
import { createFollower } from '@numen/ui'
import type { useWindowTabs } from '@/entities/tab'
import type { TabLinksState } from '@/features/tab-linking'
import type { TabPathSync, VaultPort, WindowTabsReport } from '../ports/vault'

export interface TabLinkSessionDeps {
  readonly core: Pick<VaultPort, 'watchTabLinks' | 'cancelTabLinksWatch' | 'writeTabLinks'>
  /** Window id used for the watch and for writes. */
  readonly windowId: string
  readonly tabLinks: Pick<TabLinksState, 'applySnapshot'>
  readonly applySync: (sync: TabPathSync) => void
  /** Returns the linkable tabs open in the window. */
  readonly getOpenTabs: () => WindowTabsReport['tabs']
  readonly wait: (ms: number) => Promise<unknown>
  readonly onError: (error: unknown) => void
}

export function createTabLinkSession(deps: TabLinkSessionDeps) {
  const abortController = new AbortController()
  let isOpen = true
  let isNewStream = false

  const followStream = createFollower({
    isOpen: () => isOpen,
    setLost: () => undefined,
    wait: deps.wait,
  })

  let lastReport: Promise<void> = Promise.resolve()

  /** Writes the current linkable tabs to the core, after any pending report. */
  const reportTabs = (): void => {
    const tabs = deps.getOpenTabs()
    lastReport = lastReport
      .then(() => deps.core.writeTabLinks(deps.windowId, { tabs }))
      .catch(deps.onError)
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
        if (event.links) deps.tabLinks.applySnapshot(event.links)
        else if (event.sync) deps.applySync(event.sync)
      },
    )

  /** Aborts the stream and cancels the watch; the core then unlinks the tabs of the window. */
  const close = (): void => {
    if (!isOpen) return
    isOpen = false
    abortController.abort()
    deps.core.cancelTabLinksWatch(deps.windowId).catch(deps.onError)
  }

  return { start, close, reportTabs }
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
