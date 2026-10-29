/**
 * How much of the vault the index has covered, and whether it is still working.
 *
 * A search by meaning queries the vectors, and a vault with none returns nothing
 * however it is queried. The corner and the palette both report this, from here.
 */

/** The vault's state with respect to being indexed for meaning. */
export interface IndexCoverage {
  /** Spans of text the index holds. */
  readonly chunks: number
  /** How many of those spans carry a vector. */
  readonly embedded: number
  /** Whether anything is going to turn the spans into vectors. */
  readonly isEmbedding: boolean
}

/** Whether the vault is searched by its words alone. */
export const isTextSearchOnly = (readItem: IndexCoverage): boolean =>
  readItem.chunks > 0 && !readItem.isEmbedding
