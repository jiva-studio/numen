/**
 * Creation of notes, folders, decks, stencils, and presets in the files tree.
 */
import { folderOf, type FileTree, ROOT } from './listing'
import { WORDS as words } from './words'
import type { FileMaker, FilesTabDeps } from './types'

/** The folder a row stands in or sits inside, or the root when none. */
export const folderFor = (list: FileTree, path: string | null): string => {
  if (path === null) return ROOT
  const entry = list.entryAt(path)
  return entry?.folder ? entry.path : folderOf(path)
}

/** Creates a folder under a free name and returns the created path if successful. */
export const createFolder = async (
  list: FileTree,
  deps: FilesTabDeps,
  path: string | null,
): Promise<string | null> => {
  const into = folderFor(list, path)
  const name = list.freeIn(into, words.folder)
  const made = into === ROOT ? name : `${into}/${name}`
  await deps.makes(made)
  await list.opens(into)
  return list.entryAt(made) ? made : null
}

/** Creates a note in the folder and returns the created path. */
export const createNote = async (
  list: FileTree,
  deps: FilesTabDeps,
  path: string | null,
): Promise<string | null> => {
  const into = folderFor(list, path)
  const made = await deps.writes(into)
  if (!made) return null
  await list.opens(into)
  return made
}

/** Creates a custom file type in the folder. */
export const createOne = async (
  list: FileTree,
  path: string | null,
  createEntry: FileMaker,
  name: string,
): Promise<string | null> => {
  const into = folderFor(list, path)
  const made = await createEntry(into, name)
  if (!made) return null
  await list.opens(into)
  return made
}
