/**
 * Search hit lookup, item formatting, and navigation destination mapping.
 */
import type { PaletteItem } from '@numen/shell'
import type { Selector } from '@numen/plugin-sdk'
import type { NoteType } from '@/entities/file'
import type { TextRange } from '@/shared/textRange'
import type { Words } from './search'

/**
 * One name that matched: a note title or an internal heading.
 */
export interface NameMatch {
  path: string
  title: string
  heading: string
  /** Position of that heading, counted from the first line of the prose. */
  line: number
  at: readonly TextRange[]
  type: NoteType
}

/** One passage: the text around a hit, and where it came from. */
export interface Passage {
  path: string
  /** What the note is called. Empty for a source that is not a note. */
  title: string
  /**
   * Whether the text was read from a note. It determines what the palette
   * offers over the passage: a book is no note, so it is not presented as one.
   */
  isNote: boolean
  /** The note type. It is meaningless for a file that is no note. */
  type: NoteType
  /** Where in a file that is no note the text was found, and null for a note. */
  selector: Selector | null
  text: string
  /**
   * what the document it came out of is opened at.
   */
  start: number
  length: number
  line: number
  at: readonly TextRange[]
}

/**
 * Where a chosen item leads. A file names the file and the position inside it,
 * and nothing about which editor it opens in: a note carries a line, any other
 * file the selector its reader moves to.
 */
export interface SearchDestination {
  at: 'locate' | 'file'
  path: string
  /** What the note is called, for a tab that has not been opened before. */
  title: string
  /** The line the item relies on, for a place inside a note. */
  line?: number
  /** The place in a file that is no note, as the search hit locates it. */
  selector?: Selector
}

/**
 * The things that can be done to anything the palette turns up. A note is put
 * in front where the plugin that shows notes puts it, which only a window with
 * such a plugin offers.
 */
export const LOCATE = 'locate'
export const NOTE = 'note'

/**
 * no line at all.
 */
export interface SearchHit {
  path: string
  title: string
  line: number
  /** The span of the source's own text the item was found in. */
  start: number
  length: number
  offers: readonly string[]
  type: NoteType | null
  /** The place in a file that is no note, and null for a note. */
  selector: Selector | null
}

/** One item as it is drawn, beside at its current position and what it offers. */
export interface SearchRow {
  item: PaletteItem
  hit: SearchHit
}

/** The search results over a note, including the action that shows it where one exists. */
const getNoteActions = (travel: string, first: { id: string; text: string }) => ({
  actions: travel ? [first, { id: LOCATE, text: travel }] : [first],
  offers: travel ? [first.id, LOCATE] : [first.id],
})

/**
 * A name the search turned up. A note found by its name is put in front first,
 * and a heading is opened at first. The last argument is the words of the action
 * that puts a note in front, and empty where nothing does.
 */
export function createNameItem(one: NameMatch, words: Words, travel: string): SearchRow {
  const atHeading = getNoteActions(travel, { id: NOTE, text: words.readAt })
  const read = { id: NOTE, text: words.read }
  const byName = travel
    ? { actions: [{ id: LOCATE, text: travel }, read], offers: [LOCATE, NOTE] }
    : { actions: [read], offers: [NOTE] }
  return one.heading
    ? {
        item: {
          id: `${one.path}#${one.line}`,
          title: one.heading,
          at: one.at,
          detail: one.title,
          actions: atHeading.actions,
        },
        hit: {
          path: one.path,
          title: one.title,
          line: one.line,
          start: 0,
          length: 0,
          offers: atHeading.offers,
          type: one.type,
          selector: null,
        },
      }
    : {
        item: {
          id: one.path,
          title: one.title,
          at: one.at,
          actions: byName.actions,
        },
        hit: {
          path: one.path,
          title: one.title,
          line: -1,
          start: 0,
          length: 0,
          offers: byName.offers,
          type: one.type,
          selector: null,
        },
      }
}

export function createPassageItem(
  group: string,
  one: Passage,
  words: Words,
  travel: string,
): SearchRow {
  const note = getNoteActions(travel, { id: NOTE, text: words.read })
  return {
    item: {
      id: `${group}:${one.path}:${one.start}`,
      title: one.title || one.path,
      detail: one.text,
      detailAt: one.at,
      actions: one.isNote ? note.actions : [{ id: NOTE, text: words.readDocument }],
    },
    hit: {
      path: one.path,
      title: one.title,
      line: one.isNote ? one.line : -1,
      start: one.start,
      length: one.length,
      offers: one.isNote ? note.offers : [NOTE],
      type: one.isNote ? one.type : null,
      selector: one.isNote ? null : one.selector,
    },
  }
}

export function resolveDestination(
  hit: SearchHit | undefined,
  action: string,
): SearchDestination | null {
  if (!hit || !hit.offers.includes(action)) return null
  const named = { path: hit.path, title: hit.title }
  if (action === LOCATE) return { at: 'locate', ...named }
  if (hit.selector) return { at: 'file', ...named, selector: hit.selector }
  return hit.line >= 0 ? { at: 'file', ...named, line: hit.line } : { at: 'file', ...named }
}
