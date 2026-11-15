/**
 * The vaults of the installation together with the one this window shows.
 */
import type { VaultList, Vaults } from '../lib/vault'

/** What the list of the vaults is read through. */
export interface VaultListApi {
  readonly vaults: Pick<Vaults, 'list'>
  readonly getShownVault: () => Promise<string>
}

/** Reads every vault the installation holds, and which of them this window shows. */
export const createVaultListReader = (api: VaultListApi) => async (): Promise<VaultList> => {
  const [listed, showing] = await Promise.all([api.vaults.list(), api.getShownVault()])
  return { vaults: listed, showing }
}
