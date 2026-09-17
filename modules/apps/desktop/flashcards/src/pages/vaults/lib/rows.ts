/**
 * A vault's count as the welcome screen draws it.
 *
 * The screen takes rows and a number to a row; what a vault is, and what it
 * owes, is this window's to say and not the component's.
 */
import type { VaultRow } from '@numen/ui'
import type { VaultCardsDue } from '@/entities/vault'
import type { VaultsWords } from '../words'

/** Each vault as a row, with what is worth saying under its name. */
export function getVaultRows(
  vaults: readonly VaultCardsDue[],
  words: VaultsWords,
): readonly VaultRow[] {
  return vaults.map((one) => {
    const said = one.reading ? words.reading : one.unread
    return {
      id: one.vault,
      name: one.name,
      path: one.path,
      isWorking: !one.counted,
      ...(said ? { detail: said } : {}),
    }
  })
}

/**
 * How many cards each vault has waiting, by its identity.
 *
 * A vault still being read, or one that could not be read, is absent: what
 * stands in its row is why, and not a number. A vault counted to nothing is
 * present and owes nothing.
 */
export function getDueByVault(
  vaults: readonly VaultCardsDue[],
): ReadonlyMap<string, number | null> {
  return new Map(
    vaults
      .filter((one) => !one.unread && !one.reading)
      .map((one) => [one.vault, one.counted ? one.due + one.new : null]),
  )
}
