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
  ArrowRightLeft,
  Bot,
  Copy,
  CornerDownRight,
  CornerLeftUp,
  FilePlus,
  FileText,
  FolderOpen,
  FolderPlus,
  PenLine,
  Trash2,
  Type,
  Waypoints,
  type LucideIcon,
} from '@lucide/vue'

/** What each command is drawn as. A map, so an identity answers for itself. */
const ICONS: ReadonlyMap<string, LucideIcon> = new Map([
  ['read', FileText],
  ['travel', Waypoints],
  ['newNote', FilePlus],
  ['newFolder', FolderPlus],
  ['rename', PenLine],
  ['copy', Copy],
  ['reveal', FolderOpen],
  ['child', CornerDownRight],
  ['parent', CornerLeftUp],
  ['jump', ArrowRightLeft],
  ['title', Type],
  ['ask', Bot],
  ['remove', Trash2],
])

/** The icon for a command, and nothing where it has none. */
export const iconFor = (id: string): LucideIcon | null => ICONS.get(id) ?? null
