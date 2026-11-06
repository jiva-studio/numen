/**
 * The tabs of the window, and what currently open in them.
 */
import type { OpenTab } from '@numen/plugin-host'

/** One tab of the window, as the core keeps it. */
export interface Tab {
  readonly id: string
  /** The registered name of the tab kind. */
  readonly kind: string
  /** The file it holds, empty for a tab holding none. */
  readonly path: string
  /** The display title of the tab. */
  readonly title: string
  /** The JSON the plugin owning the kind saved for the tab, empty where it saved none. */
  readonly state: string
  /** The place the reader of the tab reported, omitted when it reported none. */
  readonly place?: OpenTab['place']
}

/** What currently open: every tab, and which of them is active. */
export interface OpenTabs {
  readonly tabs: readonly Tab[]
  readonly activeId: string
}
