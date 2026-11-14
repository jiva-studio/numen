import { getVaultForKey } from '@numen/workbench'
import type { Vault } from '@/entities/vault'
import type { WindowTab } from '@/entities/tab'

export interface WindowKeysDeps {
  readonly getTab: () => WindowTab | null
  readonly getVaults: () => readonly Vault[]
  readonly openVault: (id: string) => void
}

/**
 * What a keystroke does in the window: the current tab handles it when one is
 * open, and on the list of vaults a letter opens the vault at that position.
 */
export const createWindowKeys = (deps: WindowKeysDeps) => (press: KeyboardEvent) => {
  const tab = deps.getTab()
  if (tab) {
    if (tab.state && tab.kind.events?.onKeyPress?.(tab.state, press)) press.preventDefault()
    return
  }
  const vaults = deps.getVaults()
  const vault = vaults[getVaultForKey(press, vaults.length) ?? -1]
  if (!vault) return
  press.preventDefault()
  deps.openVault(vault.id)
}
