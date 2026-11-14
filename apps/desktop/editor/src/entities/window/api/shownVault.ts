import type { WindowClients } from '@/shared/api/clients'
import { WINDOW } from './window'

/** The vault this window shows, over `windowService`. */
export const createShownVaultApi = (windowService: WindowClients['windowService']) => ({
  /** The id of the vault this window shows. */
  getShownVault: async (): Promise<string> =>
    (await windowService.getCurrentVault({ window: WINDOW })).vault,
})

export type ShownVaultApi = ReturnType<typeof createShownVaultApi>
