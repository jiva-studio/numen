import type { TabWindow } from './links'

/** The name of a window in a list of windows: its active tab and the count of its other tabs. */
export const getWindowName = (window: TabWindow): string => {
  const title = window.activeTitle || window.activeKind || 'Empty window'
  return window.tabCount > 1 ? `${title} (+${window.tabCount - 1})` : title
}
