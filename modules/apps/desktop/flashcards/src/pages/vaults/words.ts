/** What the vaults screen says, in the words it says them by. */
export const VAULTS_WORDS = {
  /** What the screen is called above the list. */
  heading: 'Vaults',
  /** A vault whose count has not arrived because it is still being read. */
  reading: 'Reading the vault',
  /** Said once while the installation has not yet named the vaults it holds. */
  counting: 'Reading the vaults',
} as const

export type VaultsWords = typeof VAULTS_WORDS
