/** A note of the vault: what it holds, who it is joined to, and the tab it is open in. */
export { getNoteIcon } from './lib/icons'
export type {
  Link,
  Neighbourhood,
  NewNote,
  NoteEdit,
  NoteHeading,
  NoteResult,
  RemovedNote,
  RemoveResult,
  RenamedNote,
  RenameResult,
  Role,
  NoteRelation,
  PairLink,
  WriteResult,
} from './lib/note'
export { openNotes } from './model/notes'
export type { Notes, OpenNote } from './lib/noteTypes'
export { getMarkOf } from './lib/tabState'
export type { NoteBaseline, State } from './lib/tabState'
export { CREATABLE, UNTITLED, createNoteWriter } from './model/maker'
export type { NoteCreator, NoteMaker, NoteRef } from './model/maker'
export { convertErrorCode } from './model/queue'
export { ERROR_MESSAGES, WORDS } from './words'
