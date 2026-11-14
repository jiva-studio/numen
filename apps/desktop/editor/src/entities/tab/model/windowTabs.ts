/**
 * The window as a set of kinds, and the tabs it keeps of each.
 *
 * A kind defines how its tabs open, their title, their content and how they
 * close. This is implemented once for all kinds: a kind is added by declaring
 * it, with no further window configuration.
 */
import { shallowRef } from 'vue'
import { createWindowHandle } from './tabOperations'
import { createTabStore } from './tabStore'
import { useRestoredTabs } from './useRestoredTabs'
import { useTabKinds } from './useTabKinds'
import { useTabPanes } from './useTabPanes'
import { useTabs } from './useTabs'
import type { PlaceholderPanes, UnavailableServices } from './unavailable'

/**
 * The tabs of a window, made of the parts that share one store: `tabs` open and navigate them,
 * `kinds` register the kinds, `panes` lay them out and close them, `restored` open the saved ones,
 * and `handle` is what a kind may ask of the window. `panes` draw the placeholders and
 * `unavailable` is what the placeholder of a missing kind reads and does.
 */
export function createWindowTabs(panes: PlaceholderPanes, unavailable: UnavailableServices) {
  const store = createTabStore(panes, unavailable)
  return {
    tabs: useTabs(store),
    kinds: useTabKinds(store),
    panes: useTabPanes(store),
    restored: useRestoredTabs(store),
    handle: createWindowHandle(store),
    /** The ids of the tabs being dragged, which the plugins read. */
    dragged: shallowRef<readonly string[]>([]),
  }
}

export type WindowTabs = ReturnType<typeof createWindowTabs>
