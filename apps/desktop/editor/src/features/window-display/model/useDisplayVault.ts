import { ref } from 'vue'
import { getAgentUnreachable, getVaultState } from '@/entities/vault'

/** What the vault last answered about itself: its identity, its read errors and its embedding counts. */
export function useDisplayVault() {
  return {
    name: ref(''),
    at: ref(''),
    error: ref(''),
    unwatched: ref(''),
    /** Why an agent cannot be reached, as the vault last answered. */
    unreachable: ref(''),
    chunks: ref(0),
    embedded: ref(0),
    isEmbedding: ref(false),
  }
}

export type DisplayVault = ReturnType<typeof useDisplayVault>

/** Reads the vault and the agent again into the refs, and returns the vault's state. */
export async function askVaultState(vault: DisplayVault) {
  const [state, agent] = await Promise.all([getVaultState(), getAgentUnreachable()])
  vault.unreachable.value = agent
  vault.name.value = state.name
  if (vault.at.value === '') vault.at.value = state.path
  vault.error.value = state.scan.error
  vault.unwatched.value = state.scan.unwatchedPath
  vault.chunks.value = Number(state.coverage.chunkCount)
  vault.embedded.value = Number(state.coverage.embeddedCount)
  vault.isEmbedding.value = state.coverage.isEmbedding
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
