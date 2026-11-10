/** A tab kind: the state its tabs hold, and the operations a kind may request from the window. */
import type { Component } from 'vue'
import type { Result } from '@numen/core-client'
import type { JsonValue, Selector, TabPlace, TabPlacement, TextRange } from '@numen/plugin-sdk'

/**
 * Opens a file in its reader, at the location the selector names, with the ranges of its text
 * version highlighted; the reader scrolls to the first range.
 */
export type SourceReader = (
  path: string,
  selector: Selector | null,
  ranges?: readonly TextRange[] | null,
) => void

/**
 * The target of a command issued against one tab: the note the tab represents,
 * and the file a run is issued against. A kind with neither returns an empty target.
 */
export interface TabTarget {
  readonly path?: string
  readonly title?: string
  readonly file?: string
}

/**
 * The file one tab shows, and the place its reader last reported, as reported to an agent acting on
 * the user's behalf. The place is left out where the reader reported none.
 */
export interface OpenTab {
  readonly path: string
  readonly place?: TabPlace
}

/** A tab kind: its contents, its title, and how it releases its state. */
export interface TabKind<TabState, K extends string = string> {
  readonly kind: K
  readonly pluginId?: string
  /**
   * What one of its tabs holds, made as the tab opens on what it was given.
   * It is made at once, so whatever it watches is caught by the tab's scope
   * and release with the tab.
   */
  open(targetPath: string, id: string, saved?: JsonValue): TabState
  /**
   * The JSON value kept with the tab across restarts and passed back to open
   * as `saved`. A kind that declares none opens its tabs again on their path.
   */
  getState?(state: TabState): JsonValue | undefined
  /** The title of the tab, derived from its current state. */
  getTitle?(state: TabState): string
  /** The one word the tab carries beside its title, or nothing. */
  getMark?(state: TabState): string | undefined
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
   * The tab releases what it held. False keeps it on screen: its contents
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
  readonly isTool?: boolean
  isAsking?(state: TabState): boolean
  /** Writes what the tab holds out to its file; answers once nothing is on its way. */
  flushPending?(state: TabState): Promise<void>
  /** A file made in a folder under a name, as the file this kind edits. */
  createFile?(folder: string, name: string): Promise<Result<{ path: string }, string>>
  readonly icon?: Component
  readonly placement?: TabPlacement
}

/**
 * A kind as the window stores it. The state of its tabs is private to the kind,
 * and the window passes it back to the kind unchanged.
 */
export type AnyTabKind = TabKind<object>
