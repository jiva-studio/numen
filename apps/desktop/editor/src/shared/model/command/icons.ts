/**
 * The icon drawn beside a command wherever it is offered. A command has one
 * icon, from Lucide, so both menus draw the same thing for the same thing.
 *
 * A command is named here by the identity it carries in `ids.ts` and in
 * what each tab does itself.
 */
import { getCommandOwner } from '@numen/plugin-loader'
import { COMMAND, COMMANDS_ROW, ROW, SETTING_COMMAND_PREFIX } from './ids'
import {
  AppWindow,
  ArrowRightLeft,
  Command,
  Copy,
  CornerDownRight,
  CornerLeftUp,
  ExternalLink,
  File,
  FilePlus,
  FileText,
  FileX,
  FolderPlus,
  FolderRoot,
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
  // open
  [COMMAND.openFile, FileText],
  [ROW.openDocument, File],
  [ROW.openNoteBeside, FileText],
  [COMMAND.revealFile, Waypoints],
  [ROW.goToNote, Navigation],
  [COMMAND.searchVault, Search],
  [COMMAND.openSettings, Settings],
  [COMMANDS_ROW, Command],
  // create
  [COMMAND.createNote, FilePlus],
  [COMMAND.createFolder, FolderPlus],
  [COMMAND.createChildNote, CornerDownRight],
  [COMMAND.createParentNote, CornerLeftUp],
  [COMMAND.createJumpNote, ArrowRightLeft],
  // file
  [COMMAND.renameNote, Type],
  [COMMAND.copyFilePath, Copy],
  [COMMAND.removeFile, Trash2],
  [COMMAND.deleteText, FileX],
  // tab
  [COMMAND.closeTab, SquareX],
  [COMMAND.detachTab, ExternalLink],
  [COMMAND.unlinkTab, Link2Off],
  [COMMAND.linkTab, Link2],
  [COMMAND.moveTabToWindow, AppWindow],
  // vault
  [COMMAND.openVault, FolderRoot],
  [COMMAND.createVault, Plus],
  [COMMAND.renameVault, PenLine],
  [COMMAND.removeVault, X],
  [COMMAND.eraseVault, Trash2],
  [COMMAND.emptyTrash, Trash2],
  // shared commands: their ids are not constants
  ['app:file.createText', FileText],
])

const isPluginCommandId = (id: string): boolean => getCommandOwner(id).includes('.')

/** The icon for a command, and nothing where it has none. */
export const findIcon = (id: string): LucideIcon | null => {
  if (id.startsWith(SETTING_COMMAND_PREFIX)) return SlidersHorizontal
  if (isPluginCommandId(id)) return Puzzle
  return ICONS.get(id) ?? null
}
