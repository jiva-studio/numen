import type { JsonValue } from '@/shared/api/json'
/**
 * The tabs of the window, and what currently open in them.
 */

/** One tab of the window, as the core keeps it. */
export interface Tab {
  readonly id: string
  readonly kind: string
  readonly path: string
  readonly title: string
  readonly state: string
  readonly place?: string
}

/** A tab kept by the core, to open again. */
export interface SavedTab {
  readonly id: string
  readonly kind: string
  readonly path: string
  readonly title?: string | undefined
  readonly saved?: JsonValue | undefined
}

/** What currently open: every tab, and which of them is active. */
export interface OpenTabs {
  readonly layout?: string
  readonly tabs: readonly Tab[]
  readonly activeId: string
}
