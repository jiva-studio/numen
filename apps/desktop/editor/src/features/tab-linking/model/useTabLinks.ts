/**
 * Mirrors the link groups of all windows from the core. Changes are written to
 * the core one at a time, in the order they are made, and appear in the next
 * snapshot.
 */
import { shallowRef } from 'vue'
import type { TabLinksState, TabLinksWriter } from '../types'
import type {
  LinkableTab,
  TabId,
  TabLinkGroup,
  TabLinksChange,
  TabLinksSnapshot,
  TabWindow,
} from '@/entities/tab'
import type { Caught } from '@/shared/api/errors'

export interface TabLinksDeps {
  readonly write: TabLinksWriter
  readonly onError: (error: Caught) => void
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
  const windowsState = shallowRef<readonly TabWindow[]>([])
  let lastWrite: Promise<void> = Promise.resolve()

  /** Writes a change to the core after the pending writes. */
  const writeChange = (change: TabLinksChange): void => {
    lastWrite = lastWrite.then(() => deps.write(change)).catch(deps.onError)
  }

  const getGroupOf = (tabId: TabId): TabLinkGroup | null => findGroupOf(groupsState.value, tabId)

  const getLinkedTargets = (tabId: TabId): readonly TabId[] => {
    const group = getGroupOf(tabId)
    return group ? group.tabs.filter((id) => id !== tabId) : []
  }

  const linkTabs = (source: TabId, target: TabId): void => {
    if (!source || !target || source === target) return
    writeChange({ link: { source, target } })
  }

  const unlinkTab = (tabId: TabId): void => {
    if (!getGroupOf(tabId)) return
    writeChange({ unlink: tabId })
  }

  const applySnapshot = (snapshot: TabLinksSnapshot): void => {
    groupsState.value = new Map(snapshot.groups.map((group) => [group.id, group]))
    tabsState.value = snapshot.tabs
    windowsState.value = snapshot.windows
  }

  return {
    get groups() {
      return groupsState.value
    },
    get tabs() {
      return tabsState.value
    },
    get windows() {
      return windowsState.value
    },
    linkTabs,
    unlinkTab,
    getGroupOf,
    getLinkedTargets,
    applySnapshot,
  }
}
