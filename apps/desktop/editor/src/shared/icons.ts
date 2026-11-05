/**
 * The icon drawn beside a command wherever it is offered. A command has one
 * icon, from Lucide, so both menus draw the same thing for the same thing.
 *
 * A command is named here by the identity it carries in `commands.ts` and in
 * what each tab does itself.
 */
import type { Component } from 'vue'
import { createNamedIcon } from '@numen/ui'
import { getCommandOwner } from '@numen/plugin-host'
import { COMMAND } from './commandIds'
import {
  ALargeSmall,
  ArrowRightLeft,
  Command,
  Contrast,
  Copy,
  CornerDownRight,
  CornerLeftUp,
  File,
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
  [COMMAND.openNote, FileText],
  [COMMAND.openFile, File],
  [COMMAND.openNoteBeside, FileText],
  [COMMAND.createChildNote, CornerDownRight],
  [COMMAND.createParentNote, CornerLeftUp],
  [COMMAND.createJumpNote, ArrowRightLeft],
  [COMMAND.renameNote, Type],
  [COMMAND.copyNotePath, Copy],
  [COMMAND.trashNote, Trash2],
  [COMMAND.deleteNote, Trash2],
  ['new', Plus],
  ['relation', Waypoints],
  ['newNote', FilePlus],
  ['newFolder', FolderPlus],
  ['rename', PenLine],
  [COMMAND.createNote, FilePlus],
  [COMMAND.searchVault, Search],
  ['commands', Command],
  [COMMAND.closeTab, SquareX],
  [COMMAND.chooseTheme, Palette],
  [COMMAND.chooseMode, Contrast],
  [COMMAND.chooseInterfaceSize, Ruler],
  [COMMAND.chooseTextSize, ALargeSmall],
  [COMMAND.chooseTitleSync, RefreshCw],
  [COMMAND.choosePerformanceProfile, Gauge],
  [COMMAND.openSettings, Settings],
  [COMMAND.goToNote, Navigation],
  [COMMAND.openVault, FolderRoot],
  [COMMAND.createVault, Plus],
  [COMMAND.renameVault, PenLine],
  [COMMAND.removeVault, X],
  [COMMAND.eraseVault, Trash2],
])

/**
 * The icon a plugin's manifest names, by its Lucide name. The same name gives the same component,
 * which draws the icon once the set is loaded, and nothing for a name with no icon.
 */
export const getNamedIcon = (name: string): Component => createNamedIcon(name)

/** Whether the id is a command of a plugin, whose owner is the dotted id of the plugin. */
const isPluginCommandId = (id: string): boolean => getCommandOwner(id).includes('.')

/** The icon for a command, and nothing where it has none. */
export const findIcon = (id: string): LucideIcon | null => {
  if (id.startsWith('plugin-setting:')) return SlidersHorizontal
  if (isPluginCommandId(id)) return Puzzle
  return ICONS.get(id) ?? null
}
