/**
 * Tab links of the demo window. The demo has no core: the link groups are held
 * in memory, and each write applies the change and hands the groups to the
 * tab links as a snapshot.
 */
import { useTabLinks } from '@editor-features/tab-linking'
import type { TabId, TabLinkGroup, TabLinksChange } from '@editor-features/tab-linking'

const MAX_COLOR_INDEX = 4

/** Returns the lowest color no group has, or 1 when every color is taken. */
function getNextColorIndex(groups: readonly TabLinkGroup[]): number {
  const used = new Set(groups.map((group) => group.colorIndex))
  for (let index = 1; index <= MAX_COLOR_INDEX; index++) {
    if (!used.has(index)) return index
  }
  return 1
}

/** Returns the groups with two tabs linked: a tab joins the group of the other, two groups merge. */
function linkTabs(
  groups: readonly TabLinkGroup[],
  source: TabId,
  target: TabId,
): readonly TabLinkGroup[] {
  const sourceGroup = groups.find((group) => group.tabs.includes(source))
  const targetGroup = groups.find((group) => group.tabs.includes(target))
  if (sourceGroup && sourceGroup === targetGroup) return groups
  if (sourceGroup) {
    const joined = targetGroup ? targetGroup.tabs : [target]
    return groups
      .filter((group) => group !== targetGroup)
      .map((group) =>
        group === sourceGroup
          ? { ...group, tabs: [...new Set([...group.tabs, ...joined])] }
          : group,
      )
  }
  if (targetGroup) {
    return groups.map((group) =>
      group === targetGroup ? { ...group, tabs: [...group.tabs, source] } : group,
    )
  }
  const created: TabLinkGroup = {
    id: crypto.randomUUID(),
    colorIndex: getNextColorIndex(groups),
    tabs: [source, target],
    path: '',
    title: '',
  }
  return [...groups, created]
}

/** Returns the groups without the tab; a group left with one tab is removed. */
function unlinkTab(groups: readonly TabLinkGroup[], tabId: TabId): readonly TabLinkGroup[] {
  return groups
    .map((group) => ({ ...group, tabs: group.tabs.filter((id) => id !== tabId) }))
    .filter((group) => group.tabs.length >= 2)
}

/** Returns tab links whose groups are held in memory. */
export function createDemoTabLinks() {
  let groups: readonly TabLinkGroup[] = []
  const tabLinks = useTabLinks({
    write: (change: TabLinksChange) => {
      if ('link' in change) groups = linkTabs(groups, change.link.source, change.link.target)
      else groups = unlinkTab(groups, change.unlink)
      tabLinks.applySnapshot({ groups, tabs: [] })
      return Promise.resolve()
    },
    onError: (error: unknown) => console.error(error),
  })
  return tabLinks
}
