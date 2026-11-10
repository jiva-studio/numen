/**
 * Public exports for the tab linking feature.
 */
export * from './types'
export { highlightWindow, syncTabPath, writeTabLinks } from './api/tab-links'
export { createTabLinkSession, getWindowReport } from './model/tabLinkSession'
export { useTabLinks } from './model/useTabLinks'
export { default as TabLinkIndicator } from './ui/TabLinkIndicator.vue'
