/**
 * What a keystroke does in the window: the current tab handles it when one is
 * open, and on the list of vaults a letter opens the vault at that position.
 */
import { getVaultForKey, type VaultRow } from '@numen/shell'
import type { WindowTab } from './host'

export interface WindowKeysDeps {
  readonly getTab: () => WindowTab | null
  /** The vaults on the list, in the order they are drawn. */
  readonly getVaults: () => readonly VaultRow[]
  readonly openVault: (id: string) => void
}

export const createWindowKeys = (deps: WindowKeysDeps) => (press: KeyboardEvent) => {
  const tab = deps.getTab()
  if (tab) {
    if (tab.view.registration.onKeyPress?.(tab.state, press)) press.preventDefault()
    return
  }
  const vaults = deps.getVaults()
  const vault = vaults[getVaultForKey(press, vaults.length) ?? -1]
  if (!vault) return
  press.preventDefault()
  deps.openVault(vault.id)
}
