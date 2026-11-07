/**
 * The secrets shared between plugins: created, renamed, given a value, deleted, ordered, and
 * allowed or denied to the declarations of the plugins, through the vault. A value goes to the
 * vault once and is not kept here.
 */
import { formatErrorMessage } from '@numen/core-client'
import type { MessageWriter } from '@/shared/notices/messages'

/** The messages this module shows in the window. */
export interface SecretWords {
  /** The secret could not be changed. */
  readonly secretFailed: string
}

/** The vault operations this module uses. */
export interface SecretDeps {
  createSecret(group: string, name: string, value: string): Promise<string>
  renameSecret(id: string, name: string): Promise<void>
  writeSecret(id: string, value: string): Promise<void>
  deleteSecret(id: string): Promise<void>
  moveSecret(id: string, beforeId: string): Promise<void>
  renameSecretGroup(group: string, name: string): Promise<void>
  allowPluginSecret(secretId: string, pluginId: string, name: string, group: string): Promise<void>
  denyPluginSecret(secretId: string, pluginId: string, name: string, group: string): Promise<void>
}

/**
 * The messages this module shows, and how the settings are read again afterwards.
 */
export interface SecretOptions {
  readonly words: SecretWords
  readonly notify: MessageWriter
  readonly refresh: () => Promise<void>
}

export function createSecretStore(core: SecretDeps, { words, notify, refresh }: SecretOptions) {
  /** Runs an operation, shows why it failed, and reads the settings again. Resolves to its answer, or to the fallback where it failed. */
  const run = async <T>(operation: () => Promise<T>, fallback: T): Promise<T> => {
    notify('')
    let answer = fallback
    try {
      answer = await operation()
    } catch (caughtError) {
      notify(`${words.secretFailed} ${formatErrorMessage(caughtError)}`, 'error')
    }
    await refresh()
    return answer
  }

  const runVoid = (operation: () => Promise<void>): Promise<void> => run(operation, undefined)

  /** A secret created with its value. It resolves to its id, and to an empty text where nothing was created. An empty value creates nothing. */
  const createSecret = (group: string, name: string, value: string): Promise<string> =>
    value === '' ? Promise.resolve('') : run(() => core.createSecret(group, name, value), '')

  const renameSecret = (id: string, name: string): Promise<void> =>
    runVoid(() => core.renameSecret(id, name))

  /** The value of a secret replaced. An empty value changes nothing. */
  const writeSecret = (id: string, value: string): Promise<void> =>
    value === '' ? Promise.resolve() : runVoid(() => core.writeSecret(id, value))

  const deleteSecret = (id: string): Promise<void> => runVoid(() => core.deleteSecret(id))

  const moveSecret = (id: string, beforeId: string): Promise<void> =>
    runVoid(() => core.moveSecret(id, beforeId))

  const renameSecretGroup = (group: string, name: string): Promise<void> =>
    runVoid(() => core.renameSecretGroup(group, name))

  const allowPluginSecret = (
    secretId: string,
    pluginId: string,
    name: string,
    group: string,
  ): Promise<void> => runVoid(() => core.allowPluginSecret(secretId, pluginId, name, group))

  const denyPluginSecret = (
    secretId: string,
    pluginId: string,
    name: string,
    group: string,
  ): Promise<void> => runVoid(() => core.denyPluginSecret(secretId, pluginId, name, group))

  return {
    createSecret,
    renameSecret,
    writeSecret,
    deleteSecret,
    moveSecret,
    renameSecretGroup,
    allowPluginSecret,
    denyPluginSecret,
  }
}
