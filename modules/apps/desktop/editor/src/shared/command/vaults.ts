/**
 * What a command does to the vaults this installation holds: showing another
 * one, adding one, calling one something else, taking one off the list.
 *
 * A vault is not a note. What it holds is the person's folder, and every one
 * of these leaves the window standing somewhere else than it stood.
 */
import type { CommandDeps, Words } from './deps'
import type { CommandInvocation } from './target'

/**
 * Another vault under this window. What the page holds belongs to the vault
 * that has gone, so the page is drawn again on the one that arrived.
 */
export const shows = async (id: string, on: CommandDeps, words: Words): Promise<void> => {
  if (!id) return
  const refusal = await on.vaults.open(id)
  if (refusal) return on.says(words.unvaulted[refusal], 'refusal')
  on.vaults.reloads()
}

/**
 * A folder chosen on this machine, added as a vault and opened. A person who
 * chose no folder has asked for nothing.
 */
export const adds = async (on: CommandDeps, words: Words): Promise<void> => {
  const path = await on.vaults.choose(words.folder)
  if (!path) return
  const answer = await on.vaults.add(path, '')
  if (answer.refusal) return on.says(words.unvaulted[answer.refusal], 'refusal')
  if (!answer.vault) return
  await shows(answer.vault.id, on, words)
}

/** A vault called something else. Its folder keeps the name it has on disk. */
export const calls = async (
  invocation: CommandInvocation,
  on: CommandDeps,
  words: Words,
): Promise<void> => {
  if (!invocation.name || invocation.name === invocation.vault.name) return
  const answer = await on.vaults.rename(invocation.vault.id, invocation.name)
  if (answer.refusal) return on.says(words.unvaulted[answer.refusal], 'refusal')
  if (answer.vault) on.vaults.calls({ id: answer.vault.id, name: answer.vault.name })
}

/**
 * A vault taken off the list. Erasing it puts the folder in the trash this
 * machine keeps; forgetting it leaves the folder where it is.
 */
export const forgets = async (
  invocation: CommandInvocation,
  erase: boolean,
  on: CommandDeps,
  words: Words,
): Promise<void> => {
  const id = invocation.vault.id
  const refusal = await on.vaults.remove(id, erase)
  if (refusal) on.says(words.unvaulted[refusal], 'refusal')
}

