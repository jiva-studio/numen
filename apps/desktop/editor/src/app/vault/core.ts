/**
 * Assembles the window core from domain module fragments.
 */
import { searchCore } from './search'
import { settingsCore } from './settings'
import { tabLinksCore } from './tabLinks'
import { tabSessionCore } from './tabSession'
import { vaultsCore } from './vaults'
import { vaultSessionCore } from './vaultSession'
import { windowSessionCore } from './windowSession'
import type { CommandsDeps, SearchDeps } from '@/features/command-palette'
import type { Core } from '@/app/ports/core'

export type VaultCore = Core & SearchDeps & CommandsDeps

export const core: VaultCore = {
  ...vaultsCore,
  ...settingsCore,
  ...vaultSessionCore,
  ...windowSessionCore,
  ...tabSessionCore,
  ...tabLinksCore,
  ...searchCore,
}
