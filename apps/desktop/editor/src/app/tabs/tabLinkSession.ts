/**
 * The window's part in the tab links the core keeps for every window: it reads
 * the stream of them, reports the linkable tabs it has open, and leaves when it
 * closes.
 */
import { createFollower } from '@numen/ui'
import type { useWindowTabs } from '@/entities/tab'
import type { TabLinksState } from '@/features/tab-linking'
import type { TabPathSync, VaultPort, WindowTabsReport } from '../ports/vault'

export interface TabLinkSessionDeps {
  readonly core: Pick<VaultPort, 'watchTabLinks' | 'cancelTabLinksWatch' | 'writeTabLinks'>
  /** The id this window watches and writes under. */
  readonly windowId: string
  readonly tabLinks: Pick<TabLinksState, 'applySnapshot'>
  readonly applySync: (sync: TabPathSync) => void
  /** The linkable tabs the window has open. */
  readonly getOpenTabs: () => WindowTabsReport['tabs']
  readonly wait: (ms: number) => Promise<unknown>
  readonly onError: (error: unknown) => void
}

export function createTabLinkSession(deps: TabLinkSessionDeps) {
  const abortController = new AbortController()
  let isOpen = true
  let isNewReading = false

  const follows = createFollower({
    isOpen: () => isOpen,
    setLost: () => undefined,
    wait: deps.wait,
  })

  let lastReport: Promise<void> = Promise.resolve()

  /** Tells the core the linkable tabs the window has open now, after the reports before it. */
  const reportTabs = (): void => {
    const tabs = deps.getOpenTabs()
    lastReport = lastReport
      .then(() => deps.core.writeTabLinks(deps.windowId, { tabs }))
      .catch(deps.onError)
  }

  /** Reads the stream until the window closes; each new reading reports the tabs again. */
  const start = (): Promise<void> =>
    follows(
      () => {
        isNewReading = true
        return deps.core.watchTabLinks(deps.windowId, abortController.signal)
      },
      (event) => {
        if (isNewReading) {
          isNewReading = false
          reportTabs()
        }
        if (event.links) deps.tabLinks.applySnapshot(event.links)
        else if (event.sync) deps.applySync(event.sync)
      },
    )

  /** Ends the stream; the core takes the window's tabs out of their groups. */
  const close = (): void => {
    if (!isOpen) return
    isOpen = false
    abortController.abort()
    deps.core.cancelTabLinksWatch(deps.windowId).catch(deps.onError)
  }

  return { start, close, reportTabs }
}

/** The tabs of the window whose kind can move to a path, in the order the window lists them. */
export function getLinkableTabs(
  windowTabs: Pick<ReturnType<typeof useWindowTabs>, 'tabs' | 'getTab'>,
): WindowTabsReport['tabs'] {
  return windowTabs.tabs.value.flatMap((tab) => {
    const kind = windowTabs.getTab(tab.id)?.kind
    return kind?.navigate ? [{ id: tab.id, kind: kind.kind, title: tab.title }] : []
  })
}
