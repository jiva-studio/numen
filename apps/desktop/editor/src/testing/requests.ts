/**
 * The requests the window made to the application, in order.
 *
 * Every mock records its arguments here, and a test reads them back.
 */
import type { Tab } from '@/entities/tab'

export const requests = {
  made: [] as string[],
  renamed: [] as string[],
  removed: [] as string[],
  moved: [] as string[],
  folders: [] as string[],
  /** The settings written, each as the path and the value of one edit. */
  settingsWritten: [] as string[],
  /** How often an open editor was requested to remeasure. */
  measured: 0,
  /** Every key the window handed down to the book it is showing. */
  pressed: [] as string[],
  /** The vaults the window requested to show, in request order. */
  opened: [] as string[],
  /** How many tabs the last write of the open tabs held, as each vault was requested to open. */
  tabsWhenOpened: [] as number[],
  /** How often a folder was requested, which corresponds to a vault being added. */
  chose: 0,
  /** What the window reported as currently open, most recent last. */
  openTabs: [] as { tabs: readonly Tab[]; activeId: string }[],
}

/** No requests yet: the initial state of every test. */
export const forgetRequests = () => {
  requests.made = []
  requests.renamed = []
  requests.removed = []
  requests.moved = []
  requests.folders = []
  requests.settingsWritten = []
  requests.measured = 0
  requests.pressed = []
  requests.opened = []
  requests.tabsWhenOpened = []
  requests.chose = 0
  requests.openTabs = []
}
