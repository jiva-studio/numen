/** What each kind of tab is drawn as, before the name it carries. */
import {
  AudioLines,
  Bot,
  BookOpen,
  BookText,
  Braces,
  FileText,
  FolderTree,
  Gauge,
  Globe,
  Layers,
  LayoutTemplate,
  SlidersHorizontal,
  type LucideIcon,
} from '@lucide/vue'
import {
  AGENT,
  BOOK,
  DECK,
  DOCUMENT,
  FILES,
  NOTE,
  PRESET,
  RECORDING,
  SETTINGS,
  SETTINGS_FILE,
  STENCIL,
  URL,
} from './workspace'

const KINDS: ReadonlyMap<string, LucideIcon> = new Map([
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
export const findTabKindIcon = (kind: string): LucideIcon | null => KINDS.get(kind) ?? null
