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
  BookOpen,
  BookText,
  Bot,
  Braces,
  Captions,
  CaptionsOff,
  Command,
  Compass,
  Contrast,
  Copy,
  CornerDownRight,
  CornerLeftUp,
  FilePlus,
  FileText,
  File,
  Folder,
  FolderOpen,
  FolderPlus,
  FolderRoot,
  FolderTree,
  Gauge,
  Globe,
  HardDriveDownload,
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
import type { Entry, NoteType, Source } from './core'
import {
  AGENT,
  BOOK,
  DECK,
  DOCUMENT,
  FILES,
  NOTE,
  PLEX,
  PRESET,
  RECORDING,
  SETTINGS,
  SETTINGS_FILE,
  STENCIL,
  URL,
} from './tabs/workspace'

/** What each command is drawn as. A map, so an identity answers for itself. */
const ICONS: ReadonlyMap<string, LucideIcon> = new Map([
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
  ['preset', Gauge],
  ['remove', Trash2],
  ['destroy', Trash2],
  ['transcribe', Captions],
  ['proofread', SpellCheck],
  ['recognise', ScanText],
  ['downloadText', RefreshCw],
  ['downloadCopy', HardDriveDownload],
  ['deleteText', CaptionsOff],
  ['deleteCopy', Trash2],
  ['newNote', FilePlus],
  ['newDeck', Layers],
  ['newStencil', LayoutTemplate],
  ['newPreset', Gauge],
  ['importUrl', Globe],
  ['newFolder', FolderPlus],
  ['rename', PenLine],
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
  ['settings', SlidersHorizontal],
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
  [BOOK, BookText],
  [RECORDING, AudioLines],
  [URL, Globe],
  [DECK, Layers],
  [STENCIL, LayoutTemplate],
  [PRESET, Gauge],
  [SETTINGS, SlidersHorizontal],
  [SETTINGS_FILE, Braces],
])

/** The icon for a kind of tab, and nothing for a kind that has none. */
export const iconOfKind = (kind: string): LucideIcon | null => KINDS.get(kind) ?? null

/**
 * What each kind of note is drawn as, wherever a note's kind is drawn: the
 * files list, the plex, and the tab it opens in. One mark to a kind, so a
 * preset is the same thing in the tree that it is in the tab.
 */
const NOTES: ReadonlyMap<NoteType, LucideIcon> = new Map([
  ['note', FileText],
  ['deck', Layers],
  ['stencil', LayoutTemplate],
  ['preset', Gauge],
])

/** The icon for a kind of note. Every kind has one. */
export const iconOfNote = (type: NoteType): LucideIcon => NOTES.get(type) ?? FileText

/**
 * What each kind of source that is not a note is drawn as: the mark of the tab
 * it opens in, so a recording is the same thing in a list that it is once it is
 * open. A note is drawn by which kind of note it is.
 */
const SOURCES: ReadonlyMap<Source, LucideIcon> = new Map([
  ['book', BookOpen],
  ['recording', AudioLines],
  ['url', Globe],
])

/** The icon for a source, and nothing for a file the vault holds no source for. */
export const iconOfSource = (kind: Source): LucideIcon | null => SOURCES.get(kind) ?? null

/**
 * The icon one entry of the files tree is drawn as. A folder says whether what
 * it holds is drawn; a note is drawn by which kind of note it is, and every
 * other file by the source the vault holds it as.
 */
export const iconOfEntry = (entry: Entry | null | undefined, open: boolean): LucideIcon => {
  if (!entry) return File
  if (entry.folder) return open ? FolderOpen : Folder
  if (entry.kind === 'note') return iconOfNote(entry.type)
  return iconOfSource(entry.kind) ?? File
}
