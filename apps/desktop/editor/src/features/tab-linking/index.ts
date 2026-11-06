/**
 * Public exports for the tab linking feature.
 */
export * from './types'
export { useTabLinks } from './model/useTabLinks'
export { useTabLinkMenu, type TabLinkMenuDeps } from './model/useTabLinkMenu'
export { createLinkMenuItems, DETACH_ID, getMoveTarget, MOVE_ID, UNLINK_ID } from './lib/menu'
export { default as TabLinkIndicator } from './ui/TabLinkIndicator.vue'
