/** What the window holds of its tabs. The kinds themselves belong to the plugin host. */
import type { AnyTabKind } from '@numen/plugin-loader'

export type { AnyTabKind, SourceReader, TabKind } from '@numen/plugin-loader'

/** What one tab is: its kind, and what that kind gave it to hold. */
export interface WindowTab {
  readonly kind: AnyTabKind
  readonly state: object
}

/** One tab of a kind, as that kind is given it back. */
export interface KindTab<TabState> {
  readonly id: string
  readonly state: TabState
}

/** The focused tab, of whichever kind. */
export interface ActiveTab {
  readonly id: string
  readonly kind: string | null
  readonly state: object | null
}

/** What a kind may ask of the window its tabs are drawn in. */
export interface WindowHandle {
  /** A tab of a kind, opened on something and put in front. */
  openTab(kind: string, at?: string): Promise<string>
  /** The same, opened in a pane adjacent to the focused pane. */
  openTabAdjacent(kind: string, at?: string): Promise<string>
  /** Activates an open tab. An id with no open tab changes nothing. */
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
