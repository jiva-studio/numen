/** A note of the vault: what it holds, who it is joined to, and the tab it is open in. */
export { getNoteIcon, registerNoteIcon } from './model/icons'
export type {
  Neighbourhood,
  NewNote,
  NoteEdit,
  NoteHeading,
  NoteResult,
  RemovedNote,
  RemoveResult,
  RenamedNote,
  RenameResult,
  NoteRelation,
  WriteResult,
} from './lib/note'
export { convertErrorCode, ERROR_MESSAGES, getMarkOf, openNotes } from './model/buffer'
export type { NoteBaseline, NoteCore as Notes, NoteState as State, OpenNote } from './model/buffer'
export { createNoteWriter } from './model/maker'
export type { NoteCreator, NoteMaker, NoteRef } from './model/maker'
export { WORDS } from './words'
