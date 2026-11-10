/** The tabs a window keeps: what kinds there are, what opens one, and what each is called. */
export { findTabKindIcon } from './lib/icons'
export { createFileCreators } from './model/makers'
export { fileOpeners } from './model/openers'
export type { FileOpener, FileOpeners, OpenPlacement } from './model/openers'
export type { OpenTabs, Tab } from './lib/tab'
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
export { useWindowTabMenu } from './model/useTabMenu'
export { createWorkspace, generateId, NOTE, SETTINGS } from './lib/workspace'
export { readOpenTabs } from './api/saved-tabs'
export { default as RestoreFailedPane } from './ui/RestoreFailedPane.vue'
export { default as UnavailablePane } from './ui/UnavailablePane.vue'
