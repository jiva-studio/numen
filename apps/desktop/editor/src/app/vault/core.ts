/**
 * Assembles the window core from domain module fragments.
 */
import { searchCore } from './search'
import { tabLinksCore } from './tabLinks'
import { tabSessionCore } from './tabSession'
import type { CommandsDeps, SearchDeps } from '@/features/command-palette'
import type { VaultPort } from '@/app/ports/vault'

export type VaultCore = VaultPort & SearchDeps & Pick<CommandsDeps, 'names'>

export const core: VaultCore = {
  ...tabSessionCore,
  ...tabLinksCore,
  ...searchCore,
}
