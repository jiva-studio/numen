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
  AudioLines,
  Captions,
  Command,
  Compass,
  Contrast,
  Copy,
  CornerDownRight,
  CornerLeftUp,
  Download,
  FilePlus,
  FileText,
  FolderPlus,
  FolderRoot,
  Gauge,
  Languages,
  Layers,
  LayoutTemplate,
  Link,
  Navigation,
  Palette,
  PenLine,
  Plus,
  Puzzle,
  RefreshCw,
  Ruler,
  ScanText,
  Search,
  Settings,
  SlidersHorizontal,
  SpellCheck,
  Sparkles,
  SquareX,
  TextSearch,
  Trash2,
  Type,
  Waypoints,
  WandSparkles,
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
 * The icons a plugin's manifest can name, by their Lucide names. The window draws only the icons it
 * bundles, so a name not listed here is drawn as a puzzle piece.
 */
const NAMED_ICONS: ReadonlyMap<string, LucideIcon> = new Map([
  ['audio-lines', AudioLines],
  ['captions', Captions],
  ['download', Download],
  ['file-text', FileText],
  ['gauge', Gauge],
  ['languages', Languages],
  ['layers', Layers],
  ['layout-template', LayoutTemplate],
  ['link', Link],
  ['refresh-cw', RefreshCw],
  ['scan-text', ScanText],
  ['settings', Settings],
  ['sliders-horizontal', SlidersHorizontal],
  ['sparkles', Sparkles],
  ['spell-check', SpellCheck],
  ['text-search', TextSearch],
  ['trash-2', Trash2],
  ['wand-sparkles', WandSparkles],
  ['waypoints', Waypoints],
])

/** The icon a plugin's manifest names. */
export const getNamedIcon = (name: string): LucideIcon => NAMED_ICONS.get(name) ?? Puzzle

/** The icon for a command, and nothing where it has none. */
export const findIcon = (id: string): LucideIcon | null => {
  if (id.startsWith('plugin-setting:')) return SlidersHorizontal
  if (id.startsWith('plugin-command:')) return Puzzle
  if (id.startsWith('file-menu:delete-version:')) return Trash2
  return ICONS.get(id) ?? null
}
