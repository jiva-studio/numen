/** The files of the vault: which of them a file is, and what making one or moving one returns. */
export type { Entry, FileKind, NoteType } from './types'
export {
  createFolder,
  emptyTrash,
  getFileKinds,
  listFiles,
  moveFile,
  removeFile,
} from './api/files'
