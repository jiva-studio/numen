/** A kind of tab: what one of its tabs holds, and what a kind may ask of the window. */
import type { Component } from 'vue'
import type { Source } from '@/shared/file'
import type { ProgressOf } from './tab'

/**
 * What a command asked over one tab is over: the note the tab means, and the
 * file a run is asked over. A kind that means neither answers with neither.
 */
export interface TabTarget {
  readonly path?: string
  readonly title?: string
  readonly file?: string
  readonly source?: Source | null
}

/**
 * What one tab holds, as whoever answers on the person's behalf is told it: the
 * file it stands at, and the document, the book or the recording it stands in.
 */
export type OpenTab<K extends string = string> = { readonly path: string } & ProgressOf<K>

/** A kind of tab: what it holds, what it is called, and what it lets go of. */
export interface TabKind<TabState, K extends string = string> {
  /** The word the identities of its tabs are filed under. */
  readonly kind: K
  /**
   * What one of its tabs holds, made as the tab opens on what it was given.
   * It is made at once, so whatever it watches is caught by the tab's scope
   * and let go of with the tab.
   */
  open(at: string): TabState
  /** What the tab is called, as what it holds now stands. */
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
  identity?(at: string): string
  /** The tab came on screen, where what it holds has room to measure. */
  onShow?(state: TabState, id: string): void
  /**
   * A key struck while one of its tabs is the one the person is in, answered
   * with whether the tab took it. Several panes are drawn at once, so the tab
   * offered it is the one showing in the pane the layout is focused on, and
   * then whatever else is on screen.
   */
  onKeyPress?(state: TabState, event: KeyboardEvent): boolean
  /** What a command asked over one of its tabs is over. */
  getTarget?(state: TabState): TabTarget
  /** What one of its tabs holds, as whoever answers for the person is told it. */
  getOpenTab?(state: TabState): OpenTab<K>
  /**
   * The tab lets go of what it held. False keeps it on screen: what it holds
   * has something to finish, and closes the tab itself once it has.
   */
  onClose?(state: TabState, id: string): boolean
  /**
   * The window is going, and nothing this tab holds outlives it. A kind that
   * says nothing here lets go the way a tab of it closes.
   */
  onDestroy?(state: TabState, id: string): void
}

/**
 * A kind as the window keeps it. What its tabs hold is the kind's own affair,
 * and the window hands it back to the kind untouched.
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

/** The tab the person is looking at, whichever kind it turns out to be. */
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
  /** The same, drawn beside the pane the person is in. */
  openTabBeside(kind: string, at?: string): Promise<string>
  /** A tab the window already holds, put in front. */
  show(id: string): void
  /** A tab that took its own close, going now. */
  closeTab(id: string): void
  /**
   * Every tab of a kind, in the order the person was last in them. The last of
   * them is the one in front.
   */
  each<TabState>(kind: string): readonly KindTab<TabState>[]
  /** The tab of a kind the person was last in, and nothing where it holds none. */
  last<TabState>(kind: string): KindTab<TabState> | null
  /**
   * The tab showing in the pane the person is in, of whatever kind. A pane
   * holding nothing answers with nothing.
   */
  front(): ActiveTab | null
  /** What one tab of a kind holds, and nothing where the tab is another kind. */
  getTabState<TabState>(kind: string, id: string): TabState | null
}
