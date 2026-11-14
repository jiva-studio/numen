import { ref } from 'vue'
import { getVaultState } from '@/entities/vault'

/** What the vault last answered about itself: its identity and its read errors. */
export function useDisplayVault() {
  return {
    name: ref(''),
    at: ref(''),
    error: ref(''),
    unwatched: ref(''),
  }
}

export type DisplayVault = ReturnType<typeof useDisplayVault>

/** Reads the vault again into the refs, and returns the vault's state. */
export async function askVaultState(vault: DisplayVault) {
  const state = await getVaultState()
  vault.name.value = state.name
  if (vault.at.value === '') vault.at.value = state.path
  vault.error.value = state.scan.error
  vault.unwatched.value = state.scan.unwatchedPath
  return state
}

/** Whether another vault is now under the window than the one it last read. */
export async function isVaultSwapped(vault: DisplayVault): Promise<boolean> {
  const was = vault.at.value
  /* Stryker disable BlockStatement: callers only test truthiness of isVaultSwapped */
  try {
    const state = await askVaultState(vault)
    return was !== '' && state.path !== was
  } catch {
    return false
  }
  /* Stryker restore BlockStatement */
}
