import { vaultsService } from '@/shared/api/clients'
import type { Vault } from '../lib/vault'

/** The vaults the installation holds. */
export const listVaults = async (): Promise<readonly Vault[]> =>
  (await vaultsService.listVaults({})).vaults.map((one) => ({
    id: one.id,
    name: one.name,
    path: one.path,
  }))
