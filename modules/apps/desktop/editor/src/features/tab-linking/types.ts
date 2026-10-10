/**
 * Types for tab linking and selection synchronization between panes.
 */

export type TabId = string

export interface TabLinkGroup {
  readonly id: string
  readonly colorIndex: number
  readonly tabs: readonly TabId[]
}

export interface TabLinksState {
  readonly groups: ReadonlyMap<string, TabLinkGroup>
  linkTabs(sourceTabId: TabId, targetTabId: TabId): void
  unlinkTab(tabId: TabId): void
  getGroupOf(tabId: TabId): TabLinkGroup | null
  getLinkedTargets(tabId: TabId): readonly TabId[]
  removeClosedTab(tabId: TabId): void
}

export interface EligibleLinkTab {
  readonly id: TabId
  readonly title: string
  readonly kind: string
  readonly isLinked: boolean
}
