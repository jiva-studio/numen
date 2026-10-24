export { openNotes } from './notes'
export {
  convertErrorCode,
  getMarkOf,
  parseFileConflict,
  type FileConflict,
  ERROR_MESSAGES,
  STALE_CONFLICT,
} from './words'
export type {
  ConflictWords,
  LinkAddress,
  NoteContents,
  Notes as NoteCore,
  NoteResult as NoteReadResult,
  OpenNote,
  OpenNotesOptions,
  WriteResult as NoteWriteResult,
  NoteBaseline,
  NoteErrorCode,
  State as NoteState,
} from './types'
