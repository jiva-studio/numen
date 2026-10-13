/** A note of the vault: what it holds, who it is joined to, and the tab it is open in. */
export { getNoteIcon } from './lib/icons'
export { areLinkTargetsEqual, extractLinkTargets, parseLinkTarget } from './lib/links'
export type { LinkTarget } from './lib/links'
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
export type { Notes, OpenNote, OpenNotesOptions } from './lib/noteTypes'
export { getMarkOf } from './lib/tabState'
export type { NoteBaseline, State } from './lib/tabState'
export { WORDS } from './words'
