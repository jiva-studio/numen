/**
 * The links between the tabs of one window and the tabs of the others, and the reports of its own
 * linkable tabs that the core links them by.
 */
import { formatError } from '@/shared/api/errors'
import type { WindowTabs } from '@/entities/tab'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { TabLinksApi } from '../api/tab-links'
import { getWindowReport } from './tabLinkSession'
import { createTabReports } from './tabReports'
import { useTabLinks } from './useTabLinks'

export interface TabLinkingDeps {
  readonly writeTabLinks: TabLinksApi['writeTabLinks']
  readonly windowId: string
  readonly windowTabs: Pick<WindowTabs, 'tabs' | 'groups'>
  readonly isMain: boolean
  readonly writeMessage: MessageWriter
}

export function createTabLinking(deps: TabLinkingDeps) {
  const { windowId, writeTabLinks, writeMessage } = deps
  const reports = createTabReports({
    writeTabLinks,
    windowId,
    getReport: () => getWindowReport(deps.windowTabs, deps.isMain),
    onError: (error) =>
      writeMessage(`The tab links were not written: ${formatError(error)}`, 'error'),
  })
  // The core ignores a link to a tab it has no report of, so a link or unlink waits for the
  // pending tab reports.
  const tabLinks = useTabLinks({
    write: (change) => reports.waitForReports().then(() => writeTabLinks(windowId, change)),
    onError: (error) => writeMessage(`The tabs were not linked: ${formatError(error)}`, 'error'),
  })
  return { tabLinks, reports }
}
