/**
 * The icon one entry of the files tree is drawn as. A folder says whether what
 * it holds is drawn; a note is drawn by which kind of note it is, and every
 * other file by the source the vault holds it as.
 */
import { File, Folder, FolderOpen, type LucideIcon } from '@lucide/vue'
import type { Entry } from '@/entities/file'
import { findSourceIcon } from '@/entities/file'
import { getNoteIcon } from '@/entities/note'

export const getEntryIcon = (entry: Entry | null | undefined, open: boolean): LucideIcon => {
  if (!entry) return File
  if (entry.isFolder) return open ? FolderOpen : Folder
  if (entry.kind === 'note') return getNoteIcon(entry.type)
  return findSourceIcon(entry.kind) ?? File
}
