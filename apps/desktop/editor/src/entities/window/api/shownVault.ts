import type { WindowClients } from '@/shared/api/clients'
import { WINDOW } from './window'

/** Asks the core which vault it opened this window on. */
export const createShownVaultApi = (windowService: WindowClients['windowService']) => ({
  /** The id of the vault this window shows. */
  getShownVault: async (): Promise<string> =>
    (await windowService.getCurrentVault({ windowId: WINDOW })).vaultId,
})
