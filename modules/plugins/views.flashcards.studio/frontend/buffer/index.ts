export { openNotes } from './notes'
export { convertErrorCode } from './queue'
export { getMarkOf } from './tabState'
export { parseFileConflict, type FileConflict } from './conflict'
export { ERROR_MESSAGES, STALE_CONFLICT } from './words'
export type {
  ConflictWords,
  LinkAddress,
  NoteContents,
  Notes as NoteCore,
  NoteResult as NoteReadResult,
  OpenNote,
  OpenNotesOptions,
  WriteResult as NoteWriteResult,
} from './types'
export type { NoteBaseline, NoteErrorCode, State as NoteState } from './tabState'
