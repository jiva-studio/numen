/**
 * Assembles the window core from domain module fragments.
 */
import { filesCore } from './files'
import { notesCore } from './notes'
import { searchCore } from './search'
import { sessionCore } from './session'
import { settingsCore } from './settings'
import { vaultsCore } from './vaults'
import type { CommandsDeps } from '../../shared/command/target'
import type { SearchDeps } from '../../shared/command/search'
import type { Core } from '../../shared/core'

export type VaultCore = Core & SearchDeps & CommandsDeps

export const core: VaultCore = {
  ...vaultsCore,
  ...notesCore,
  ...filesCore,
  ...settingsCore,
  ...sessionCore,
  ...searchCore,
}
