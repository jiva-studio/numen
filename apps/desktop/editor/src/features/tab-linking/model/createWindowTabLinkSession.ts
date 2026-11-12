/**
 * The session of a window with the tab links of the core: it reports the linkable tabs whenever
 * they change, and cancels and resumes its watch as the page is hidden and shown again.
 */
import { ref, watch } from 'vue'
import { runInBackground } from '@numen/ui'
import { formatError } from '@/shared/api/errors'
import type { MovedTab, TabLinkGroup, TabPathSync, WindowTabs } from '@/entities/tab'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { TabLinksState } from '../types'
import { createTabLinkSession, getWindowReport } from './tabLinkSession'
import type { TabReports } from './tabReports'

export interface WindowTabLinkSessionDeps {
  readonly windowId: string
  readonly windowTabs: Pick<WindowTabs, 'tabs' | 'groups'>
  readonly isMain: boolean
  readonly tabLinks: Pick<TabLinksState, 'applySnapshot'>
  readonly reports: Pick<TabReports, 'reportTabs' | 'waitForReports' | 'stop' | 'resume'>
  readonly applySync: (sync: TabPathSync) => void
  readonly applyGroupPaths: (groups: readonly TabLinkGroup[]) => void
  readonly openMovedTab: (moved: MovedTab) => void
  readonly writeMessage: MessageWriter
}

export function createWindowTabLinkSession(deps: WindowTabLinkSessionDeps) {
  const { writeMessage } = deps
  /** Whether another window offers to move a tab into this one. */
  const isHighlighted = ref(false)
  const session = createTabLinkSession({
    ...deps,
    setHighlighted: (isOn) => {
      isHighlighted.value = isOn
    },
    onError: (error) =>
      writeMessage(`The tab links were not written: ${formatError(error)}`, 'error'),
  })
  watch(() => JSON.stringify(getWindowReport(deps.windowTabs, deps.isMain)), session.reportTabs)
  // Hiding the page cancels the watch in the core; a page restored from the back-forward cache
  // watches again.
  globalThis.addEventListener('pagehide', session.close)
  globalThis.addEventListener('pageshow', (event) => {
    if (!event.persisted) return
    runInBackground(session.reopen(), (error) =>
      writeMessage(`The tab links stopped: ${formatError(error)}`, 'error'),
    )
  })
  return { isHighlighted, start: session.start, close: session.close }
}
