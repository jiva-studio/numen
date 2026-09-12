/**
 * Creation of notes, folders, decks, stencils, and presets in the files tree.
 */
import { getFolderPath, ROOT } from './useFileTree'
import { WORDS as words } from '../words'
import type { FileMaker, FilesTabDeps, FileTree } from '../types'

/** The folder a row stands in or sits inside, or the root when none. */
export const getFolderFor = (list: FileTree, path: string | null): string => {
  if (path === null) return ROOT
  const entry = list.getEntryAt(path)
  return entry?.folder ? entry.path : getFolderPath(path)
}

/** Creates a folder under a unique name and returns the created path if successful. */
export const createFolder = async (
  list: FileTree,
  deps: FilesTabDeps,
  path: string | null,
): Promise<string | null> => {
  const into = getFolderFor(list, path)
  const name = list.getUniqueNameInFolder(into, words.folder)
  const made = into === ROOT ? name : `${into}/${name}`
  await deps.createFolder(made)
  await list.openFolder(into)
  return list.getEntryAt(made) ? made : null
}

/** Creates a note in the folder and returns the created path. */
export const createNote = async (
  list: FileTree,
  deps: FilesTabDeps,
  path: string | null,
): Promise<string | null> => {
  const into = getFolderFor(list, path)
  const made = await deps.createNote(into)
  if (!made) return null
  await list.openFolder(into)
  return made
}

/** Creates a custom file type in the folder. */
export const createOne = async (
  list: FileTree,
  path: string | null,
  createEntry: FileMaker,
  name: string,
): Promise<string | null> => {
  const into = getFolderFor(list, path)
  const made = await createEntry(into, name)
  if (!made) return null
  await list.openFolder(into)
  return made
}
