/**
 * Gets a vault of the cloud onto this device: one question, where to put it, and then the vault
 * opens while it downloads.
 */
import type { PickerHost } from '@numen/plugin-loader'
import type { CloudVault, RemoteVaults, Vaults } from '@/entities/vault'
import type { MessageWriter } from '@/shared/model/notices/messages'
import { VAULT_WORDS } from '../words'

const USE_FOLDER = 'use'
const CHOOSE_FOLDER = 'choose'

/** What getting a cloud vault uses of the window. */
export interface CloudOpenDeps {
  readonly vaults: Pick<Vaults, 'choose'>
  readonly remote: Pick<RemoteVaults, 'suggestPath' | 'open'>
  readonly ask: Pick<PickerHost, 'selectOption'>
  readonly showVault: (id: string) => Promise<void>
  readonly writeMessage: MessageWriter
}

/**
 * Asks where the vault goes until the person answers or leaves. The answer is the path to use, and
 * nothing where the person dismissed the question or the core refused the chosen folder.
 */
const askPlace = async (vault: CloudVault, first: string, deps: CloudOpenDeps): Promise<string> => {
  let path = first
  for (;;) {
    const answer = await deps.ask.selectOption({
      title: VAULT_WORDS.cloudWhere(vault.name, path),
      options: [
        { value: USE_FOLDER, title: VAULT_WORDS.useFolder },
        { value: CHOOSE_FOLDER, title: VAULT_WORDS.chooseFolder },
      ],
    })
    if (answer === undefined) return ''
    if (answer === USE_FOLDER) return path
    const folder = await deps.vaults.choose(VAULT_WORDS.folder)
    if (!folder) continue
    const suggested = await deps.remote.suggestPath(vault.name, folder)
    if (!suggested.ok) {
      deps.writeMessage(suggested.error || VAULT_WORDS.cloudFailed, 'error')
      return ''
    }
    path = suggested.value.path
  }
}

/** Gets `vault` from the cloud and shows it; a refusal is written as the sentence the core gave. */
export const openCloudVault = async (vault: CloudVault, deps: CloudOpenDeps): Promise<void> => {
  const suggested = await deps.remote.suggestPath(vault.name)
  if (!suggested.ok) return deps.writeMessage(suggested.error || VAULT_WORDS.cloudFailed, 'error')
  const path = await askPlace(vault, suggested.value.path, deps)
  if (!path) return
  const opened = await deps.remote.open(vault.remoteId, path)
  if (!opened.ok) return deps.writeMessage(opened.error || VAULT_WORDS.cloudFailed, 'error')
  await deps.showVault(opened.value)
}
