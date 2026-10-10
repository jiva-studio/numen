/**
 * Tab linking state and synchronization between split panes.
 */
import { shallowRef, type ShallowRef } from 'vue'
import type { TabId, TabLinkGroup, TabLinksState } from '../types'

const MAX_COLOR_INDEX = 4

function mergeIntoGroup(
  groups: Map<string, TabLinkGroup>,
  targetGroup: TabLinkGroup,
  otherGroup: TabLinkGroup,
): void {
  groups.delete(otherGroup.id)
  const mergedTabs = Array.from(new Set([...targetGroup.tabs, ...otherGroup.tabs]))
  groups.set(targetGroup.id, {
    ...targetGroup,
    tabs: mergedTabs,
  })
}

function addToGroup(groups: Map<string, TabLinkGroup>, group: TabLinkGroup, tabId: TabId): void {
  groups.set(group.id, {
    ...group,
    tabs: [...group.tabs, tabId],
  })
}

function createGroup(
  groups: Map<string, TabLinkGroup>,
  sourceTabId: TabId,
  targetTabId: TabId,
  colorIndex: number,
): void {
  const groupId = crypto.randomUUID()
  groups.set(groupId, {
    id: groupId,
    colorIndex,
    tabs: [sourceTabId, targetTabId],
  })
}

function shouldSkipLinking(source: TabId, target: TabId): boolean {
  return !source || !target || source === target
}

function connectGroups(
  groups: Map<string, TabLinkGroup>,
  sourceGroup: TabLinkGroup | null,
  targetGroup: TabLinkGroup | null,
  sourceTabId: TabId,
  targetTabId: TabId,
  colorIndex: number,
): void {
  if (sourceGroup && targetGroup) {
    mergeIntoGroup(groups, sourceGroup, targetGroup)
  } else if (sourceGroup) {
    addToGroup(groups, sourceGroup, targetTabId)
  } else if (targetGroup) {
    addToGroup(groups, targetGroup, sourceTabId)
  } else {
    createGroup(groups, sourceTabId, targetTabId, colorIndex)
  }
}

export function useTabLinks(): TabLinksState {
  const groupsState: ShallowRef<ReadonlyMap<string, TabLinkGroup>> = shallowRef(new Map())

  const getNextColorIndex = (): number => {
    const used = new Set<number>()
    for (const group of groupsState.value.values()) {
      used.add(group.colorIndex)
    }
    for (let index = 1; index <= MAX_COLOR_INDEX; index++) {
      if (!used.has(index)) return index
    }
    return 1
  }

  const getGroupOf = (tabId: TabId): TabLinkGroup | null => {
    for (const group of groupsState.value.values()) {
      if (group.tabs.includes(tabId)) return group
    }
    return null
  }

  const getLinkedTargets = (tabId: TabId): readonly TabId[] => {
    const group = getGroupOf(tabId)
    if (!group) return []
    return group.tabs.filter((id) => id !== tabId)
  }

  const unlinkTab = (tabId: TabId): void => {
    const group = getGroupOf(tabId)
    if (!group) return

    const remainingTabs = group.tabs.filter((id) => id !== tabId)
    const nextGroups = new Map(groupsState.value)

    if (remainingTabs.length < 2) {
      nextGroups.delete(group.id)
    } else {
      nextGroups.set(group.id, {
        ...group,
        tabs: remainingTabs,
      })
    }

    groupsState.value = nextGroups
  }

  const linkTabs = (sourceTabId: TabId, targetTabId: TabId): void => {
    if (shouldSkipLinking(sourceTabId, targetTabId)) return

    const sourceGroup = getGroupOf(sourceTabId)
    const targetGroup = getGroupOf(targetTabId)

    if (sourceGroup && targetGroup && sourceGroup.id === targetGroup.id) {
      return
    }

    const nextGroups = new Map(groupsState.value)
    connectGroups(
      nextGroups,
      sourceGroup,
      targetGroup,
      sourceTabId,
      targetTabId,
      getNextColorIndex(),
    )
    groupsState.value = nextGroups
  }

  const removeClosedTab = (tabId: TabId): void => {
    unlinkTab(tabId)
  }

  return {
    get groups() {
      return groupsState.value
    },
    linkTabs,
    unlinkTab,
    getGroupOf,
    getLinkedTargets,
    removeClosedTab,
  }
}
