/**
 * Search hit lookup, item formatting, and navigation destination mapping.
 */
import type { PaletteItem } from '@numen/shell'
import type { PlacedCommand, Selector } from '@numen/plugin-host'
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
  at: 'file'
  path: string
  /** What the note is called, for a tab that has not been opened before. */
  title: string
  /** The line the item relies on, for a place inside a note. */
  line?: number
  /** The place in a file that is no note, as the search hit locates it. */
  selector?: Selector
}

/** The action that opens a note or a file the palette turns up. */
export const NOTE = 'note'

/** The prefix of the id of an action that runs an entry of the menu of a search result. */
export const MENU_ACTION_PREFIX = 'menu:'

/** The entries plugins place in the menu of a search result, in the order they are drawn. */
export interface SearchMenuDeps {
  getEntries(): readonly PlacedCommand[]
}

/** The command an action of a search result runs, and the note it runs over. */
export interface MenuChoice {
  readonly command: string
  readonly path: string
  readonly title: string
  readonly type: NoteType
  /** The line of the heading or passage the result stands for, and -1 for a note found by its name. */
  readonly line: number
}

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

type PaletteAction = NonNullable<PaletteItem['actions']>[number]

/** The actions of a note: opening it, and then the entries plugins place in the menu of a result. */
function createNoteActions(
  openText: string,
  menu: readonly PaletteAction[],
): { actions: readonly PaletteAction[]; offers: readonly string[] } {
  const actions = [{ id: NOTE, text: openText }, ...menu]
  return { actions, offers: actions.map((one) => one.id) }
}

function createMenuActions(entries: readonly PlacedCommand[]): readonly PaletteAction[] {
  return entries.map((one) => ({ id: `${MENU_ACTION_PREFIX}${one.command}`, text: one.title }))
}

/**
 * A name the search turned up. A name offers to open the note, and a heading offers to open the note
 * at that heading.
 */
export function createNameItem(
  one: NameMatch,
  words: Words,
  entries: readonly PlacedCommand[] = [],
): SearchRow {
  const menu = createMenuActions(entries)
  const atHeading = createNoteActions(words.readAt, menu)
  const byName = createNoteActions(words.read, menu)
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
  entries: readonly PlacedCommand[] = [],
): SearchRow {
  const note = createNoteActions(words.read, createMenuActions(entries))
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

/** The command the action runs over the note of the hit, and null where the action is none of the menu. */
export function resolveMenuChoice(
  searchHit: SearchHit | undefined,
  action: string,
): MenuChoice | null {
  if (!searchHit || searchHit.type === null || !action.startsWith(MENU_ACTION_PREFIX)) return null
  if (!searchHit.offers.includes(action)) return null
  return {
    command: action.slice(MENU_ACTION_PREFIX.length),
    path: searchHit.path,
    title: searchHit.title,
    type: searchHit.type,
  }
}

export function resolveDestination(
  hit: SearchHit | undefined,
  action: string,
): SearchDestination | null {
  if (!hit || action.startsWith(MENU_ACTION_PREFIX) || !hit.offers.includes(action)) return null
  const named = { path: hit.path, title: hit.title }
  if (hit.selector) return { at: 'file', ...named, selector: hit.selector }
  return hit.line >= 0 ? { at: 'file', ...named, line: hit.line } : { at: 'file', ...named }
}
