import { type CommandDeps } from '@/entities/command'
import type { WelcomeAction } from '@numen/workbench'
import type { Tab } from '@numen/workbench'
import type { Commands, SearchState } from '@/features/command-palette'
import type { CommandTarget } from '@/entities/command'
import type { VaultList } from '@/entities/vault'

export interface WelcomeScreenProps {
  vaultList: VaultList
  tabs: readonly Tab[]
  commands: Commands
  search: SearchState
  doing: CommandDeps
  getTarget: () => CommandTarget
  pluginWays: readonly WelcomeAction[]
  runCommand: (id: string, target: CommandTarget) => void
  agent?: string
  openCurrentVault?: () => void
}
