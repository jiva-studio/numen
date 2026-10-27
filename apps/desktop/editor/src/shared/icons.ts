/**
 * The icon drawn beside a command wherever it is offered. A command has one
 * icon, from Lucide, so both menus draw the same thing for the same thing.
 *
 * A command is named here by the identity it carries in `commands.ts` and in
 * what each tab does itself.
 */
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
  Globe,
  Navigation,
  Palette,
  PenLine,
  Plus,
  RefreshCw,
  Ruler,
  ScanText,
  Search,
  SlidersHorizontal,
  SpellCheck,
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
  ['importUrl', Globe],
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
  ['settings', SlidersHorizontal],
  ['first', Compass],
  ['goto', Navigation],
  ['openVault', FolderRoot],
  ['newVault', Plus],
  ['renameVault', PenLine],
  ['removeVault', X],
  ['eraseVault', Trash2],
])

/** The icons of the entries of a file's menu, by what each does. */
const FILE_MENU_ICONS: ReadonlyMap<string, LucideIcon> = new Map([
  ['recognise', ScanText],
  ['proofread', SpellCheck],
  ['delete-version', Trash2],
])

/** The icon for a command, and nothing where it has none. */
export const findIcon = (id: string): LucideIcon | null => {
  if (id.startsWith('plugin-setting:')) return SlidersHorizontal
  if (id.startsWith('file-menu:')) return FILE_MENU_ICONS.get(id.split(':')[1] ?? '') ?? null
  return ICONS.get(id) ?? null
}
