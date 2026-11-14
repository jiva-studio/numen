/** The icons of the tab kinds the window declares itself; a plugin kind carries the icon it registers. */
import { SlidersHorizontal, type LucideIcon } from '@lucide/vue'
import { SETTINGS } from './workspace'

const KINDS: ReadonlyMap<string, LucideIcon> = new Map([[SETTINGS, SlidersHorizontal]])

/** The icon of a tab kind, or null for a kind without one. */
export const findTabKindIcon = (kind: string): LucideIcon | null => KINDS.get(kind) ?? null
