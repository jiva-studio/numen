/**
 * What the title of the window offers to switch to: the vaults of the installation in the order the
 * welcome screen lists them, and a last row that adds a folder as a vault.
 */
import { computed, type ComputedRef, type Ref } from 'vue'
import { formatErrorMessage } from '@numen/core-client'
import type { TitleBarVault } from '@numen/plugin-host'
import { runInBackground } from '@numen/ui'
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

export interface TitleVaultsDeps {
  readonly vaultList: Readonly<Ref<VaultList>>
  readonly words: { readonly add: string }
  /** The runner the palette and the shortcuts use. */
  readonly runCommand: (id: string, target: CommandTarget) => void
  /** Shows a vault of the list in this window, as the welcome screen does. */
  readonly openVault: (vault: VaultRef) => void
  readonly getTarget: () => CommandTarget
}

export interface TitleVaults {
  /** Empty while the window shows no vault. */
  readonly vaults: ComputedRef<readonly TitleBarVault[]>
  /** Shows the vault of the id, or adds a vault for the last row. */
  switchVault(id: string): void
}

export function useVaultSwitcher(deps: TitleVaultsDeps): TitleVaults {
  const vaults = computed<readonly TitleBarVault[]>(() => {
    const { vaults: listed, showing } = deps.vaultList.value
    if (showing === '') return []
    return [
      ...listed.map((one) => ({ id: one.id, name: one.name, isCurrent: one.id === showing })),
      { id: COMMAND.createVault, name: deps.words.add },
    ]
  })

  const switchVault = (id: string): void => {
    if (id === COMMAND.createVault) return deps.runCommand(COMMAND.createVault, deps.getTarget())
    const one = deps.vaultList.value.vaults.find((vault) => vault.id === id)
    if (one) deps.openVault({ id: one.id, name: one.name })
  }

  return { vaults, switchVault }
}

/** The vaults of a window over its vault list, opening a vault through the use case of the palette. */
export function useWindowVaultSwitcher(
  window: Pick<TitleVaultsDeps, 'vaultList' | 'runCommand' | 'getTarget'> & {
    readonly commandDeps: CommandDeps
  },
): TitleVaults {
  const { commandDeps } = window
  return useVaultSwitcher({
    vaultList: window.vaultList,
    words: { add: WORDS.newVault },
    runCommand: window.runCommand,
    getTarget: window.getTarget,
    openVault: (vault) =>
      runInBackground(
        runInvocation(
          createInvocation(COMMAND.openVault, { ...window.getTarget(), vault }),
          commandDeps,
          ANSWER_WORDS,
        ),
        (error) => commandDeps.writeMessage(formatErrorMessage(error), 'error'),
      ),
  })
}
