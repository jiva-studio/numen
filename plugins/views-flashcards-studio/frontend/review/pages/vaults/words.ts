/** What the summary of a vault says, in the words it says them by. */
// Stryker disable StringLiteral: interface copy
export const VAULTS_WORDS = {
  /** A vault whose count has not arrived because it is still being read. */
  indexing: 'Indexing the vault',
} as const

export type VaultsWords = typeof VAULTS_WORDS
