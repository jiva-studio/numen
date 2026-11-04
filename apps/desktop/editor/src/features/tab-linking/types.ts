/**
 * Types for tab linking and path synchronization between tabs of every window.
 */

export type TabId = string

export interface TabLinkGroup {
  readonly id: string
  readonly colorIndex: number
  readonly tabs: readonly TabId[]
}

/** An open tab of any window whose kind can move to a path. */
export interface LinkableTab {
  readonly id: TabId
  readonly kind: string
  readonly title: string
  /** The window that has the tab open. */
  readonly windowId: string
}

/** Every link group of the vault and every linkable tab of every window, as the core sends them. */
export interface TabLinksSnapshot {
  readonly groups: readonly TabLinkGroup[]
  readonly tabs: readonly LinkableTab[]
}

/** One change a window writes to the link groups the core keeps. */
export type TabLinksChange =
  { readonly link: { readonly source: TabId; readonly target: TabId } } | { readonly unlink: TabId }

/** Where the window writes a change to the link groups. */
export type TabLinksWriter = (change: TabLinksChange) => Promise<void>

export interface TabLinksState {
  readonly groups: ReadonlyMap<string, TabLinkGroup>
  /** The linkable tabs of every window. */
  readonly tabs: readonly LinkableTab[]
  linkTabs(sourceTabId: TabId, targetTabId: TabId): void
  unlinkTab(tabId: TabId): void
  getGroupOf(tabId: TabId): TabLinkGroup | null
  getLinkedTargets(tabId: TabId): readonly TabId[]
  /** Replaces the groups and tabs with the ones the core sent. */
  applySnapshot(snapshot: TabLinksSnapshot): void
}

export interface EligibleLinkTab {
  readonly id: TabId
  readonly title: string
  readonly kind: string
  readonly isLinked: boolean
}
