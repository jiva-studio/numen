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
import { COMMAND, ROW, SETTING_COMMAND_PREFIX } from './commandIds'
import {
  AppWindow,
  ArrowRightLeft,
  Command,
  Copy,
  Download,
  CornerDownRight,
  CornerLeftUp,
  ExternalLink,
  File,
  FilePlus,
  FileText,
  FileX,
  FolderPlus,
  FolderRoot,
  Link,
  Link2,
  Link2Off,
  Navigation,
  PenLine,
  Plus,
  Puzzle,
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
  [COMMAND.openFile, FileText],
  [ROW.openDocument, File],
  [ROW.openNoteBeside, FileText],
  [COMMAND.createChildNote, CornerDownRight],
  [COMMAND.createParentNote, CornerLeftUp],
  [COMMAND.createJumpNote, ArrowRightLeft],
  [COMMAND.renameNote, Type],
  [COMMAND.copyFilePath, Copy],
  [COMMAND.removeFile, Trash2],
  [ROW.destroyFile, Trash2],
  [COMMAND.deleteText, FileX],
  [COMMAND.detachTab, ExternalLink],
  [COMMAND.unlinkTab, Link2Off],
  [COMMAND.linkTab, Link2],
  [COMMAND.moveTabToWindow, AppWindow],
  ['app:file.createText', FileText],
  ['app:link.import', Link],
  ['app:link.downloadMedia', Download],
  ['app:link.deleteMedia', Trash2],
  ['new', Plus],
  ['relation', Waypoints],
  ['newNote', FilePlus],
  ['newFolder', FolderPlus],
  ['rename', PenLine],
  [COMMAND.createNote, FilePlus],
  [COMMAND.createFolder, FolderPlus],
  [COMMAND.searchVault, Search],
  ['commands', Command],
  [COMMAND.closeTab, SquareX],
  [COMMAND.openSettings, Settings],
  [ROW.goToNote, Navigation],
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

const isPluginCommandId = (id: string): boolean => getCommandOwner(id).includes('.')

/** The icon for a command, and nothing where it has none. */
export const findIcon = (id: string): LucideIcon | null => {
  if (id.startsWith(SETTING_COMMAND_PREFIX)) return SlidersHorizontal
  if (isPluginCommandId(id)) return Puzzle
  return ICONS.get(id) ?? null
}
