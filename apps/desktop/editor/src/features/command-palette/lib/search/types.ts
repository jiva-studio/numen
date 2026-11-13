/**
 * What the palette searches for and what the core answers, in the palette's own types.
 */
import type { Selector } from '@numen/plugin-loader'
import type { NoteType } from '@/entities/file'
import type { TextRange } from '@/shared/lib/textRange'

/**
 * One name that matched: a note title or an internal heading.
 */
export interface NameMatch {
  path: string
  title: string
  heading: string
  line: number
  at: readonly TextRange[]
  type: NoteType
  tabKind?: string
}

/** One passage: the text around a hit, and where it came from. */
export interface Passage {
  path: string
  title: string
  isNote: boolean
  type: NoteType
  tabKind?: string
  selector: Selector | null
  text: string
  start: number
  length: number
  line: number
  at: readonly TextRange[]
}

/**
 * How a text search is run. Each mode has its own ordering, and `hybrid`
 * combines all of them in one ranking.
 */
export type SearchMode = 'hybrid' | 'words' | 'meaning' | 'names'

/** The two queries the palette makes of the vault. An aborted signal stops waiting for the answer. */
export interface SearchDeps {
  names(query: string, limit: number, signal?: AbortSignal): Promise<readonly NameMatch[]>
  search(
    query: string,
    mode: SearchMode,
    limit: number,
    signal?: AbortSignal,
  ): Promise<readonly Passage[]>
}
