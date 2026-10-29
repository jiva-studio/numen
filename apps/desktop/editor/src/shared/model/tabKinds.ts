/** A tab kind: the state its tabs hold, and the operations a kind may request from the window. */
import type { Component } from 'vue'
import type { Result, Selector, TabPlacement } from '@numen/plugin-sdk'

/** Opens a file in its reader, at the location the selector names. */
export type SourceReader = (path: string, selector: Selector | null) => void

/**
 * The target of a command issued against one tab: the note the tab represents,
 * and the file a run is issued against. A kind with neither returns an empty target.
 */
export interface TabTarget {
  readonly path?: string
  readonly title?: string
  readonly file?: string
}

/** The file one tab holds, as reported to an agent acting on the user's behalf. */
export interface OpenTab {
  readonly path: string
}

/** A tab kind: what it holds, its title, and how it releases its state. */
export interface TabKind<TabState, K extends string = string> {
  /** The word the identities of its tabs are filed under. */
  readonly kind: K
  /** The plugin that registered the kind, and nothing for a kind the window declares. */
  readonly pluginId?: string
  /**
   * What one of its tabs holds, made as the tab opens on what it was given.
   * It is made at once, so whatever it watches is caught by the tab's scope
   * and release with the tab.
   */
  open(targetPath: string, id: string, saved?: unknown): TabState
  /**
   * The JSON value kept with the tab across restarts and passed back to open
   * as `saved`. A kind that declares none opens its tabs again on their path.
   */
  getState?(state: TabState): unknown
  /** The title of the tab, derived from its current state. */
  getTitle?(state: TabState): string
  /** The one word the tab carries beside its title, or nothing. */
  getMark?(state: TabState): string | undefined
  /** What is drawn in the pane, given what the tab holds. */
  readonly pane: Component
  /**
   * The identity a tab of this kind takes from what it opens on. A kind that
   * declares one has a tab per thing, so the same thing opened twice is the
   * tab it already has.
   */
  identity?(targetPath: string): string
  /** The tab became visible, so its content can be measured. */
  onShow?(state: TabState, id: string): void
  /** The tab is handed text meant for it, such as a question the window puts to it. */
  receiveText?(state: TabState, text: string): void
  /**
   * A key press while one of its tabs is focused. Returns whether the tab
   * handled it. Several panes are visible at once, so the tab offered the key
   * is the one in the focused pane, followed by the other visible tabs.
   */
  onKeyPress?(state: TabState, event: KeyboardEvent): boolean
  /** The target of a command issued against one of its tabs. */
  getTarget?(state: TabState): TabTarget
  getOpenTab?(state: TabState): OpenTab
  /**
   * The tab lets go of what it held. False keeps it on screen: what it holds
   * has something to finish, and closes the tab itself once it has.
   */
  onClose?(state: TabState, id: string): boolean
  /**
   * The window is closing, and nothing this tab holds outlives it. A kind that
   * omits this hook releases its state the same way as a tab close.
   */
  onDestroy?(state: TabState, id: string): void
  /**
   * The tab moves to a path a tab linked to it moved to. A kind that declares
   * it can be linked.
   */
  navigate?(state: TabState, path: string, title: string): void | Promise<void>
  /** Its tabs stand beside files: a file opened on a path never closes one of them. */
  readonly isTool?: boolean
  isAsking?(state: TabState): boolean
  /** Writes what the tab holds out to its file; answers once nothing is on its way. */
  flushPending?(state: TabState): Promise<void>
  /** A file made in a folder under a name, as the file this kind edits. */
  createFile?(folder: string, name: string): Promise<Result<{ path: string }, string>>
  /** The icon its tabs carry before their title. */
  readonly icon?: Component
  /** Where a tab of it opens in a window that starts with nothing to restore. */
  readonly placement?: TabPlacement
}

/**
 * A kind as the window stores it. The state of its tabs is private to the kind,
 * and the window passes it back to the kind unchanged.
 */
export type AnyTabKind = TabKind<unknown>

/** What one tab is: its kind, and what that kind gave it to hold. */
export interface WindowTab {
  readonly kind: AnyTabKind
  readonly state: unknown
}

/** One tab of a kind, as that kind is given it back. */
export interface KindTab<TabState> {
  readonly id: string
  readonly state: TabState
}

/** The focused tab, of whichever kind. */
export interface ActiveTab {
  readonly id: string
  /** The word its kind is filed under, and nothing where the window holds no such tab. */
  readonly kind: string | null
  /** What its kind gave it to hold, and nothing where the window holds no such tab. */
  readonly state: unknown
}

/** What a kind may ask of the window its tabs are drawn in. */
export interface WindowHandle {
  /** A tab of a kind, opened on something and put in front. */
  openTab(kind: string, at?: string): Promise<string>
  /** The same, opened in a pane adjacent to the focused pane. */
  openTabAdjacent(kind: string, at?: string): Promise<string>
  /** A tab the window already holds, put in front. */
  show(id: string): void
  /** A tab that took its own close, going now. */
  closeTab(id: string): void
  /**
   * Every tab of a kind, ordered by last focus. The last entry is the most
   * recently focused tab.
   */
  each<TabState>(kind: string): readonly KindTab<TabState>[]
  /** The most recently focused tab of a kind, or null when there is none. */
  last<TabState>(kind: string): KindTab<TabState> | null
  /**
   * The tab visible in the focused pane, of any kind. Null when the pane is
   * empty.
   */
  front(): ActiveTab | null
  /** What one tab of a kind holds, and nothing where the tab is another kind. */
  getTabState<TabState>(kind: string, id: string): TabState | null
}
