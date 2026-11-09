/**
 * The tabs of the window, and what currently open in them.
 */
import type { OpenTab } from '@numen/plugin-loader'

/** One tab of the window, as the core keeps it. */
export interface Tab {
  readonly id: string
  readonly kind: string
  readonly path: string
  readonly title: string
  readonly state: string
  readonly place?: OpenTab['place']
}

/** What currently open: every tab, and which of them is active. */
export interface OpenTabs {
  readonly tabs: readonly Tab[]
  readonly activeId: string
}
