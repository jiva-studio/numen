import type { VaultsClient } from '../api/vaults'

type SyncOutcome =
  | { readonly kind: 'synced' }
  | { readonly kind: 'already' }
  | { readonly kind: 'choose'; readonly remoteId: string; readonly name: string }

type SyncClient = Pick<VaultsClient, 'listVaults' | 'listRemoteVaults' | 'connectVault'>

/**
 * Starts the sync of a vault. A vault that is synced already needs nothing. A vault with the name of a vault
 * of the identity that no device holds yet waits for the person to choose between the two copies.
 */
export async function startSync(client: SyncClient, vaultId: string): Promise<SyncOutcome> {
  const vault = (await client.listVaults()).vaults.find((one) => one.id === vaultId)
  if (vault?.state === 'connected') return { kind: 'already' }
  const sameName = (await client.listRemoteVaults()).vaults.find(
    (one) => !one.isConnected && one.name === vault?.name,
  )
  if (sameName !== undefined) {
    return { kind: 'choose', remoteId: sameName.remoteId, name: sameName.name }
  }
  await client.connectVault({ vaultId, mode: 'new' })
  return { kind: 'synced' }
}

/** Ties a vault to the vault of the identity the person chose, or sends the vault as a new one. */
export async function finishSync(
  client: Pick<VaultsClient, 'connectVault'>,
  vaultId: string,
  remoteId: string,
): Promise<void> {
  await client.connectVault(
    remoteId === '' ? { vaultId, mode: 'new' } : { vaultId, mode: 'existing', remoteId },
  )
}
