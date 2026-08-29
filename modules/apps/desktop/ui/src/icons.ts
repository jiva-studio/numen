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
  BookOpen,
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
  FolderRoot,
  FolderTree,
  Layers,
  LayoutTemplate,
  ListTree,
  Navigation,
  Palette,
  PenLine,
  Plus,
  RefreshCw,
  Rows3,
  Ruler,
  Search,
  SquareX,
  Trash2,
  Type,
  Waypoints,
  X,
  type LucideIcon,
} from '@lucide/vue'
import { AGENT, DECK, DOCUMENT, FILES, NOTE, PLEX, STENCIL } from './workspace'

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
  ['newDeck', Layers],
  ['newStencil', LayoutTemplate],
  ['newFolder', FolderPlus],
  ['rename', PenLine],
  // Over the window.
  ['note', FilePlus],
  ['deck', Layers],
  ['stencil', LayoutTemplate],
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
  ['hanging', ListTree],
  ['parts', Rows3],
  // Over the vault.
  ['first', Compass],
  ['goto', Navigation],
  ['openVault', FolderRoot],
  ['newVault', Plus],
  ['renameVault', PenLine],
  ['forgetVault', X],
  ['eraseVault', Trash2],
])

/** The icon for a command, and nothing where it has none. */
export const iconFor = (id: string): LucideIcon | null => ICONS.get(id) ?? null

/** What each kind of tab is drawn as, before the name it carries. */
const KINDS: ReadonlyMap<string, LucideIcon> = new Map([
  [PLEX, Waypoints],
  [AGENT, Bot],
  [FILES, FolderTree],
  [NOTE, FileText],
  [DOCUMENT, BookOpen],
  [DECK, Layers],
  [STENCIL, LayoutTemplate],
])

/** The icon for a kind of tab, and nothing for a kind that has none. */
export const iconOfKind = (kind: string): LucideIcon | null => KINDS.get(kind) ?? null
