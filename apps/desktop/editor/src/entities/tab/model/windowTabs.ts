/**
 * The window as a set of kinds, and the tabs it keeps of each.
 *
 * A kind defines how its tabs open, their title, their content and how they
 * close. This is implemented once for all kinds: a kind is added by declaring
 * it, with no further window configuration.
 */
import { shallowRef } from 'vue'
import { createTabStore } from './tabStore'
import { useRestoredTabs } from './useRestoredTabs'
import { useTabGroups } from './useTabGroups'
import { useTabPanes } from './useTabPanes'
import { useTabs } from './useTabs'
import type { PlaceholderPanes, UnavailableServices } from './unavailable'

/**
 * The tabs of a window, made of the parts that share one store: `tabs` open and navigate them,
 * `groups` register the kinds, `panes` lay them out and close them, `restored` open the saved ones.
 * `panes` draw the placeholders and `unavailable` is what the placeholder of a missing kind reads
 * and does.
 */
export function createWindowTabs(panes: PlaceholderPanes, unavailable: UnavailableServices = {}) {
  const store = createTabStore(panes, unavailable)
  const tabPanes = useTabPanes(store)
  /** Closes a tab, or runs `hold` where its kind holds it, and returns whether the tab was released. */
  const closeTab = (id: string, hold?: () => void): boolean => {
    if (!tabPanes.releaseTab(id)) {
      hold?.()
      return false
    }
    if (!hold) tabPanes.finishClose(id)
    return true
  }
  return {
    tabs: useTabs(store),
    groups: useTabGroups(store),
    panes: tabPanes,
    closeTab,
    restored: useRestoredTabs(store),
    /** The ids of the tabs being dragged, which the plugins read. */
    dragged: shallowRef<readonly string[]>([]),
  }
}

export type WindowTabs = ReturnType<typeof createWindowTabs>
