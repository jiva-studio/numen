import type { WindowClients } from '@/shared/api/clients'
import type { Vault } from '../lib/vault'

/** Lists through `vaultsService` the vaults the installation holds. */
export const createVaultLister =
  (vaultsService: WindowClients['vaultsService']) => async (): Promise<readonly Vault[]> =>
    (await vaultsService.listVaults({})).vaults.map((one) => ({
      id: one.id,
      name: one.name,
      path: one.path,
    }))

export type VaultLister = ReturnType<typeof createVaultLister>
