/** What a file is filed as, which is the last segment of the path. */
export const getFileName = (path: string): string => path.split('/').pop() ?? path

/** A note or file that is no longer where it was, and where it now is. */
export interface PathRename {
  readonly from: string
  readonly to: string
}

/** Where a file went, and nothing where none of these moved it. */
export const getRenamedPath = (renames: readonly PathRename[], path: string): string =>
  renames.find((one) => one.from === path)?.to ?? ''
