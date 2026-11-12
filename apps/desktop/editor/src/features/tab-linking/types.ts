/**
 * Types for tab linking and path sync between tabs of all windows.
 */
import type {
  LinkableTab,
  TabId,
  TabLinkGroup,
  TabLinksChange,
  TabLinksSnapshot,
  TabWindow,
} from '@/entities/tab'

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
