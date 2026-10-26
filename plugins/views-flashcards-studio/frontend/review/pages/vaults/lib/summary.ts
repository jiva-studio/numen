/**
 * What the summary beside a vault says: how many cards it has waiting, or why
 * there is no number.
 */
import type { VaultCardsDue } from '../../../entities/vault'
import type { VaultsWords } from '../words'

/**
 * A count, which is nothing while it is on its way, or a sentence where no
 * count will come: the vault is being read, or could not be.
 */
export type VaultSummary =
  | { readonly kind: 'due'; readonly due: number | null }
  | { readonly kind: 'detail'; readonly text: string }

/** The summary of a vault, and a count on its way while the vault has not been named by one. */
export function getVaultSummary(vault: VaultCardsDue | null, words: VaultsWords): VaultSummary {
  if (vault?.isIndexing) return { kind: 'detail', text: words.indexing }
  if (vault?.unread) return { kind: 'detail', text: vault.unread }
  if (!vault?.isCounted) return { kind: 'due', due: null }
  return { kind: 'due', due: vault.due + vault.new }
}
