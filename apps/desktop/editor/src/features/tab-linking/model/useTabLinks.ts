/**
 * Mirrors the link groups of all windows from the core. Changes are written to
 * the core and appear in the next snapshot.
 */
import { shallowRef } from 'vue'
import type {
  LinkableTab,
  TabId,
  TabLinkGroup,
  TabLinksSnapshot,
  TabLinksState,
  TabLinksWriter,
} from '../types'

export interface TabLinksDeps {
  readonly write: TabLinksWriter
  /** Receives changes the core rejected. */
  readonly onError: (error: unknown) => void
}

function findGroupOf(groups: ReadonlyMap<string, TabLinkGroup>, tabId: TabId): TabLinkGroup | null {
  for (const group of groups.values()) {
    if (group.tabs.includes(tabId)) return group
  }
  return null
}

export function useTabLinks(deps: TabLinksDeps): TabLinksState {
  const groupsState = shallowRef<ReadonlyMap<string, TabLinkGroup>>(new Map())
  const tabsState = shallowRef<readonly LinkableTab[]>([])

  const getGroupOf = (tabId: TabId): TabLinkGroup | null => findGroupOf(groupsState.value, tabId)

  const getLinkedTargets = (tabId: TabId): readonly TabId[] => {
    const group = getGroupOf(tabId)
    return group ? group.tabs.filter((id) => id !== tabId) : []
  }

  const linkTabs = (source: TabId, target: TabId): void => {
    if (!source || !target || source === target) return
    deps.write({ link: { source, target } }).catch(deps.onError)
  }

  const unlinkTab = (tabId: TabId): void => {
    if (!getGroupOf(tabId)) return
    deps.write({ unlink: tabId }).catch(deps.onError)
  }

  const applySnapshot = (snapshot: TabLinksSnapshot): void => {
    groupsState.value = new Map(snapshot.groups.map((group) => [group.id, group]))
    tabsState.value = snapshot.tabs
  }

  return {
    get groups() {
      return groupsState.value
    },
    get tabs() {
      return tabsState.value
    },
    linkTabs,
    unlinkTab,
    getGroupOf,
    getLinkedTargets,
    applySnapshot,
  }
}
