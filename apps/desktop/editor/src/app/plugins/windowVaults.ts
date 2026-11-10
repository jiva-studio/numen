/**
 * The vaults the window gives its plugins: the vaults of the installation in the order the welcome
 * screen lists them, the vault the window shows, and the switch to another one.
 */
import { computed, watch, type Ref } from 'vue'
import type { VaultsHost, VaultsState } from '@numen/plugin-loader'
import { createInvocation, runInvocation } from '@/features/command-palette'
import { COMMAND, type CommandTarget, type VaultRef, type CommandDeps } from '@/entities/command'
import type { VaultList } from '@/entities/vault'
import { formatError, type Caught } from '@/shared/api/errors'

interface WindowVaultsDeps {
  readonly vaultList: Readonly<Ref<VaultList>>
  readonly openVault: (vault: VaultRef) => Promise<void>
}

export function createWindowVaults(deps: WindowVaultsDeps): VaultsHost {
  const state = computed<VaultsState>(() => {
    const { vaults: listed, showing } = deps.vaultList.value
    const current = listed.find((one) => one.id === showing)
    return {
      vaults: listed.map(({ id, name }) => ({ id, name })),
      current: current ? { id: current.id, name: current.name } : null,
    }
  })

  return {
    getState: () => state.value,
    watch: (handler) => ({ dispose: watch(state, handler, { immediate: true }) }),
    switchTo: async (id) => {
      const one = deps.vaultList.value.vaults.find((vault) => vault.id === id)
      if (one) await deps.openVault({ id: one.id, name: one.name })
    },
  }
}

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
