/**
 * The window core: the search the window and its command palette ask.
 */
import { searchCore } from './search'
import type { CommandsDeps, SearchDeps } from '@/features/command-palette'

export type VaultCore = SearchDeps & Pick<CommandsDeps, 'names'>

export const core: VaultCore = searchCore
