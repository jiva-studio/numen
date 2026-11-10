/**
 * The window as a set of kinds, and the tabs it keeps of each.
 *
 * A kind defines how its tabs open, their title, their content and how they
 * close. This is implemented once for all kinds: a kind is added by declaring
 * it, with no further window configuration.
 */
import { createTabStore } from './tabStore'
import { useRestoredTabs } from './useRestoredTabs'
import { useTabGroups } from './useTabGroups'
import { useTabPanes } from './useTabPanes'
import { useTabs } from './useTabs'
import type { PlaceholderPanes, UnavailableServices } from './unavailable'

/**
 * The tabs of a window, made of the parts that share one store. `panes` draw the placeholders and
 * `unavailable` is what the placeholder of a missing kind reads and does.
 */
export function createWindowTabs(panes: PlaceholderPanes, unavailable: UnavailableServices = {}) {
  const store = createTabStore(panes, unavailable)
  return {
    ...useTabs(store),
    ...useTabGroups(store),
    ...useTabPanes(store),
    ...useRestoredTabs(store),
  }
}

/** The tabs of a window. */
export type WindowTabs = ReturnType<typeof createWindowTabs>
