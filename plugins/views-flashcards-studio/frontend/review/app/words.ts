/** What the review says while a vault's count is on its way, in the words it says them by. */
// Stryker disable StringLiteral: interface copy
export const REVIEW_WORDS = {
  /** A vault whose count has not arrived because it is still being read into the index. */
  indexing: 'Indexing the vault',
  /** A vault whose count has not arrived yet. */
  counting: 'Counting the cards',
} as const
