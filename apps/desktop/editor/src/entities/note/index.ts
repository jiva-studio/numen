/** A note of the vault: its contents, who it is joined to, and the tab it is open in. */
export { getNoteIcon, registerTabKindIcon } from './model/icons'
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
export type { NoteBaseline, NoteState as State, OpenNote } from './model/buffer'
export { createNoteWriter } from './model/maker'
export type { NoteCreator, NoteRef } from './model/maker'
export { WORDS } from './words'
export {
  getInitialOpenPath,
  getNeighbourhood,
  listHeadings,
  readNote,
  renameNote,
  resolveLinks,
  watchEdits,
  writeNote,
} from './api/notes'
export { relations } from './api/relations'
