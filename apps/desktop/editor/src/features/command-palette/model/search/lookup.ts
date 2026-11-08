/**
 * Search hit lookup, item formatting, and the command a chosen action of a hit runs.
 */
import type { PaletteItem } from '@numen/shell'
import type { Selector, SlotItem, CommandTarget as RunTarget } from '@numen/plugin-host'
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

/** The entries of the slot of a search result, over the note of a result, in the order drawn. */
export interface SearchMenuDeps {
  getEntries(target: RunTarget): readonly SlotItem[]
}

/**
 * no line at all.
 */
export interface SearchHit {
  path: string
  title: string
  line: number
  start: number
  length: number
  actionIds: readonly string[]
  type: NoteType | null
  tabKind?: string
  selector: Selector | null
  isCreation?: boolean
}

/** One item as it is drawn, beside at its current position and what it offers. */
export interface SearchRow {
  item: PaletteItem
  hit: SearchHit
}

type PaletteAction = NonNullable<PaletteItem['actions']>[number]

function createNoteActions(
  openText: string,
  menu: readonly PaletteAction[],
): { actions: readonly PaletteAction[]; actionIds: readonly string[] } {
  const actions = [{ id: COMMAND.openNote, text: openText }, ...menu]
  return { actions, actionIds: actions.map((one) => one.id) }
}

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
          ...(one.tabKind ? { tabKind: one.tabKind } : {}),
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
          ...(one.tabKind ? { tabKind: one.tabKind } : {}),
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
  readonly over: {
    readonly path: string
    readonly file: string
    readonly title: string
    readonly line?: number
    readonly selector?: Selector
  } | null
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
