/**
 * The keys of the secrets plugins declare, stored and removed through the
 */
import { formatErrorMessage } from '@numen/wire'
import type { MessageWriter } from '@/shared/notices/messages'

/** Everything this says in the window's voice. */
export interface SecretWords {
  /** The key could not be stored or removed. */
  readonly secretFailed: string
}

/** What this asks of the vault. */
export interface SecretDeps {
  writePluginSecret(pluginId: string, name: string, key: string): Promise<void>
  deletePluginSecret(pluginId: string, name: string): Promise<void>
}

/** Where this says what happened, and how the settings are read again after. */
export interface SecretOptions {
  readonly words: SecretWords
  readonly notify: MessageWriter
  readonly refresh: () => Promise<void>
}

export function createSecretStore(core: SecretDeps, { words, notify, refresh }: SecretOptions) {
  const run = async (operation: () => Promise<void>): Promise<void> => {
    notify('')
    try {
      await operation()
    } catch (caughtError) {
      notify(`${words.secretFailed} ${formatErrorMessage(caughtError)}`, 'error')
    }
    await refresh()
  }

  /** The key of one secret stored. An empty key stores nothing. */
  const writeSecret = (pluginId: string, name: string, key: string): Promise<void> =>
    key === '' ? Promise.resolve() : run(() => core.writePluginSecret(pluginId, name, key))

  /** The key of one secret removed. */
  const deleteSecret = (pluginId: string, name: string): Promise<void> =>
    run(() => core.deletePluginSecret(pluginId, name))

  return { writeSecret, deleteSecret }
}
