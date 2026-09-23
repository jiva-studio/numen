/**
 * What each kind of source that is not a note is drawn as: the mark of the tab
 * it opens in, so a recording is the same thing in a list that it is once it is
 * open. A note is drawn by which kind of note it is.
 */
import { AudioLines, BookOpen, Globe, type LucideIcon } from '@lucide/vue'
import type { Source } from '../types'

const SOURCES: ReadonlyMap<Source, LucideIcon> = new Map([
  ['book', BookOpen],
  ['recording', AudioLines],
  ['url', Globe],
])

/** The icon for a source, and nothing for a file the vault holds no source for. */
export const iconOfSource = (kind: Source): LucideIcon | null => SOURCES.get(kind) ?? null
