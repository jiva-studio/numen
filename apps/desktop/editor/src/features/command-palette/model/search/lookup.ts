/**
 * Search hit lookup, item formatting, and navigation destination mapping.
 */
import type { PaletteItem } from '@numen/ui'
import type { NoteType, Source } from '@/entities/file'
import type { Span } from '@/shared/span'
import type { Words } from './search'

/**
 * One name that matched: a note title or an internal heading.
 */
export interface NameMatch {
  path: string
  title: string
  heading: string
  /** Where that heading stands, counted from the first line of the prose. */
  line: number
  at: readonly Span[]
  type: NoteType
}

/** One passage: the text around a hit, and where it came from. */
export interface Passage {
  path: string
  /** What the note is called. Empty for a source that is not a note. */
  title: string
  /**
   * Whether the text was read out of a note. It says what the palette offers
   * over the passage: a book is no note, so nothing puts it in front as one.
   */
  isNote: boolean
  /** Which of four that note is. It says nothing about a source that is not one. */
  type: NoteType
  /** What the vault holds at that path, whatever sort of source it is. */
  kind: Source
  text: string
  /**
   * what the document it came out of is opened at.
   */
  start: number
  length: number
  line: number
  at: readonly Span[]
}

/**
 * Where an item chosen takes the person. A file names the file and the place
 * inside it, and nothing about which editor it opens in: a landing at a file
 * carries a line, one at a document a span of the source's own text.
 */
export interface SearchDestination {
  at: 'travel' | 'file' | 'document'
  path: string
  /** What the note is called, for a tab that has not been opened before. */
  title: string
  /** The line the item relies on, for a place inside a file. */
  line?: number
  /** The span of the source's own text to light, for a place in a document. */
  start?: number
  length?: number
}

/**
 * The things that can be done to anything the palette turns up. A note is put
 * in front where the plugin that shows notes puts it, which only a window with
 * such a plugin offers.
 */
export const TRAVEL = 'travel'
export const NOTE = 'note'
export const DOCUMENT = 'document'

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
  /** What the vault holds at its current position. */
  kind: Source
}

/** One item as it is drawn, beside at its current position and what it offers. */
export interface SearchRow {
  item: PaletteItem
  hit: SearchHit
}

/** What the search says over a note, the action that puts it in front among them where there is one. */
const getNoteActions = (travel: string, first: { id: string; text: string }) => ({
  actions: travel ? [first, { id: TRAVEL, text: travel }] : [first],
  offers: travel ? [first.id, TRAVEL] : [first.id],
})

/**
 * A name the search turned up. A note found by its name is put in front first,
 * and a heading is opened at first. `travel` is the words of the action that puts
 * a note in front, and empty where nothing does.
 */
export function createNameItem(one: NameMatch, words: Words, travel: string): SearchRow {
  const atHeading = getNoteActions(travel, { id: NOTE, text: words.readAt })
  const read = { id: NOTE, text: words.read }
  const byName = travel
    ? { actions: [{ id: TRAVEL, text: travel }, read], offers: [TRAVEL, NOTE] }
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
          kind: 'note',
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
          kind: 'note',
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
      actions: one.isNote ? note.actions : [{ id: DOCUMENT, text: words.readDocument }],
    },
    hit: {
      path: one.path,
      title: one.title,
      line: one.isNote ? one.line : -1,
      start: one.start,
      length: one.length,
      offers: one.isNote ? note.offers : [DOCUMENT],
      type: one.isNote ? one.type : null,
      kind: one.kind,
    },
  }
}

export function resolveDestination(
  hit: SearchHit | undefined,
  action: string,
): SearchDestination | null {
  if (!hit || !hit.offers.includes(action)) return null
  const named = { path: hit.path, title: hit.title }
  if (action === TRAVEL) return { at: 'travel', ...named }
  if (action === DOCUMENT) {
    return { at: 'document', ...named, start: hit.start, length: hit.length }
  }
  return hit.line >= 0 ? { at: 'file', ...named, line: hit.line } : { at: 'file', ...named }
}
