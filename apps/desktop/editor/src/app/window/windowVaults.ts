/**
 * The vaults the window gives its plugins: the vaults of the installation in the order the welcome
 * screen lists them and a last row that adds a folder as a vault, the vault the window shows, and
 * the switch to another one.
 */
import { computed, watch, type Ref } from 'vue'
import { formatErrorMessage } from '@numen/core-client'
import type { VaultsHost, VaultsState } from '@numen/plugin-host'
import {
  ANSWER_WORDS,
  createInvocation,
  runInvocation,
  type CommandDeps,
  type CommandTarget,
  type VaultRef,
} from '@/features/command-palette'
import type { VaultList } from '@/entities/vault'
import { COMMAND } from '@/shared/commandIds'
import { WORDS } from '@/shared/words'

interface WindowVaultsDeps {
  readonly vaultList: Readonly<Ref<VaultList>>
  readonly words: { readonly add: string }
  /** The runner the palette and the shortcuts use. */
  readonly runCommand: (id: string, target: CommandTarget) => void
  /** Shows a vault of the list in this window, as the welcome screen does, and reports a failure. */
  readonly openVault: (vault: VaultRef) => Promise<void>
  readonly getTarget: () => CommandTarget
}

export function createWindowVaults(deps: WindowVaultsDeps): VaultsHost {
  const state = computed<VaultsState>(() => {
    const { vaults: listed, showing } = deps.vaultList.value
    const current = listed.find((one) => one.id === showing)
    return {
      vaults: [
        ...listed.map(({ id, name }) => ({ id, name })),
        { id: COMMAND.createVault, name: deps.words.add },
      ],
      current: current ? { id: current.id, name: current.name } : null,
    }
  })

  return {
    getState: () => state.value,
    watch: (handler) => ({ dispose: watch(state, handler, { immediate: true }) }),
    switchTo: async (id) => {
      if (id === COMMAND.createVault) return deps.runCommand(COMMAND.createVault, deps.getTarget())
      const one = deps.vaultList.value.vaults.find((vault) => vault.id === id)
      if (one) await deps.openVault({ id: one.id, name: one.name })
    },
  }
}

/** The vaults of the editor over its vault list, opening a vault through the use case of the palette. */
export function createEditorVaults(
  window: Pick<WindowVaultsDeps, 'vaultList' | 'runCommand' | 'getTarget'> & {
    readonly commandDeps: CommandDeps
  },
): VaultsHost {
  const { commandDeps } = window
  return createWindowVaults({
    vaultList: window.vaultList,
    words: { add: WORDS.newVault },
    runCommand: window.runCommand,
    getTarget: window.getTarget,
    openVault: (vault) =>
      runInvocation(
        createInvocation(COMMAND.openVault, { ...window.getTarget(), vault }),
        commandDeps,
        ANSWER_WORDS,
      ).then(
        () => undefined,
        (error: unknown) => commandDeps.writeMessage(formatErrorMessage(error), 'error'),
      ),
  })
}
