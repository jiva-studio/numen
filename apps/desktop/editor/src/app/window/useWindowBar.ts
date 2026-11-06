/**
 * What the window gives the view of its title bar: the vault it shows, the buttons the plugins
 * place in `titleBar`, and the vaults the title offers to switch to.
 */
import { computed, type Ref } from 'vue'
import type { Menus } from '@numen/plugin-host'
import type { CommandDeps, CommandTarget } from '@/features/command-palette'
import type { VaultList } from '@/entities/vault'
import { useWindowTitleBar } from './useTitleBarCommands'
import { useWindowVaultSwitcher } from './useVaultSwitcher'

export interface WindowBarDeps {
  readonly menus: Pick<Menus, 'getEntries'>
  readonly vaultList: Readonly<Ref<VaultList>>
  readonly runCommand: (id: string, target: CommandTarget) => void
  readonly getTarget: () => CommandTarget
  readonly commandDeps: CommandDeps
}

export function useWindowBar(deps: WindowBarDeps) {
  /** The vault the window shows; empty where none is open. */
  const vaultName = computed(
    () =>
      deps.vaultList.value.vaults.find((one) => one.id === deps.vaultList.value.showing)?.name ??
      '',
  )
  const titleBar = useWindowTitleBar({ ...deps, vaultName })
  const vaultSwitcher = useWindowVaultSwitcher(deps)
  return {
    vaultName,
    commands: titleBar.commands,
    runCommand: titleBar.runCommand,
    vaults: vaultSwitcher.vaults,
    switchVault: vaultSwitcher.switchVault,
  }
}
