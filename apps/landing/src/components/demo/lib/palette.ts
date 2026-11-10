import { keyChord, type Keystroke, type TextRange } from '@numen/ui'
import type { PaletteGroup, PaletteItem } from '@numen/workbench'
import { DEMO_COMMAND } from './commands'
import type { VaultNote } from './notes'

export type NamingRole = 'child' | 'parent' | 'jump' | 'title'

export const NAMING_LABELS: Readonly<Record<NamingRole, string>> = {
  child: 'Create child note',
  parent: 'Create parent note',
  jump: 'Create jump note',
  title: 'Rename note',
}

/** The command each naming step belongs to. */
export const NAMING_COMMAND: Readonly<Record<NamingRole, string>> = {
  child: DEMO_COMMAND.createChildNote,
  parent: DEMO_COMMAND.createParentNote,
  jump: DEMO_COMMAND.createJumpNote,
  title: DEMO_COMMAND.renameNote,
}

export const KEYBOARD = 'Macintosh'

/** The runs of `title` that `word` matches, ignoring case. */
export function getMarks(title: string, word: string): TextRange[] {
  const query = word.trim().toLowerCase()
  if (!query) return []
  const spans: TextRange[] = []
  const lower = title.toLowerCase()
  for (let pos = lower.indexOf(query); pos >= 0; pos = lower.indexOf(query, pos + 1)) {
    spans.push({ start: pos, end: pos + query.length })
  }
  return spans
}

const TRAVEL_ACTION = { id: 'travel', text: 'Show in plex' }
const READ_ACTION = { id: 'read', text: 'Open note' }
const READ_DOC_ACTION = { id: 'readDocument', text: 'Open the document here' }

/** Passages of the vault the search shows whatever it is asked; those that do not match the query are left out. */
const PASSAGES = [
  {
    id: 'reading/Boltzmann 1877.pdf',
    title: 'Boltzmann 1877',
    detail: 'the entropy of a state is the logarithm of the number of arrangements it could have been made of',
    actions: [READ_DOC_ACTION],
  },
  {
    id: 'physics/The second law.md',
    title: 'The second law',
    detail: 'entropy never falls in an isolated system, which is the whole law stated in one line',
    actions: [READ_ACTION, TRAVEL_ACTION],
  },
]

/** The names and passages that match a search. */
export function getSearchGroups(notes: readonly VaultNote[], typed: string): readonly PaletteGroup[] {
  const query = typed.trim().toLowerCase() || 'entrop'
  const names = notes.filter((one) => one.title.toLowerCase().includes(query))
  const passages = PASSAGES.filter(
    (one) => one.title.toLowerCase().includes(query) || one.detail.toLowerCase().includes(query),
  ).map((one) => ({ ...one, detailAt: getMarks(one.detail, query) }))
  const groups: PaletteGroup[] = []
  if (names.length > 0) {
    const items = names.map((one) => ({
      id: one.path,
      title: one.title,
      at: getMarks(one.title, query),
      actions: [TRAVEL_ACTION, READ_ACTION],
    }))
    groups.push({ id: 'names', title: 'Names', items })
  }
  if (passages.length > 0) groups.push({ id: 'text', title: 'Text', items: passages })
  return groups
}

interface CommandRow {
  readonly id: string
  readonly title: string
  readonly keys?: Keystroke
  readonly also?: { readonly id: string; readonly text: string }
}

const COMMAND_GROUPS: readonly { readonly id: string; readonly title: string; readonly rows: readonly CommandRow[] }[] = [
  {
    id: 'note',
    title: 'Note',
    rows: [
      {
        id: DEMO_COMMAND.openNote,
        title: 'Open note',
        also: { id: DEMO_COMMAND.openNoteBeside, text: 'Open note beside' },
      },
      { id: DEMO_COMMAND.showNoteInPlex, title: 'Show note in plex', keys: keyChord('p', KEYBOARD, true) },
      { id: DEMO_COMMAND.createChildNote, title: 'Create child note', keys: keyChord('c', KEYBOARD, true) },
      { id: DEMO_COMMAND.createParentNote, title: 'Create parent note' },
      { id: DEMO_COMMAND.createJumpNote, title: 'Create jump note' },
      { id: DEMO_COMMAND.renameNote, title: 'Rename note' },
      { id: DEMO_COMMAND.askAgent, title: 'Ask agent about note' },
      { id: DEMO_COMMAND.copyNotePath, title: 'Copy note path' },
      { id: DEMO_COMMAND.removeFile, title: 'Move note to trash' },
    ],
  },
  {
    id: 'window',
    title: 'General',
    rows: [
      { id: DEMO_COMMAND.createNote, title: 'Create note', keys: keyChord('n', KEYBOARD) },
      { id: DEMO_COMMAND.openPlex, title: 'Open plex graph' },
      { id: DEMO_COMMAND.openFiles, title: 'Open file tree' },
      { id: DEMO_COMMAND.openAgent, title: 'Open agent chat', keys: keyChord('a', KEYBOARD, true) },
    ],
  },
]

function getCommandItem(one: CommandRow, typed: string): PaletteItem {
  const at = getMarks(one.title, typed)
  return {
    id: one.id,
    title: one.title,
    ...(at.length > 0 ? { at } : {}),
    ...(one.keys ? { keys: one.keys } : {}),
    actions: [{ id: one.id, text: one.title }, ...(one.also ? [one.also] : [])],
  }
}

/** The commands whose title holds what is typed. */
export function getCommandGroups(typed: string): readonly PaletteGroup[] {
  const query = typed.trim().toLowerCase()
  return COMMAND_GROUPS.map((group) => ({
    id: group.id,
    title: group.title,
    items: group.rows
      .filter((one) => !query || one.title.toLowerCase().includes(query))
      .map((one) => getCommandItem(one, query)),
  })).filter((group) => group.items.length > 0)
}

/** The one row that takes what is typed as the name. */
export function getNamingGroups(typed: string): readonly PaletteGroup[] {
  const name = typed.trim()
  const items = name
    ? [{ id: 'name', title: `Name it “${name}”`, actions: [{ id: 'name', text: 'Name it' }] }]
    : []
  return [{ id: 'naming', title: 'Name', items, silence: 'Name' }]
}
