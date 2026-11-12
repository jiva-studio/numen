/**
 * The folder a row stands for in the files tree.
 */
import { getFolderPath, ROOT } from './useFileTree'
import type { FileTree } from '../types'

export const getFolderFor = (list: FileTree, path: string | null): string => {
  if (path === null) return ROOT
  const entry = list.queries.getEntryAt(path)
  return entry?.isFolder ? entry.path : getFolderPath(path)
}
