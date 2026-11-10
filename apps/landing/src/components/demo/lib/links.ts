/** Tabs that follow one another: they move to the same vault path together. */
export interface TabLinkGroup {
  readonly id: string
  readonly colorIndex: number
  readonly tabs: readonly string[]
}

const COLOR_COUNT = 4

function getNextColorIndex(groups: readonly TabLinkGroup[]): number {
  const used = new Set(groups.map((group) => group.colorIndex))
  for (let index = 1; index <= COLOR_COUNT; index++) {
    if (!used.has(index)) return index
  }
  return 1
}

export function findGroupOf(groups: readonly TabLinkGroup[], tab: string): TabLinkGroup | null {
  return groups.find((group) => group.tabs.includes(tab)) ?? null
}

/** The groups after the source tab joins the group of the target tab, or both start one. */
export function linkTabs(
  groups: readonly TabLinkGroup[],
  source: string,
  target: string,
  createId: () => string,
): readonly TabLinkGroup[] {
  const sourceGroup = findGroupOf(groups, source)
  const targetGroup = findGroupOf(groups, target)
  if (sourceGroup && sourceGroup === targetGroup) return groups
  if (sourceGroup) {
    const joined = targetGroup ? targetGroup.tabs : [target]
    return groups
      .filter((group) => group !== targetGroup)
      .map((group) =>
        group === sourceGroup ? { ...group, tabs: [...new Set([...group.tabs, ...joined])] } : group,
      )
  }
  if (targetGroup) {
    return groups.map((group) =>
      group === targetGroup ? { ...group, tabs: [...group.tabs, source] } : group,
    )
  }
  return [
    ...groups,
    { id: createId(), colorIndex: getNextColorIndex(groups), tabs: [source, target] },
  ]
}

/** The groups after a tab leaves its group; a group left with one tab ends. */
export function unlinkTab(groups: readonly TabLinkGroup[], tab: string): readonly TabLinkGroup[] {
  return groups
    .map((group) => ({ ...group, tabs: group.tabs.filter((id) => id !== tab) }))
    .filter((group) => group.tabs.length >= 2)
}
