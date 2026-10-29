/** The text the review shows while a vault's count is pending. */
// Stryker disable StringLiteral: interface copy
export const REVIEW_WORDS = {
  /** A vault whose count has not arrived because it is still being read into the index. */
  indexing: 'Indexing the vault',
  /** A vault whose count has not arrived yet. */
  counting: 'Counting the cards',
} as const
