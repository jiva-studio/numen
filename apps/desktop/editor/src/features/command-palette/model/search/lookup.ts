/**
 * Search hit lookup, item formatting, and the command a chosen action of a hit runs.
 */
import type { PaletteItem } from '@numen/shell'
import type { Selector, SlotItem, SlotTarget } from '@numen/plugin-host'
import type { NoteType } from '@/entities/file'
import { COMMAND } from '@/shared/commandIds'
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

/** The entries of the slot of a search result, over the note of a result, in the order drawn. */
export interface SearchMenuDeps {
  getEntries(target: SlotTarget): readonly SlotItem[]
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
  actionIds: readonly string[]
  type: NoteType | null
  /** The place in a file that is no note, and null for a note. */
  selector: Selector | null
  /** Whether the hit is the row that creates a note under the words typed, which is no result. */
  isCreation?: boolean
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
): { actions: readonly PaletteAction[]; actionIds: readonly string[] } {
  const actions = [{ id: COMMAND.openNote, text: openText }, ...menu]
  return { actions, actionIds: actions.map((one) => one.id) }
}

/** The actions of the commands among the entries; a view, a submenu and a separator offer none. */
function createMenuActions(entries: readonly SlotItem[]): readonly PaletteAction[] {
  return entries.flatMap((one) =>
    one.kind === 'command' ? [{ id: one.command, text: one.title }] : [],
  )
}

/**
 * A name the search turned up. A name offers to open the note, and a heading offers to open the note
 * at that heading.
 */
export function createNameItem(
  one: NameMatch,
  words: Words,
  entries: readonly SlotItem[] = [],
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
          actionIds: atHeading.actionIds,
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
          actionIds: byName.actionIds,
          type: one.type,
          selector: null,
        },
      }
}

export function createPassageItem(
  group: string,
  one: Passage,
  words: Words,
  entries: readonly SlotItem[] = [],
): SearchRow {
  const note = createNoteActions(words.read, createMenuActions(entries))
  const document = [{ id: COMMAND.openFile, text: words.readDocument }]
  return {
    item: {
      id: `${group}:${one.path}:${one.start}`,
      title: one.title || one.path,
      detail: one.text,
      detailAt: one.at,
      actions: one.isNote ? note.actions : document,
    },
    hit: {
      path: one.path,
      title: one.title,
      line: one.isNote ? one.line : -1,
      start: one.start,
      length: one.length,
      actionIds: one.isNote ? note.actionIds : document.map((each) => each.id),
      type: one.isNote ? one.type : null,
      selector: one.isNote ? null : one.selector,
    },
  }
}

/** What a chosen action runs: the command, what it is over, and the text typed for it. */
export interface ResultChoice {
  readonly command: string
  /** The chosen result as a target; null: the target the window holds. */
  readonly over: {
    readonly path: string
    readonly file: string
    readonly title: string
    readonly line?: number
    readonly selector?: Selector
  } | null
  /** The text typed for it, which answers a step that asks for a name; '' where none. */
  readonly input: string
}

/**
 * The command an action of a hit runs, over what, with which input; null where the hit does not
 * carry the action. A note is the target by its path and a line, a file that is no note by its
 * file and selector, and the row that creates a note by the window and the words typed.
 */
export function resolveChoice(
  hit: SearchHit | undefined,
  action: string,
  typed = '',
): ResultChoice | null {
  if (!hit || !hit.actionIds.includes(action)) return null
  if (hit.isCreation) {
    return {
      command: action,
      over: null,
      input: typed.trim(),
    }
  }
  if (hit.type === null) {
    if (action !== COMMAND.openFile) return null
    const place = { path: '', file: hit.path, title: hit.title }
    return {
      command: action,
      over: hit.selector ? { ...place, selector: hit.selector } : place,
      input: '',
    }
  }
  const place = { path: hit.path, file: hit.path, title: hit.title }
  return {
    command: action,
    over: hit.line >= 0 ? { ...place, line: hit.line } : place,
    input: '',
  }
}
