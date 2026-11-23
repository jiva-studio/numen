/** What a watch of the folder tree of a vault saw during a move: how many files each folder held at each sample. */

export interface FolderSample {
  readonly counts: Readonly<Record<string, number>>
}

/** The samples in which both folders held files at once. */
export const findMixedSamples = (samples: readonly FolderSample[], first: string, second: string): FolderSample[] => {
  throw new Error(`not implemented: ${samples.length} samples of "${first}" and "${second}"`)
}

/** The count of files of each of the folders in a listing of paths of files, by their top folder. */
export const countFilesByFolder = (paths: readonly string[], folders: readonly string[]): Record<string, number> => {
  throw new Error(`not implemented: ${paths.length} paths in ${folders.length} folders`)
}
