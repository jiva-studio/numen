/**
 * The icons of what the menu on a node offers of the plex's own. A command a
 * plugin declares is drawn with the icon the window gives it.
 */
import {
  ArrowRightLeft,
  Copy,
  CornerDownRight,
  CornerLeftUp,
  FilePlus,
  FileText,
  Trash2,
  Type,
  type LucideIcon,
} from '@lucide/vue'

const COMMAND_ICONS: ReadonlyMap<string, LucideIcon> = new Map([
  ['read', FileText],
  ['child', CornerDownRight],
  ['parent', CornerLeftUp],
  ['jump', ArrowRightLeft],
  ['title', Type],
  ['copy', Copy],
  ['remove', Trash2],
  ['newNote', FilePlus],
])

export const findIcon = (id: string): LucideIcon | null => COMMAND_ICONS.get(id) ?? null
