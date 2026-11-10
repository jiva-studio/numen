/**
 * The vaults the window gives its plugins: the vaults of the installation in the order the welcome
 * screen lists them, the vault the window shows, and the switch to another one.
 */
import type { Ref } from 'vue'
import type { VaultsHost } from '@numen/plugin-loader'
import { createInvocation, runInvocation } from '@/features/command-palette'
import { COMMAND, type CommandTarget, type CommandDeps } from '@/entities/command'
import type { VaultList } from '@/entities/vault'
import { formatError, type Caught } from '@/shared/api/errors'
import { createWindowVaults } from '@/features/plugin-commands'

/** The vaults of the editor over its vault list, opening a vault through the use case of the palette. */
export function createEditorVaults(window: {
  readonly vaultList: Readonly<Ref<VaultList>>
  readonly getTarget: () => CommandTarget
  readonly commandDeps: CommandDeps
}): VaultsHost {
  const { commandDeps } = window
  return createWindowVaults({
    vaultList: window.vaultList,
    openVault: (vault) =>
      runInvocation(
        createInvocation(COMMAND.openVault, { ...window.getTarget(), vault }),
        commandDeps,
      ).then(
        () => undefined,
        (error: Caught) => commandDeps.writeMessage(formatError(error), 'error'),
      ),
  })
}
