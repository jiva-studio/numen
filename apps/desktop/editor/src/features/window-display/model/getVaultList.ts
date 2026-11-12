/**
 * The vaults of the installation together with the one this window shows.
 */
import { getShownVault } from '@/entities/window'
import { vaults, type VaultList } from '@/entities/vault'

/** Every vault the installation holds, and which of them this window shows. */
export async function getVaultList(): Promise<VaultList> {
  const [listed, showing] = await Promise.all([vaults.list(), getShownVault()])
  return { vaults: listed, showing }
}
