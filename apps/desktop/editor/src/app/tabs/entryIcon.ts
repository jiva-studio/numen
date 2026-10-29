/**
 * The icon for one entry of the files tree. A folder's icon reflects whether
 * its contents are shown; a note's icon depends on its note type, and every
 * other file's icon is that of the reader that opens it.
 */
import type { Component } from 'vue'
import { File, Folder, FolderOpen } from '@lucide/vue'
import type { Entry } from '@/entities/file'
import { getNoteIcon } from '@/entities/note'

export const getEntryIcon = (
  entry: Entry | null | undefined,
  isOpen: boolean,
  findFileIcon?: (path: string) => Component | undefined,
): Component => {
  if (!entry) return File
  if (entry.isFolder) return isOpen ? FolderOpen : Folder
  if (entry.isNote) return getNoteIcon(entry.type)
  return findFileIcon?.(entry.path) ?? File
}
