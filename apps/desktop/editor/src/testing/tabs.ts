/**
 * The tabs of a window as the window makes them, with the placeholder panes it draws.
 */
import { createWindowTabs, RestoreFailedPane, UnavailablePane } from '@/entities/tab'

/** The tabs of a window; `unavailable` is what the placeholder of a missing kind reads and does. */
export const createTestTabs = (unavailable?: Parameters<typeof createWindowTabs>[1]) =>
  createWindowTabs({ unavailable: UnavailablePane, restoreFailed: RestoreFailedPane }, unavailable)
