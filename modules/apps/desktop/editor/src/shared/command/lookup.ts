/**
 * Search hit lookup, item formatting, and navigation destination mapping.
 */
import type { PaletteItem } from '@numen/ui'
import type { Source } from '../file'
import type { NoteType } from '../note'
import type { Words } from './search'

/** A run of a name or a passage, counted the way this window counts text. */
export interface Span {
  from: number
  to: number
}

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
  /** Which of four the note is. A heading carries the type of the note it stands in. */
  type: NoteType
}

/** One passage: the text around a hit, and where it came from. */
export interface Passage {
  path: string
  /** What the note is called. Empty for a source that is not a note. */
  title: string
  /**
   * Whether the text was read out of a note. It says what the palette offers
   * over the passage: a book is not a node, so there is nowhere to travel to.
   */
  isNote: boolean
  /** Which of four that note is. It says nothing about a source that is not one. */
  type: NoteType
  /** What the vault holds at that path, whatever sort of source it is. */
  kind: Source
  text: string
  /**
   * Where the hit stands in the source's own text, counted in bytes, which is
   * what the document it came out of is opened at.
   */
  start: number
  length: number
  /** Where the hit stands in the prose, counted from its first line. */
  line: number
  at: readonly Span[]
}

/**
 * Where an item chosen takes the person. A file names the file and the place
 * inside it, and nothing about which editor it opens in: a landing at a file
 * carries a line, one at a document a span of the source's own text.
 */
export interface SearchDestination {
  at: 'plex' | 'file' | 'document'
  path: string
  /** What the note is called, for a tab that has not been opened before. */
  title: string
  /** The line the item stands on, for a place inside a file. */
  line?: number
  /** The span of the source's own text to light, for a place in a document. */
  start?: number
  length?: number
}

/** The things that can be done to anything the palette turns up. */
export const PLEX = 'plex'
export const NOTE = 'note'
export const DOCUMENT = 'document'

/**
 * Where one item stands in the vault, and what it can be asked. A line of -1 is
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
  /** Which of four the note it stands in is, and nothing where it stands in none. */
  type: NoteType | null
  /** What the vault holds where it stands. */
  kind: Source
}

/** One item as it is drawn, beside where it stands and what it offers. */
export interface SearchRow {
  item: PaletteItem
  hit: SearchHit
}

export function createNameItem(one: NameMatch, words: Words): SearchRow {
  return one.heading
    ? {
        item: {
          id: `${one.path}#${one.line}`,
          title: one.heading,
          at: one.at,
          detail: one.title,
          actions: [
            { id: NOTE, text: words.readAt },
            { id: PLEX, text: words.travel },
          ],
        },
        hit: {
          path: one.path,
          title: one.title,
          line: one.line,
          start: 0,
          length: 0,
          offers: [NOTE, PLEX],
          type: one.type,
          kind: 'note',
        },
      }
    : {
        item: {
          id: one.path,
          title: one.title,
          at: one.at,
          actions: [
            { id: PLEX, text: words.travel },
            { id: NOTE, text: words.read },
          ],
        },
        hit: {
          path: one.path,
          title: one.title,
          line: -1,
          start: 0,
          length: 0,
          offers: [PLEX, NOTE],
          type: one.type,
          kind: 'note',
        },
      }
}

export function createPassageItem(group: string, one: Passage, words: Words): SearchRow {
  return {
    item: {
      id: `${group}:${one.path}:${one.start}`,
      title: one.title || one.path,
      detail: one.text,
      detailAt: one.at,
      actions: one.isNote
        ? [
            { id: NOTE, text: words.read },
            { id: PLEX, text: words.travel },
          ]
        : [{ id: DOCUMENT, text: words.readDocument }],
    },
    hit: {
      path: one.path,
      title: one.title,
      line: one.isNote ? one.line : -1,
      start: one.start,
      length: one.length,
      offers: one.isNote ? [NOTE, PLEX] : [DOCUMENT],
      type: one.isNote ? one.type : null,
      kind: one.kind,
    },
  }
}

export function resolveDestination(
  stands: SearchHit | undefined,
  action: string,
): SearchDestination | null {
  if (!stands || !stands.offers.includes(action)) return null
  const named = { path: stands.path, title: stands.title }
  if (action === PLEX) return { at: 'plex', ...named }
  if (action === DOCUMENT) {
    return { at: 'document', ...named, start: stands.start, length: stands.length }
  }
  return stands.line >= 0
    ? { at: 'file', ...named, line: stands.line }
    : { at: 'file', ...named }
}
