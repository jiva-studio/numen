import type { NoteErrorCode, State } from './types'

export const STALE_CONFLICT = {
  says: 'saving paused: the note changed on disk',
  keep: 'keep my version',
  take: 'use disk version',
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
  notACardModel: 'notANote',
  notADeck: 'notANote',
  notAPreset: 'notANote',
  notFound: 'unreadable',
  alreadyExists: 'unreadable',
  invalidName: 'unreadable',
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
