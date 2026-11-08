import type { WelcomeAction } from '@numen/shell'
import type { Tab } from '@numen/shell'
import type { CommandDeps, Commands, CommandTarget, SearchState } from '@/features/command-palette'
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
  openCurrentVault?: () => void
}
