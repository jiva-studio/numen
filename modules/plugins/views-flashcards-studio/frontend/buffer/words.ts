import type { NoteErrorCode, State } from './types'

export const ERROR_MESSAGES: Record<NoteErrorCode, string> = {
  notANote: 'this file is not a note',
  notText: 'this file is not text',
  tooLarge: 'this note is longer than the editor holds',
  bodyUnwritable: 'a note begins below its frontmatter, and this text begins with one',
  unreadable: 'the frontmatter of this note cannot be read',
  unreachable: 'the vault could not be reached, so this note was not written',
}

export const STALE_CONFLICT = {
  says: 'this note changed on disk, and saving stopped',
  keep: 'keep mine',
  take: "take the file's",
}

export type FileConflict = 'gone' | 'stale' | null

export function parseFileConflict(state: string): FileConflict {
  return state === 'gone' || state === 'stale' ? state : null
}

const TAB_CODES: ReadonlySet<string> = new Set<NoteErrorCode>([
  'tooLarge',
  'bodyUnwritable',
  'notANote',
  'notText',
  'unreadable',
  'unreachable',
])

const TAB_CODE_OF: Readonly<Record<string, NoteErrorCode>> = {
  deckTooLarge: 'tooLarge',
  notAStencil: 'notANote',
  notADeck: 'notANote',
  notAPreset: 'notANote',
  missing: 'unreadable',
  occupied: 'unreadable',
  unnameable: 'unreadable',
}

export const convertErrorCode = (from: string): NoteErrorCode =>
  TAB_CODE_OF[from] ?? (TAB_CODES.has(from) ? (from as NoteErrorCode) : 'unreadable')

const MARKS: Record<string, string | undefined> = {
  gone: 'gone',
  stale: 'stale',
  unsaved: 'unsaved',
  saving: 'unsaved',
  loading: undefined,
  clean: undefined,
  error: undefined,
}

export function getMarkOf(state: State): string | undefined {
  return MARKS[state]
}
