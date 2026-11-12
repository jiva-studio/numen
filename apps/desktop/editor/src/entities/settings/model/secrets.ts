/**
 * The secrets shared between plugins: created, renamed, given a value, deleted, ordered, and
 * allowed or denied to the declarations of the plugins, through the vault. A value goes to the
 * vault once and is not kept here.
 */
import { shallowRef } from 'vue'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import * as secretsApi from '../api/secrets'

/** The messages this module shows in the window. */
interface SecretWords {
  readonly secretFailed: string
}

/**
 * The messages this module shows, and how the settings are read again afterwards.
 */
interface SecretOptions {
  readonly words: SecretWords
  readonly notify: MessageWriter
  readonly refresh: () => Promise<void>
}

export function createSecretStore({ words, notify, refresh }: SecretOptions) {
  /** Whether each secret has a value on this machine, as the core last reported it. */
  const states = shallowRef<ReadonlyMap<string, boolean>>(new Map())

  /** Reads which secrets have a value. A failed read keeps the states as they were. */
  const loadStates = async (): Promise<void> => {
    try {
      states.value = await secretsApi.listSecretStates()
    } catch (caughtError) {
      notify(`${words.secretFailed} ${formatError(caughtError)}`, 'error')
    }
  }

  /**
   * Runs an operation, shows why it failed, and reads the settings again, and the states where the
   * operation changes a value. Resolves to its answer, or to the fallback where it failed.
   */
  const run = async <T>(
    operation: () => Promise<T>,
    fallback: T,
    changesValue = false,
  ): Promise<T> => {
    notify('')
    let answer = fallback
    try {
      answer = await operation()
    } catch (caughtError) {
      notify(`${words.secretFailed} ${formatError(caughtError)}`, 'error')
    }
    await refresh()
    if (changesValue) await loadStates()
    return answer
  }

  const runVoidOperation = (operation: () => Promise<void>, changesValue = false): Promise<void> =>
    run(operation, undefined, changesValue)

  /** A secret created with its value. It resolves to its id, and to an empty text where nothing was created. An empty value creates nothing. */
  const createSecret = (group: string, name: string, value: string): Promise<string> =>
    value === ''
      ? Promise.resolve('')
      : run(() => secretsApi.createSecret(group, name, value), '', true)

  const renameSecret = (id: string, name: string): Promise<void> =>
    runVoidOperation(() => secretsApi.renameSecret(id, name))

  /** The value of a secret replaced. An empty value changes nothing. */
  const writeSecret = (id: string, value: string): Promise<void> =>
    value === '' ? Promise.resolve() : runVoidOperation(() => secretsApi.writeSecret(id, value), true)

  const deleteSecret = (id: string): Promise<void> =>
    runVoidOperation(() => secretsApi.deleteSecret(id), true)

  const deleteSecretGroup = (group: string): Promise<void> =>
    runVoidOperation(() => secretsApi.deleteSecretGroup(group), true)

  const moveSecret = (id: string, beforeId: string): Promise<void> =>
    runVoidOperation(() => secretsApi.moveSecret(id, beforeId))

  const renameSecretGroup = (group: string, name: string): Promise<void> =>
    runVoidOperation(() => secretsApi.renameSecretGroup(group, name))

  const allowPluginSecret = (
    secretId: string,
    pluginId: string,
    name: string,
    group: string,
  ): Promise<void> => runVoidOperation(() => secretsApi.allowPluginSecret(secretId, pluginId, name, group))

  const denyPluginSecret = (
    secretId: string,
    pluginId: string,
    name: string,
    group: string,
  ): Promise<void> => runVoidOperation(() => secretsApi.denyPluginSecret(secretId, pluginId, name, group))

  return {
    states,
    loadStates,
    edit: {
      createSecret,
      renameSecret,
      writeSecret,
      deleteSecret,
      deleteSecretGroup,
      moveSecret,
      renameSecretGroup,
    },
    access: { allowPluginSecret, denyPluginSecret },
  }
}
