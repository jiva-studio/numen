/**
 * The vaults of the installation together with the one this window shows.
 */
import { windowService } from '@/shared/api/clients'
import { WINDOW } from '@/shared/api/window'
import type { VaultList } from '../lib/vault'
import { vaults } from './vaults'

/** Every vault the installation holds, and which of them this window shows. */
export async function getVaultList(): Promise<VaultList> {
  const [listed, current] = await Promise.all([
    vaults.list(),
    windowService.getCurrentVault({ window: WINDOW }),
  ])
  return { vaults: listed, showing: current.vault }
}
