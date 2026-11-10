import type { MenuItem } from '@numen/ui'
import { DEMO_COMMAND } from './commands'
import { ROOT, type Entry } from './entries'

/** The id of the menu row that opens the rows under it. */
export const NEW_SUBMENU = 'submenu:new'
export const RELATION_SUBMENU = 'submenu:relation'

type Row = Omit<MenuItem, 'group'>

const row = (id: string, text: string, children?: readonly Row[]): Row => ({
  id,
  text,
  ...(children ? { children } : {}),
})

const OPEN = row(DEMO_COMMAND.openNote, 'Open note')
const SHOW_IN_PLEX = row(DEMO_COMMAND.showNoteInPlex, 'Show note in plex')
const REVEAL = row(DEMO_COMMAND.revealNote, 'Show note in file tree')
const ASK = row(DEMO_COMMAND.askAgent, 'Ask agent about note')
const COPY = row(DEMO_COMMAND.copyNotePath, 'Copy note path')
const RENAME_ENTRY = row(DEMO_COMMAND.renameTreeEntry, 'Rename')
const CHILD = row(DEMO_COMMAND.createChildNote, 'Create child note')
const PARENT = row(DEMO_COMMAND.createParentNote, 'Create parent note')
const JUMP = row(DEMO_COMMAND.createJumpNote, 'Create jump note')
const RENAME_NOTE = row(DEMO_COMMAND.renameNote, 'Rename note')
const TRASH = row(DEMO_COMMAND.removeFile, 'Move note to trash')

const NEW = row(NEW_SUBMENU, 'New', [
  row(DEMO_COMMAND.createNote, 'Create note'),
  row(DEMO_COMMAND.createFolder, 'Create folder'),
])
const RELATION = row(RELATION_SUBMENU, 'Relation', [CHILD, PARENT, JUMP])

function toItems(sections: readonly (readonly Row[])[]): readonly MenuItem[] {
  return sections.flatMap((rows, index) => rows.map((one) => ({ ...one, group: String(index) })))
}

/** The rows a command from a menu is run over: the row it was opened on, and the rows selected with it. */
export const getRowsOver = (path: string, selected: readonly string[]): readonly string[] =>
  selected.includes(path) ? selected : [path]

/** The menu on a node of the plex. */
export const PLEX_MENU: readonly MenuItem[] = toItems([
  [OPEN],
  [SHOW_IN_PLEX, REVEAL, ASK],
  [COPY],
  [CHILD, PARENT, JUMP, RENAME_NOTE],
  [TRASH],
])

/** What the menu on a row of the file tree is over. */
export interface FileMenuTarget {
  readonly path: string
  readonly others: readonly string[]
  readonly entry: Entry | undefined
  readonly isDemoNote: (path: string) => boolean
}

function getNoteRows(target: FileMenuTarget): readonly (readonly Row[])[] {
  const isDemoNote = target.isDemoNote(target.path)
  return [
    [OPEN, ...(isDemoNote ? [SHOW_IN_PLEX] : [])],
    [RENAME_ENTRY, COPY],
    [RELATION],
    [ASK],
    ...(isDemoNote ? [[TRASH]] : []),
  ]
}

/** The menu on a row of the file tree, or on the empty space under it. */
export function getFileMenu(target: FileMenuTarget): readonly MenuItem[] {
  const { path, others, entry } = target
  if (path === ROOT) return toItems([[NEW]])
  if (others.length > 0) {
    const remove = row(DEMO_COMMAND.removeFile, 'Remove')
    return [path, ...others].every(target.isDemoNote) ? toItems([[remove]]) : []
  }
  if (!entry || entry.isFolder) return toItems([[NEW, RENAME_ENTRY, COPY]])
  if (!entry.isNote) return toItems([[row(DEMO_COMMAND.openNote, 'Open the file')], [RENAME_ENTRY, COPY]])
  return toItems(getNoteRows(target))
}
