import type { WelcomeAction } from '@numen/shell'
import type { Tab } from '@numen/shell'
import type { CommandDeps, Commands, CommandTarget, SearchState } from '@/features/command-palette'
import type { VaultList } from '@/entities/vault'

export interface WelcomeScreenProps {
  /** Every vault the installation holds, as the list last answered. */
  vaultList: VaultList
  /** Every tab the window holds, which is none while this screen is what shows. */
  tabs: readonly Tab[]
  commands: Commands
  search: SearchState
  doing: CommandDeps
  getTarget: () => CommandTarget
  /** The commands the plugins declare over the window with a keystroke, offered as ways in. */
  pluginWays: readonly WelcomeAction[]
  /** A command asked for, which is what every way in but the commands evaluates to. */
  runCommand: (id: string, target: CommandTarget) => void
}
