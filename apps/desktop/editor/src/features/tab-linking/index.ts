/**
 * Public exports for the tab linking feature.
 */
export * from './types'
export { syncTabPath } from './api/tab-links'
export { createTabLinking } from './model/createTabLinking'
export { createWindowTabLinkSession } from './model/createWindowTabLinkSession'
export type { TabReports } from './model/tabReports'
export { default as TabLinkIndicator } from './ui/TabLinkIndicator.vue'
