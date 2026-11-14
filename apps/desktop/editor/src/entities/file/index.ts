/** The files of the vault: which of them a file is, and what making one or moving one returns. */
export type { Entry, FileKind, NoteType } from './types'
export { countNotes } from './lib/countNotes'
export { createFileApi, type FileApi } from './api/files'
