/**
 * Types for tab linking and path sync between tabs of all windows.
 */

export type TabId = string

export interface TabLinkGroup {
  readonly id: string
  readonly colorIndex: number
  readonly tabs: readonly TabId[]
  readonly path: string
  readonly title: string
}

/** Open tab of any window whose kind supports navigate. */
export interface LinkableTab {
  readonly id: TabId
  readonly kind: string
  readonly title: string
  readonly windowId: string
}

/** A window as it last reported itself to the core. */
export interface TabWindow {
  readonly id: string
  readonly isMain: boolean
  readonly activeTitle: string
  readonly activeKind: string
  readonly tabCount: number
}

/** Link groups of the vault, linkable tabs of all windows, and the windows, as sent by the core. */
export interface TabLinksSnapshot {
  readonly groups: readonly TabLinkGroup[]
  readonly tabs: readonly LinkableTab[]
  readonly windows: readonly TabWindow[]
}

/** Link or unlink change written to the core. */
export type TabLinksChange =
  { readonly link: { readonly source: TabId; readonly target: TabId } } | { readonly unlink: TabId }

/** Writes a link change to the core. */
export type TabLinksWriter = (change: TabLinksChange) => Promise<void>

export interface TabLinksState {
  readonly groups: ReadonlyMap<string, TabLinkGroup>
  readonly tabs: readonly LinkableTab[]
  readonly windows: readonly TabWindow[]
  linkTabs(sourceTabId: TabId, targetTabId: TabId): void
  unlinkTab(tabId: TabId): void
  getGroupOf(this: void, tabId: TabId): TabLinkGroup | null
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
