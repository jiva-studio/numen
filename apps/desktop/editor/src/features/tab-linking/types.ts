/**
 * Types for tab linking and path sync between tabs of all windows.
 */

export type TabId = string

export interface TabLinkGroup {
  readonly id: string
  readonly colorIndex: number
  readonly tabs: readonly TabId[]
  /** Last path synced in the group, and its title; empty before the first sync. */
  readonly path: string
  readonly title: string
}

/** Open tab of any window whose kind supports navigate. */
export interface LinkableTab {
  readonly id: TabId
  readonly kind: string
  readonly title: string
  /** Id of the window that has the tab open. */
  readonly windowId: string
}

/** A window as it last reported itself to the core. */
export interface TabWindow {
  readonly id: string
  readonly isMain: boolean
  /** The active tab; both fields are empty for a window with no tabs. */
  readonly activeTitle: string
  readonly activeKind: string
  /** Every tab the window has open, linkable or not. */
  readonly tabCount: number
}

/** Link groups of the vault, linkable tabs of all windows, and the windows, as sent by the core. */
export interface TabLinksSnapshot {
  readonly groups: readonly TabLinkGroup[]
  readonly tabs: readonly LinkableTab[]
  /** The main window first. */
  readonly windows: readonly TabWindow[]
}

/** Link or unlink change written to the core. */
export type TabLinksChange =
  { readonly link: { readonly source: TabId; readonly target: TabId } } | { readonly unlink: TabId }

/** Writes a link change to the core. */
export type TabLinksWriter = (change: TabLinksChange) => Promise<void>

export interface TabLinksState {
  readonly groups: ReadonlyMap<string, TabLinkGroup>
  /** Linkable tabs of all windows. */
  readonly tabs: readonly LinkableTab[]
  /** Every window that reported its tabs, the main window first. */
  readonly windows: readonly TabWindow[]
  linkTabs(sourceTabId: TabId, targetTabId: TabId): void
  unlinkTab(tabId: TabId): void
  getGroupOf(tabId: TabId): TabLinkGroup | null
  getLinkedTargets(tabId: TabId): readonly TabId[]
  /** Replaces the groups and tabs with a snapshot from the core. */
  applySnapshot(snapshot: TabLinksSnapshot): void
}

export interface EligibleLinkTab {
  readonly id: TabId
  readonly title: string
  readonly kind: string
  readonly isLinked: boolean
}
