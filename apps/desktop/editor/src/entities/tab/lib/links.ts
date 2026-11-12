/**
 * The link groups and the linkable tabs of all windows, and the messages the core sends about them.
 */
import type { Tab } from './tab'

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

/** Why a move of a tab to another window did not complete. */
export type TabMoveError = 'window-closed' | 'timeout' | 'not-owner' | 'cancelled'

/** A tab sent to this window, and the id of the move that AcceptTabMove names. */
export interface MovedTab {
  readonly tab: Tab
  readonly moveId: string
}

/** Linkable tabs open in a window, and the window summary shown in the menus of other windows. */
export interface WindowTabsReport {
  readonly tabs: readonly { readonly id: string; readonly kind: string; readonly title: string }[]
  readonly window: Omit<TabWindow, 'id'>
}

/** New path of a linked tab. */
export interface TabPathSync {
  readonly groupId: string
  readonly sourceTabId: string
  readonly path: string
  readonly title: string
}

/** Tab links stream message. A keepalive sets none of the fields. */
export interface TabLinksEvent {
  readonly links?: TabLinksSnapshot
  readonly sync?: TabPathSync
  readonly movedTab?: MovedTab
  readonly isHighlighted?: boolean
}
