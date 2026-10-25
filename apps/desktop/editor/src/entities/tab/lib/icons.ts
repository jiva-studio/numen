/** What each kind of tab is drawn as, before the name it carries. */
import {
  AudioLines,
  BookOpen,
  BookText,
  Braces,
  FileText,
  Globe,
  SlidersHorizontal,
  type LucideIcon,
} from '@lucide/vue'
import { BOOK, DOCUMENT, NOTE, RECORDING, SETTINGS, SETTINGS_FILE, URL } from './workspace'

const KINDS: ReadonlyMap<string, LucideIcon> = new Map([
  [NOTE, FileText],
  [DOCUMENT, BookOpen],
  [BOOK, BookText],
  [RECORDING, AudioLines],
  [URL, Globe],
  [SETTINGS, SlidersHorizontal],
  [SETTINGS_FILE, Braces],
])

/** The icon for a kind of tab, and nothing for a kind that has none. */
export const findTabKindIcon = (kind: string): LucideIcon | null => KINDS.get(kind) ?? null
