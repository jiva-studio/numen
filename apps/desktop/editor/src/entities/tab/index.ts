/** The tabs a window keeps: what kinds there are, what opens one, and what each is called. */
export { fileOpeners } from './model/openers'
export type { FileOpeners } from './model/openers'
export type { Tab } from './lib/tab'
export type { Writable } from './lib/writable'
export type {
  LinkableTab,
  MovedTab,
  TabId,
  TabLinkGroup,
  TabLinksChange,
  TabLinksEvent,
  TabLinksSnapshot,
  TabMoveError,
  TabPathSync,
  TabWindow,
  WindowTabsReport,
} from './lib/links'
export * from './lib/kinds'
export { getAllTabs, getEligibleTabs } from './lib/eligibility'
export { getWindowName } from './lib/windowName'
export { createWindowTabs } from './model/windowTabs'
export type { WindowTabs } from './model/windowTabs'
export { formatSavedState, parseSavedTab, parseSavedTabs, useOpenTabs } from './model/useOpenTabs'
export { createTabSync } from './model/useTabSync'
export { createTabNavigation } from './model/tabNavigation'
export { useWindowTabMenu } from './model/useWindowTabMenu'
export { SETTINGS } from './lib/workspace'
export { readOpenTabs } from './api/saved-tabs'
export { default as RestoreFailedPane } from './ui/RestoreFailedPane.vue'
export { default as UnavailablePane } from './ui/UnavailablePane.vue'
export { getNoteInFocus } from './model/noteInFocus'
export { measureTabs } from './model/measureTabs'
export { createWindowLayout } from './model/createWindowLayout'
export { openPlacedTabs, startPlacedTabs } from './model/openPlacedTabs'
