/**
 * Reports the linkable tabs of the window to the core, one report at a time, and lets whatever
 * writes a link wait until the pending report is written: the core ignores a link to a tab it has
 * no report of.
 */
import type { WindowTabsReport } from '@/entities/tab'
import type { TabLinksApi } from '../api/tab-links'
import type { Caught } from '@/shared/api/errors'
import { wait } from '@/shared/model/wait'

/** How long a failed tab report waits before it is sent again. */
const REPORT_RETRY_MS = 2000

export interface TabReportsDeps {
  readonly writeTabLinks: TabLinksApi['writeTabLinks']
  readonly windowId: string
  readonly getReport: () => WindowTabsReport
  readonly wait?: (ms: number) => Promise<void>
  readonly onError: (error: Caught) => void
}

export function createTabReports(deps: TabReportsDeps) {
  let isOpen = true
  /** The latest report not yet written to the core. */
  let pendingReport: WindowTabsReport | null = null
  let isReporting = false
  /** Settles when the running writeReports loop ends. */
  let reporting: Promise<void> = Promise.resolve()

  /**
   * Writes the pending report until none is left. A failed report is reported
   * once and sent again after REPORT_RETRY_MS, unless a newer report replaced
   * it. Stops while the reports are stopped.
   */
  const writeReports = async (): Promise<void> => {
    isReporting = true
    let hasFailed = false
    while (pendingReport && isOpen) {
      const report = pendingReport
      pendingReport = null
      try {
        await deps.writeTabLinks(deps.windowId, report)
        hasFailed = false
      } catch (error) {
        if (!hasFailed) deps.onError(error)
        hasFailed = true
        pendingReport ??= report
        await (deps.wait ?? wait)(REPORT_RETRY_MS)
      }
    }
    isReporting = false
  }

  return {
    /** Writes the current report to the core, after any pending report. */
    reportTabs: (): void => {
      pendingReport = deps.getReport()
      if (!isReporting) reporting = writeReports().catch(deps.onError)
    },
    /** Settles when the pending tab reports are written. */
    waitForReports: (): Promise<void> => reporting,
    stop: (): void => {
      isOpen = false
    },
    resume: (): void => {
      isOpen = true
    },
  }
}

export type TabReports = ReturnType<typeof createTabReports>
