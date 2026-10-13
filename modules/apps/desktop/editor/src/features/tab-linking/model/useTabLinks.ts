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
  tabs: readonly [TabId, TabId],
  colorIndex: number,
): void {
  const groupId = crypto.randomUUID()
  groups.set(groupId, {
    id: groupId,
    colorIndex,
    tabs: [tabs[0], tabs[1]],
  })
}

function shouldSkipLinking(source: TabId, target: TabId): boolean {
  return !source || !target || source === target
}

interface ConnectOptions {
  sourceGroup: TabLinkGroup | null
  targetGroup: TabLinkGroup | null
  colorIndex: number
}

function connectGroups(
  groups: Map<string, TabLinkGroup>,
  tabs: readonly [TabId, TabId],
  options: ConnectOptions,
): void {
  const [sourceTabId, targetTabId] = tabs
  const { sourceGroup, targetGroup, colorIndex } = options
  if (sourceGroup && targetGroup) {
    mergeIntoGroup(groups, sourceGroup, targetGroup)
  } else if (sourceGroup) {
    addToGroup(groups, sourceGroup, targetTabId)
  } else if (targetGroup) {
    addToGroup(groups, targetGroup, sourceTabId)
  } else {
    createGroup(groups, [sourceTabId, targetTabId], colorIndex)
  }
}

function getNextColorIndex(groups: ReadonlyMap<string, TabLinkGroup>): number {
  const used = new Set<number>()
  for (const group of groups.values()) {
    used.add(group.colorIndex)
  }
  for (let index = 1; index <= MAX_COLOR_INDEX; index++) {
    if (!used.has(index)) return index
  }
  return 1
}

function findGroupOf(groups: ReadonlyMap<string, TabLinkGroup>, tabId: TabId): TabLinkGroup | null {
  for (const group of groups.values()) {
    if (group.tabs.includes(tabId)) return group
  }
  return null
}

function removeTabFromGroups(
  groups: ReadonlyMap<string, TabLinkGroup>,
  tabId: TabId,
): ReadonlyMap<string, TabLinkGroup> {
  const group = findGroupOf(groups, tabId)
  if (!group) return groups

  const remainingTabs = group.tabs.filter((id) => id !== tabId)
  const nextGroups = new Map(groups)

  if (remainingTabs.length < 2) {
    nextGroups.delete(group.id)
  } else {
    nextGroups.set(group.id, {
      ...group,
      tabs: remainingTabs,
    })
  }

  return nextGroups
}

export function useTabLinks(): TabLinksState {
  const groupsState: ShallowRef<ReadonlyMap<string, TabLinkGroup>> = shallowRef(new Map())

  const getGroupOf = (tabId: TabId): TabLinkGroup | null => findGroupOf(groupsState.value, tabId)

  const getLinkedTargets = (tabId: TabId): readonly TabId[] => {
    const group = getGroupOf(tabId)
    return group ? group.tabs.filter((id) => id !== tabId) : []
  }

  const unlinkTab = (tabId: TabId): void => {
    groupsState.value = removeTabFromGroups(groupsState.value, tabId)
  }

  const linkTabs = (sourceTabId: TabId, targetTabId: TabId): void => {
    if (shouldSkipLinking(sourceTabId, targetTabId)) return

    const sourceGroup = getGroupOf(sourceTabId)
    const targetGroup = getGroupOf(targetTabId)

    if (sourceGroup && targetGroup && sourceGroup.id === targetGroup.id) {
      return
    }

    const nextGroups = new Map(groupsState.value)
    connectGroups(nextGroups, [sourceTabId, targetTabId], {
      sourceGroup,
      targetGroup,
      colorIndex: getNextColorIndex(groupsState.value),
    })
    groupsState.value = nextGroups
  }

  return {
    get groups() {
      return groupsState.value
    },
    linkTabs,
    unlinkTab,
    getGroupOf,
    getLinkedTargets,
    removeClosedTab: (tabId: TabId) => unlinkTab(tabId),
  }
}
