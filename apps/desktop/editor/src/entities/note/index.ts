/** A note of the vault: its contents, who it is joined to, and the tab it is open in. */
export { getNoteIcon, registerTabKindIcon } from './model/icons'
export type { NoteEdit, NoteResult, WriteResult } from './lib/note'
export { convertErrorCode, ERROR_MESSAGES, openNotes } from './model/buffer'
export { createNoteWriter } from './model/maker'
export type { NoteCreator } from './model/maker'
export { WORDS } from './words'
export {
  getInitialOpenPath,
  listRelatedNotes,
  listHeadings,
  readNote,
  renameNote,
  resolveLinks,
  resolveNameLink,
  watchEdits,
  writeNote,
} from './api/notes'
export { relations } from './api/relations'
