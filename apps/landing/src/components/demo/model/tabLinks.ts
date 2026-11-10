import { shallowRef } from 'vue'
import { findGroupOf, linkTabs, unlinkTab, type TabLinkGroup } from '../lib/links'

/** The link groups of the demo window, held in memory. */
export function createTabLinks() {
  const groups = shallowRef<readonly TabLinkGroup[]>([])
  let count = 0
  const createId = (): string => `link-group-${(count += 1)}`

  const getGroupOf = (tab: string): TabLinkGroup | null => findGroupOf(groups.value, tab)

  return {
    getGroupOf,
    getLinkedTargets: (tab: string): readonly string[] =>
      getGroupOf(tab)?.tabs.filter((id) => id !== tab) ?? [],
    link(source: string, target: string) {
      if (source && target && source !== target) {
        groups.value = linkTabs(groups.value, source, target, createId)
      }
    },
    unlink(tab: string) {
      groups.value = unlinkTab(groups.value, tab)
    },
  }
}

export type TabLinks = ReturnType<typeof createTabLinks>
