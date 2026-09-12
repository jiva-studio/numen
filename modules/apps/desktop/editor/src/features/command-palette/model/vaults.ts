/**
 * What a command does to the vaults this installation holds: showing another
 * one, adding one, calling one something else, taking one off the list.
 *
 * A vault is not a note. What it holds is the person's folder, and every one
 * of these leaves the window standing somewhere else than it stood.
 */
import type { VaultContext, Voice, Words } from '../deps'
import type { CommandInvocation } from '../target'

/**
 * Another vault under this window. What the page holds belongs to the vault
 * that has gone, so the page is drawn again on the one that arrived.
 */
export const showVault = async (id: string, on: VaultContext & Voice, words: Words): Promise<void> => {
  if (!id) return
  const error = await on.vaults.open(id)
  if (error) return on.says(words.vaultErrors[error], 'error')
  on.vaults.reload()
}

/**
 * A folder chosen on this machine, added as a vault and opened. A person who
 * chose no folder has asked for nothing.
 */
export const addVault = async (on: VaultContext & Voice, words: Words): Promise<void> => {
  const path = await on.vaults.choose(words.folder)
  if (!path) return
  const answer = await on.vaults.add(path, '')
  if (answer.error) return on.says(words.vaultErrors[answer.error], 'error')
  if (!answer.vault) return
  await showVault(answer.vault.id, on, words)
}

/** A vault called something else. Its folder keeps the name it has on disk. */
export const renameVault = async (
  invocation: CommandInvocation,
  on: VaultContext & Voice,
  words: Words,
): Promise<void> => {
  if (!invocation.name || invocation.name === invocation.vault.name) return
  const answer = await on.vaults.rename(invocation.vault.id, invocation.name)
  if (answer.error) return on.says(words.vaultErrors[answer.error], 'error')
  if (answer.vault) on.vaults.calls({ id: answer.vault.id, name: answer.vault.name })
}

/**
 * A vault taken off the list. Erasing it puts the folder in the trash this
 * machine keeps; forgetting it leaves the folder where it is.
 */
export const removeVault = async (
  invocation: CommandInvocation,
  erase: boolean,
  on: VaultContext & Voice,
  words: Words,
): Promise<void> => {
  const id = invocation.vault.id
  const error = await on.vaults.remove(id, erase)
  if (error) on.says(words.vaultErrors[error], 'error')
}

