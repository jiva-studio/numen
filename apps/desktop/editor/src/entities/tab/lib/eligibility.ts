/**
 * Which tabs can be linked, and which of them a given tab can be linked to.
 */
import type { LinkableTab, TabId, TabLinkGroup } from './links'

/** A tab a link may end at, with the kind it shows and whether it is linked already. */
export interface EligibleLinkTab {
  readonly id: TabId
  readonly title: string
  readonly kind: string
  readonly isLinked: boolean
}

/** What the eligibility is read from. */
export interface EligibilityInput {
  readonly from: TabId
  readonly tabs: readonly { readonly id: TabId; readonly title: string }[]
  getKind(tab: TabId): string | undefined
  /** Whether the tab's selection can follow another's. */
  isLinkable(tab: TabId): boolean
  getGroupOf(tab: TabId): TabLinkGroup | null
}

/** The tabs a link from one tab may end at: linkable, not itself, not in its group already. */
export function getEligibleTabs(input: EligibilityInput): readonly EligibleLinkTab[] {
  if (!input.isLinkable(input.from)) return []
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

/** Returns the tabs of this window, then the linkable tabs of other windows. */
export function getAllTabs(
  own: readonly { readonly id: TabId; readonly title: string }[],
  linkable: readonly LinkableTab[],
): readonly { readonly id: TabId; readonly title: string }[] {
  const ownIds = new Set(own.map((tab) => tab.id))
  const others = linkable
    .filter((tab) => !ownIds.has(tab.id))
    .map((tab) => ({ id: tab.id, title: tab.title }))
  return [...own, ...others]
}
