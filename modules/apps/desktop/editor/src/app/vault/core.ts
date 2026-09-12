/**
 * Assembles the window core from domain module fragments.
 */
import { filesCore } from './files'
import { notesCore } from './notes'
import { searchCore } from './search'
import { sessionCore } from './session'
import { settingsCore } from './settings'
import { vaultsCore } from './vaults'
import type { CommandsDeps } from '@/features/command-palette/target'
import type { SearchDeps } from '@/features/command-palette/search'
import type { Core } from '@/app/ports/core'

export type VaultCore = Core & SearchDeps & CommandsDeps

export const core: VaultCore = {
  ...vaultsCore,
  ...notesCore,
  ...filesCore,
  ...settingsCore,
  ...sessionCore,
  ...searchCore,
}
