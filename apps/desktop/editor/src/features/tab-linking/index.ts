/**
 * Public exports for the tab linking feature.
 */
export * from './types'
export { useTabLinks } from './model/useTabLinks'
export { getWindowName } from './lib/windowName'
export { getAllTabs, getEligibleTabs } from './lib/eligibility'
export { default as TabLinkIndicator } from './ui/TabLinkIndicator.vue'
