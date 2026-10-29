/**
 * The keys of the secrets plugins declare, stored and removed through the
 */
import { formatErrorMessage } from '@numen/plugin-sdk'
import type { MessageWriter } from '@/shared/notices/messages'

/** The messages this module shows in the window. */
export interface SecretWords {
  /** The key could not be stored or removed. */
  readonly secretFailed: string
}

/** The vault operations this module uses. */
export interface SecretDeps {
  writePluginSecret(pluginId: string, name: string, key: string): Promise<void>
  deletePluginSecret(pluginId: string, name: string): Promise<void>
}

/**
 * The messages this module shows, and how the settings are reloaded
 * afterwards.
 */
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
