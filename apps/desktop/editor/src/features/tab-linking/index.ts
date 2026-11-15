/**
 * Public exports for the tab linking feature.
 */
export * from './types'
export { createTabLinksApi } from './api/tab-links'
export { createTabLinking } from './model/createTabLinking'
export { createWindowTabLinkSession } from './model/createWindowTabLinkSession'
export { default as TabLinkIndicator } from './ui/TabLinkIndicator.vue'
