/**
 * Creation of notes, folders and other files in the files tree.
 */
import { getFolderPath, ROOT } from './useFileTree'
import { WORDS as words } from '../words'
import type { FilesTabDeps, FileTree } from '../types'

export const getFolderFor = (list: FileTree, path: string | null): string => {
  if (path === null) return ROOT
  const entry = list.getEntryAt(path)
  return entry?.isFolder ? entry.path : getFolderPath(path)
}

/** Creates a folder under a unique name and returns the created path if successful. */
export const createFolder = async (
  list: FileTree,
  deps: FilesTabDeps,
  path: string | null,
): Promise<string | null> => {
  const into = getFolderFor(list, path)
  const name = list.getUniqueNameInFolder(into, words.folder)
  const createdItem = into === ROOT ? name : `${into}/${name}`
  await deps.createFolder(createdItem)
  await list.openFolder(into)
  return list.getEntryAt(createdItem) ? createdItem : null
}

/** Creates a note in the folder and returns the created path. */
export const createNote = async (
  list: FileTree,
  deps: FilesTabDeps,
  path: string | null,
): Promise<string | null> => {
  const into = getFolderFor(list, path)
  const createdItem = await deps.createNote(into)
  if (!createdItem) return null
  await list.openFolder(into)
  return createdItem
}
