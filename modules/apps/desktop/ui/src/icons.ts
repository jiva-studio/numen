/**
 * The icon drawn beside a command wherever it is offered.
 *
 * A command has one icon, so the menu on a row of the tree and the menu on a
 * node of the plex draw the same thing for the same thing. The icons are
 * Lucide's, which is the set the palette's key caps are drawn from.
 *
 * A command is named here by the identity it carries in `commanding.ts` and in
 * what each tab does itself. `icons.test.ts` asks that every item either menu
 * offers has one.
 */
import {
  ALargeSmall,
  ArrowRightLeft,
  Bot,
  Command,
  Compass,
  Contrast,
  Copy,
  CornerDownRight,
  CornerLeftUp,
  FilePlus,
  FileText,
  FolderOpen,
  FolderPlus,
  FolderTree,
  Navigation,
  Palette,
  PenLine,
  Plus,
  RefreshCw,
  Ruler,
  Search,
  SquareX,
  Trash2,
  Type,
  Vault,
  Waypoints,
  X,
  type LucideIcon,
} from '@lucide/vue'

/** What each command is drawn as. A map, so an identity answers for itself. */
const ICONS: ReadonlyMap<string, LucideIcon> = new Map([
  // Over the note in front.
  ['read', FileText],
  ['beside', FileText],
  ['travel', Waypoints],
  ['child', CornerDownRight],
  ['parent', CornerLeftUp],
  ['jump', ArrowRightLeft],
  ['title', Type],
  ['ask', Bot],
  ['copy', Copy],
  ['reveal', FolderOpen],
  ['remove', Trash2],
  ['destroy', Trash2],
  // What a tab of the tree does itself.
  ['newNote', FilePlus],
  ['newFolder', FolderPlus],
  ['rename', PenLine],
  // Over the window.
  ['note', FilePlus],
  ['plex', Waypoints],
  ['agent', Bot],
  ['files', FolderTree],
  ['find', Search],
  ['commands', Command],
  ['close', SquareX],
  ['appearance', Palette],
  ['mode', Contrast],
  ['interfaceScale', Ruler],
  ['textScale', ALargeSmall],
  ['syncing', RefreshCw],
  // Over the vault.
  ['first', Compass],
  ['goto', Navigation],
  ['openVault', Vault],
  ['newVault', Plus],
  ['renameVault', PenLine],
  ['forgetVault', X],
  ['eraseVault', Trash2],
])

/** The icon for a command, and nothing where it has none. */
export const iconFor = (id: string): LucideIcon | null => ICONS.get(id) ?? null
