/**
 * How much of the vault the index has covered, and whether it is still working.
 *
 * A search by meaning is asked of the vectors, and a vault holding none
 * answers nothing however it is asked. The corner and the palette both say so,
 * and both say it from here.
 */

/** What the vault says about itself about having been read for meaning. */
export interface IndexCoverage {
  /** Spans of text the index holds. */
  readonly chunks: number
  /** How many of those spans carry a vector. */
  readonly embedded: number
  /** Whether anything is going to turn the spans into vectors. */
  readonly isEmbedding: boolean
}

/** Whether the vault is searched by its words alone. */
export const wordsOnly = (read: IndexCoverage): boolean => read.chunks > 0 && !read.isEmbedding
