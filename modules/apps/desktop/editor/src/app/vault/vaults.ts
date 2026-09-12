/**
 * Vault management domain methods for the window core and vaults interface.
 */
import { WINDOW } from './clients'
import { vaultsService, windowService } from '@/shared/clients'
import { mapVault, mapVaultResult, turnedDown } from './words'
import type { CommandsDeps } from '@/features/command-palette'
import type { Vaults } from '@/shared/vaults'

/** The vaults this installation holds, in the shape the window asks about them. */
export const vaults: Vaults = {
  list: async () => {
    const [answer, shown] = await Promise.all([
      vaultsService.listVaults({}),
      windowService.getShownVault({ window: WINDOW }),
    ])
    return { vaults: answer.vaults.map(mapVault), showing: shown.vault }
  },
  choose: async (title) => {
    const answer = await vaultsService.chooseFolder({ title, startingAt: '' })
    return answer.chose ? answer.path : ''
  },
  add: async (path, called) => mapVaultResult(await vaultsService.addVault({ path, name: called })),
  rename: async (id, called) => mapVaultResult(await vaultsService.renameVault({ id, name: called })),
  remove: async (id, trash) => turnedDown(await vaultsService.removeVault({ id, trash })),
  open: async (id) => turnedDown(await vaultsService.openVault({ id })),
}

export type VaultsCore = Pick<CommandsDeps, 'vaults'>

export const vaultsCore: VaultsCore = {
  vaults: () => vaults.list(),
}
