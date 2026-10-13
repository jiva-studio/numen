/**
 * Which tabs can be linked, and which of them a given tab can be linked to.
 */
import type { EligibleLinkTab, TabId, TabLinkGroup } from '../types'

/** The kinds of tab whose selection can follow another. */
export const LINKABLE_KINDS = ['note', 'files', 'plex'] as const

/** Whether a tab of this kind can be linked. */
export function isLinkableKind(kind: string | undefined): boolean {
  return LINKABLE_KINDS.some((one) => one === kind)
}

/** What the eligibility is read from. */
export interface EligibilityInput {
  /** The tab a link would leave. */
  readonly from: TabId
  readonly tabs: readonly { readonly id: TabId; readonly title: string }[]
  getKind(tab: TabId): string | undefined
  getGroupOf(tab: TabId): TabLinkGroup | null
}

/** The tabs a link from one tab may end at: linkable, not itself, not in its group already. */
export function getEligibleTabs(input: EligibilityInput): readonly EligibleLinkTab[] {
  const group = input.getGroupOf(input.from)
  return input.tabs
    .filter((tab) => {
      if (tab.id === input.from) return false
      if (group?.tabs.includes(tab.id)) return false
      return isLinkableKind(input.getKind(tab.id))
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
