/**
 * A path as the vault files a note under, taken apart.
 *
 * A path is separated by `/` whatever this machine writes its own paths with:
 * it is the vault's, not the disk's, and the same string on every platform.
 */

/** What a file is filed as, which is the last segment of the path. */
export const getFileName = (path: string): string => path.split('/').pop() ?? path

/**
 * The file's name without its extension, which is what a link writes and what a
 * name resolves by. A link therefore remains valid when the file is moved.
 */
export const getFileStem = (path: string): string => {
  const file = getFileName(path)
  const dot = file.lastIndexOf('.')
  return dot > 0 ? file.slice(0, dot) : file
}

/** A note or file that moved, with its previous and new path. */
export interface PathRename {
  readonly from: string
  readonly to: string
}

/** Where a file went, and nothing where none of these moved it. */
export const getRenamedPath = (renames: readonly PathRename[], path: string): string =>
  renames.find((one) => one.from === path)?.to ?? ''
