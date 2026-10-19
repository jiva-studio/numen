/**
 * Which tabs can be linked, and which of them a given tab can be linked to.
 */
import type { EligibleLinkTab, TabId, TabLinkGroup } from '../types'

/** What the eligibility is read from. */
export interface EligibilityInput {
  /** The tab a link would leave. */
  readonly from: TabId
  readonly tabs: readonly { readonly id: TabId; readonly title: string }[]
  getKind(tab: TabId): string | undefined
  /** Whether the tab's selection can follow another's. */
  isLinkable(tab: TabId): boolean
  getGroupOf(tab: TabId): TabLinkGroup | null
}

/** The tabs a link from one tab may end at: linkable, not itself, not in its group already. */
export function getEligibleTabs(input: EligibilityInput): readonly EligibleLinkTab[] {
  const group = input.getGroupOf(input.from)
  return input.tabs
    .filter((tab) => {
      if (tab.id === input.from) return false
      if (group?.tabs.includes(tab.id)) return false
      return input.isLinkable(tab.id)
    })
    .map((tab) => ({
      id: tab.id,
      title: tab.title,
      kind: input.getKind(tab.id) ?? '',
      isLinked: input.getGroupOf(tab.id) !== null,
    }))
}

/** The titles of the tabs a tab is linked to, joined; a tab with no title is named by its identity. */
export function getLinkedTitle(
  targets: readonly TabId[],
  tabs: readonly { readonly id: TabId; readonly title: string }[],
): string {
  return targets.map((id) => tabs.find((tab) => tab.id === id)?.title || id).join(', ')
}
