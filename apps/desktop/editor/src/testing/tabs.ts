/**
 * The tabs of a window as the window makes them, with the placeholder panes it draws.
 */
import { createWindowTabs, RestoreFailedPane, UnavailablePane } from '@/entities/tab'

type UnavailableServices = Parameters<typeof createWindowTabs>[1]

/** A catalog that names no plugin for any kind of tab. */
export const NO_TAB_KIND_PLUGINS: UnavailableServices = { findTabKindPlugin: () => Promise.resolve(null) }

/**
 * The tabs of a window; `unavailable` is what the placeholder of a missing kind reads and does, a
 * catalog that names no plugin where none is given.
 */
export const createTestTabs = (unavailable: UnavailableServices = NO_TAB_KIND_PLUGINS) =>
  createWindowTabs({ unavailable: UnavailablePane, restoreFailed: RestoreFailedPane }, unavailable)
