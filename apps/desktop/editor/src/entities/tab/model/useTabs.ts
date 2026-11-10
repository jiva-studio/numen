/**
 * The tabs that are open: what each is called, how one is looked up, opened, and navigated.
 */
import { computed } from 'vue'
import type { Tab } from '@numen/workbench'
import { createTabNavigation, createTabOpeners, createTabQueries } from './tabOperations'
import type { TabStore } from './tabStore'

export function useTabs(store: TabStore) {
  const tabs = computed<readonly Tab[]>(() =>
    [...store.open.value].map(([id, one]): Tab => {
      const mark = one.kind.display?.getMark?.(one.state)
      const title = one.kind.getTitle?.(one.state) ?? ''
      return { id, title, ...(mark ? { mark } : {}) }
    }),
  )
  const queries = createTabQueries(store)
  const openers = createTabOpeners(store)
  const navigation = createTabNavigation(store, openers)

  return {
    tabs,
    getTab: queries.getTab,
    getTabStateIn: queries.getTabStateIn,
    createTab: openers.createTab,
    openTabOfKind: openers.openTabOfKind,
    openTabAdjacentOfKind: openers.openTabAdjacentOfKind,
    navigateTab: navigation.navigateTab,
    navigateTabsAt: navigation.navigateTabsAt,
  }
}
