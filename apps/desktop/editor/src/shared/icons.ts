/**
 * The icon drawn beside a command wherever it is offered. A command has one
 * icon, from Lucide, so both menus draw the same thing for the same thing.
 *
 * A command is named here by the identity it carries in `commands.ts` and in
 * what each tab does itself.
 */
import type { Component } from 'vue'
import { createNamedIcon } from '@numen/ui'
import {
  ALargeSmall,
  ArrowRightLeft,
  Command,
  Compass,
  Contrast,
  Copy,
  CornerDownRight,
  CornerLeftUp,
  FilePlus,
  FileText,
  FolderPlus,
  FolderRoot,
  Gauge,
  Navigation,
  Palette,
  PenLine,
  Plus,
  Puzzle,
  RefreshCw,
  Ruler,
  Search,
  Settings,
  SlidersHorizontal,
  SquareX,
  Trash2,
  Type,
  Waypoints,
  X,
  type LucideIcon,
} from '@lucide/vue'

/** What each command is drawn as. A map, so an identity answers for itself. */
const ICONS: ReadonlyMap<string, LucideIcon> = new Map([
  ['read', FileText],
  ['beside', FileText],
  ['child', CornerDownRight],
  ['parent', CornerLeftUp],
  ['jump', ArrowRightLeft],
  ['title', Type],
  ['copy', Copy],
  ['remove', Trash2],
  ['destroy', Trash2],
  ['new', Plus],
  ['relation', Waypoints],
  ['newNote', FilePlus],
  ['newFolder', FolderPlus],
  ['rename', PenLine],
  ['note', FilePlus],
  ['find', Search],
  ['commands', Command],
  ['close', SquareX],
  ['appearance', Palette],
  ['mode', Contrast],
  ['interfaceScale', Ruler],
  ['textScale', ALargeSmall],
  ['syncing', RefreshCw],
  ['performanceProfile', Gauge],
  ['settings', Settings],
  ['first', Compass],
  ['goto', Navigation],
  ['openVault', FolderRoot],
  ['newVault', Plus],
  ['renameVault', PenLine],
  ['removeVault', X],
  ['eraseVault', Trash2],
])

/**
 * The icon a plugin's manifest names, by its Lucide name. The same name gives the same component,
 * which draws the icon once the set is loaded, and nothing for a name with no icon.
 */
export const getNamedIcon = (name: string): Component => createNamedIcon(name)

/** The icon for a command, and nothing where it has none. */
export const findIcon = (id: string): LucideIcon | null => {
  if (id.startsWith('plugin-setting:')) return SlidersHorizontal
  if (id.startsWith('plugin-command:')) return Puzzle
  return ICONS.get(id) ?? null
}
