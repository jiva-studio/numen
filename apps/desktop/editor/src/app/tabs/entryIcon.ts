/**
 * The icon one entry of the files tree is drawn as. A folder says whether what
 * it holds is drawn; a note is drawn by which kind of note it is, and every
 * other file by the icon of the reader that opens it.
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
