/**
 * The panes the open tabs are laid out in: which tab shows, which keystroke a tab takes, and how
 * a tab closes.
 */
import { computed } from 'vue'
import { getPanes, paneById } from '@numen/workbench'
import { createTabClosers } from './tabOperations'
import type { TabStore } from './tabStore'

export function useTabPanes(store: TabStore) {
  const { layout, open } = store
  const closers = createTabClosers(store)
  /** The tab in front in the focused pane. */
  const frontId = computed(() => paneById(layout.value.root, layout.value.focus)?.active)

  /** Offers the keystroke to the tab in front first, then to the other tabs that show, last shown first. */
  const onKeyPress = (event: KeyboardEvent): boolean => {
    const activeTabIds = getPanes(layout.value.root).map((one) => one.active)
    const proposed = new Set<string>()
    for (const id of [frontId.value, ...[...open.value.keys()].reverse()]) {
      if (!id || proposed.has(id) || !activeTabIds.includes(id)) continue
      proposed.add(id)
      const one = open.value.get(id)
      if (one?.kind.events?.onKeyPress?.(one.state, event)) return true
    }
    return false
  }

  return {
    layout,
    show: closers.show,
    onTabShown: closers.onTabShown,
    onKeyPress,
    finishClose: closers.finishClose,
    requestClose: closers.requestClose,
    releaseTab: closers.releaseTab,
    reconcileTab: closers.reconcileTab,
  }
}
