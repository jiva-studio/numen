/**
 * A path as the vault files a note under, taken apart.
 *
 * A path is separated by `/` whatever this machine writes its own paths with:
 * it is the vault's, not the disk's, and the same string on every platform.
 */

/** What a file is filed as, which is the last segment of the path. */
export const getFileName = (path: string): string => path.split('/').pop() ?? path

/** A note or file that moved, with its previous and new path. */
export interface PathRename {
  readonly from: string
  readonly to: string
}

/** Where a file went, and nothing where none of these moved it. */
export const getRenamedPath = (renames: readonly PathRename[], path: string): string =>
  renames.find((one) => one.from === path)?.to ?? ''

/** The word, or the word and the lowest count from 2 up, that none of the names carries. */
export const generateUniqueName = (names: readonly string[], word: string): string => {
  const existingNames = new Set(names)
  if (!existingNames.has(word)) return word
  for (let count = 2; ; count++) {
    const name = `${word} ${count}`
    if (!existingNames.has(name)) return name
  }
}
