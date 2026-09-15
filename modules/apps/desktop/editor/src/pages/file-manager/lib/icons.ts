/**
 * The icon one entry of the files tree is drawn as. A folder says whether what
 * it holds is drawn; a note is drawn by which kind of note it is, and every
 * other file by the source the vault holds it as.
 */
import { File, Folder, FolderOpen, type LucideIcon } from '@lucide/vue'
import type { Entry } from '@/entities/file'
import { iconOfSource } from '@/entities/file'
import { iconOfNote } from '@/entities/note'

export const iconOfEntry = (entry: Entry | null | undefined, open: boolean): LucideIcon => {
  if (!entry) return File
  if (entry.folder) return open ? FolderOpen : Folder
  if (entry.kind === 'note') return iconOfNote(entry.type)
  return iconOfSource(entry.kind) ?? File
}
