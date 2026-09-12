/**
 * A path as the vault files a note under, taken apart.
 *
 * A path is separated by `/` whatever this machine writes its own paths with:
 * it is the vault's, not the disk's, and the same string on every platform.
 */

/** What a file is filed as, which is the last segment of the path. */
export const fileOf = (path: string): string => path.split('/').pop() ?? path

/**
 * The file's name without its ending, which is what a link writes and what a
 * name resolves by. A link therefore stands where the file is moved.
 */
export const nameOf = (path: string): string => {
  const file = fileOf(path)
  const dot = file.lastIndexOf('.')
  return dot > 0 ? file.slice(0, dot) : file
}

/** A note or file that is no longer where it was, and where it now is. */
export interface PathRename {
  readonly from: string
  readonly to: string
}

/** Where a file went, and nothing where none of these moved it. */
export const getRenamedPath = (renamed: readonly PathRename[], path: string): string =>
  renamed.find((one) => one.from === path)?.to ?? ''
