/**
 * Search hit lookup, item formatting, and the command a chosen action of a hit runs.
 */
import type { PaletteItem } from '@numen/workbench'
import type { Selector, SlotItem, CommandTarget as RunTarget } from '@numen/plugin-loader'
import type { NoteType } from '@/entities/file'
import { COMMAND, ROW } from '@/shared/model/command'
import type { NameMatch, Passage } from '../../lib/search/types'
import type { Words } from './search'

export type { NameMatch, Passage }

/** The entries of the slot of a search result, over the note of a result, in the order drawn. */
export interface SearchMenuDeps {
  getEntries(target: RunTarget): readonly SlotItem[]
}

/**
 * no line at all.
 */
export interface SearchHit {
  spot: HitSpot
  actionIds: readonly string[]
  type: NoteType | null
  tabKind?: string
  selector: Selector | null
  creationIds?: readonly string[]
  actionValues?: Readonly<Record<string, Readonly<Record<string, string>>>>
  actionCommands?: Readonly<Record<string, string>>
}

/** Where a hit is: its file, the title shown for it, and the line and the span of the text found. */
interface HitSpot {
  path: string
  title: string
  line: number
  start: number
  length: number
}


/** One item as it is drawn, beside at its current position and what it offers. */
export interface SearchRow {
  item: PaletteItem
  hit: SearchHit
}

type PaletteAction = NonNullable<PaletteItem['actions']>[number]

function createNoteActions(
  openText: string,
  entries: readonly SlotItem[],
): {
  actions: readonly PaletteAction[]
  actionIds: readonly string[]
  valuesOfHit: Pick<SearchHit, 'actionValues' | 'actionCommands'>
} {
  const commands = entries
    .filter((one) => one.kind === 'command')
    .map((one) => ({ one, id: getActionId(one.command, one.values) }))
  const menu = commands.map(({ one, id }) => ({ id, text: one.title }))
  const actions = [{ id: COMMAND.openFile, text: openText }, ...menu]
  const withValues = commands.flatMap(({ one, id }) =>
    one.values && Object.keys(one.values).length > 0 ? [[id, one.values] as const] : [],
  )
  const otherIds = commands.filter(({ one, id }) => id !== one.command)
  return {
    actions,
    actionIds: actions.map((one) => one.id),
    valuesOfHit: {
      ...(withValues.length > 0 ? { actionValues: Object.fromEntries(withValues) } : {}),
      ...(otherIds.length > 0
        ? { actionCommands: Object.fromEntries(otherIds.map(({ one, id }) => [id, one.command])) }
        : {}),
    },
  }
}

function getActionId(command: string, values: Readonly<Record<string, string>> | undefined) {
  const names = Object.keys(values ?? {}).sort()
  if (!values || names.length === 0) return command
  const pairs = names.map(
    (name) => `${encodeURIComponent(name)}=${encodeURIComponent(values[name] ?? '')}`,
  )
  return `${command}?${pairs.join('&')}`
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
  const atHeading = createNoteActions(words.readAt, entries)
  const byName = createNoteActions(words.read, entries)
  return one.heading
    ? {
        item: {
          id: `${one.path}#${one.line}`,
          title: one.heading,
          at: one.at,
          detail: one.title,
          actions: atHeading.actions,
        },
        hit: getNoteHit(
          { path: one.path, title: one.title, line: one.line, start: 0, length: 0 },
          atHeading,
          one,
        ),
      }
    : {
        item: {
          id: one.path,
          title: one.title,
          at: one.at,
          actions: byName.actions,
        },
        hit: getNoteHit(
          { path: one.path, title: one.title, line: -1, start: 0, length: 0 },
          byName,
          one,
        ),
      }
}

function getNoteHit(
  spot: HitSpot,
  note: ReturnType<typeof createNoteActions>,
  found: Pick<NameMatch, 'type' | 'tabKind'>,
): SearchHit {
  return {
    spot,
    actionIds: note.actionIds,
    ...note.valuesOfHit,
    type: found.type,
    ...(found.tabKind ? { tabKind: found.tabKind } : {}),
    selector: null,
  }
}

function createNotePassageHit(
  one: Passage,
  note: ReturnType<typeof createNoteActions>,
): SearchHit {
  const spot = { path: one.path, title: one.title, line: one.line, start: one.start, length: one.length }
  return getNoteHit(spot, note, one)
}

function createDocumentPassageHit(one: Passage, actionIds: readonly string[]): SearchHit {
  return {
    spot: { path: one.path, title: one.title, line: -1, start: one.start, length: one.length },
    actionIds,
    type: null,
    selector: one.selector,
  }
}

export function createPassageItem(
  group: string,
  one: Passage,
  words: Words,
  entries: readonly SlotItem[] = [],
): SearchRow {
  const note = createNoteActions(words.read, entries)
  const document = [{ id: ROW.openDocument, text: words.readDocument }]
  return {
    item: {
      id: `${group}:${one.path}:${one.start}`,
      title: one.title || one.path,
      detail: one.text,
      detailAt: one.at,
      actions: one.isNote ? note.actions : document,
    },
    hit: one.isNote
      ? createNotePassageHit(one, note)
      : createDocumentPassageHit(
          one,
          document.map((each) => each.id),
        ),
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
  readonly values?: Readonly<Record<string, string>>
}

function resolveNoteChoice(result: SearchHit, action: string): ResultChoice {
  const { path, title, line } = result.spot
  const place = { path, file: path, title }
  const values = result.actionValues?.[action]
  return {
    command: result.actionCommands?.[action] ?? action,
    over: line >= 0 ? { ...place, line } : place,
    input: '',
    ...(values ? { values } : {}),
  }
}

/**
 * The command an action of a hit runs, over what, with which input; null where the hit does not
 * carry the action. A note is the target by its path and a line, a file that is no note by its
 * file and selector, and an action that creates a note by the window and the words typed.
 */
export function resolveChoice(
  result: SearchHit | undefined,
  action: string,
  typed = '',
): ResultChoice | null {
  if (!result || !result.actionIds.includes(action)) return null
  if (result.creationIds?.includes(action)) {
    return {
      command: action,
      over: null,
      input: typed.trim(),
    }
  }
  if (result.type === null) {
    if (action !== ROW.openDocument) return null
    const place = { path: '', file: result.spot.path, title: result.spot.title }
    return {
      command: action,
      over: result.selector ? { ...place, selector: result.selector } : place,
      input: '',
    }
  }
  return resolveNoteChoice(result, action)
}
