/** The icons of the tab kinds the window declares itself; a plugin kind carries the icon it registers. */
import { FileText, SlidersHorizontal, type LucideIcon } from '@lucide/vue'
import { NOTE, SETTINGS } from './workspace'

const KINDS: ReadonlyMap<string, LucideIcon> = new Map([
  [NOTE, FileText],
  [SETTINGS, SlidersHorizontal],
])

/** The icon of a tab kind, or null for a kind without one. */
export const findTabKindIcon = (kind: string): LucideIcon | null => KINDS.get(kind) ?? null
