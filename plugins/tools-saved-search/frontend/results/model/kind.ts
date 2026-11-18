/**
 * The tab kind of saved search notes: one tab per note path, titled by the note, following its
 * results from the moment it opens and aborting the watch when it closes.
 */
import type { Component } from 'vue'
import { createNamedIcon } from '@numen/ui'
import { getFileStem, type TabKindRegistration } from '@numen/plugin-sdk'
import {
  createSavedSearchTab,
  type SavedSearchTab,
  type SavedSearchTabDeps,
  type SavedSearchTabState,
} from './tab'
import { SAVED_SEARCH } from './tabKind'

/** What the window gives the kind, and the component that draws a tab. */
interface SavedSearchKindDeps extends SavedSearchTabDeps {
  readonly component: Component
}

/** One tab per note path; a tab follows its note from open until onClose or onDestroy. */
export function createSavedSearchKind(
  deps: SavedSearchKindDeps,
): TabKindRegistration<SavedSearchTabState> {
  const tabs = new WeakMap<SavedSearchTabState, SavedSearchTab>()
  const getTitle = (state: SavedSearchTabState): string =>
    state.view.value?.title || getFileStem(state.path.value)
  const dispose = (state: SavedSearchTabState): void => {
    tabs.get(state)?.dispose()
    tabs.delete(state)
  }

  const base: TabKindRegistration<SavedSearchTabState> = {
    kind: SAVED_SEARCH,
    component: deps.component,
    icon: createNamedIcon('search'),
    identity: (path) => path,
    open: (path) => {
      const tab = createSavedSearchTab(path, deps)
      tabs.set(tab.state, tab)
      tab.follow()
      return tab.state
    },
    getTitle,
    getOpenTab: (state) => ({ path: state.path.value }),
  }
  const lifecycle: Pick<
    TabKindRegistration<SavedSearchTabState>,
    'onShow' | 'onClose' | 'onDestroy'
  > = {
    onShow: (state) => tabs.get(state)?.follow(),
    onClose: (state) => {
      dispose(state)
      return true
    },
    onDestroy: (state) => dispose(state),
  }
  return Object.assign(base, lifecycle)
}
