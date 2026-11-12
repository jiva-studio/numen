/**
 * The secrets shared between plugins. A value is stored in the system keychain, or in a protected
 * file where there is none, and is never read back or written to the settings file. A machine that
 * can store neither rejects the call.
 */
import { settingsService } from '@/shared/api/clients'

export const createSecret = async (group: string, name: string, value: string): Promise<string> =>
  (await settingsService.createSecret({ group, name, value })).id

export const renameSecret = async (id: string, name: string): Promise<void> => {
  await settingsService.renameSecret({ id, name })
}

export const writeSecret = async (id: string, value: string): Promise<void> => {
  await settingsService.writeSecret({ id, value })
}

/** A secret removed with its value and its permissions. */
export const deleteSecret = async (id: string): Promise<void> => {
  await settingsService.deleteSecret({ id })
}

/** Every secret of a group removed with its values and its permissions. */
export const deleteSecretGroup = async (group: string): Promise<void> => {
  await settingsService.deleteSecretGroup({ group })
}

/**
 * For each secret id, whether its value is on this machine. The core reads the key store for each
 * secret, so only the page of secrets asks.
 */
export const listSecretStates = async (): Promise<ReadonlyMap<string, boolean>> => {
  const answer = await settingsService.listSecretStates({})
  return new Map(answer.states.map((one) => [one.id, one.hasValue]))
}

/** A secret put before another one of its group; an empty `beforeId` puts it at the end. */
export const moveSecret = async (id: string, beforeId: string): Promise<void> => {
  await settingsService.moveSecret({ id, beforeId })
}

export const renameSecretGroup = async (group: string, name: string): Promise<void> => {
  await settingsService.renameSecretGroup({ group, name })
}

/** One declaration of a plugin let to read a secret. The group is empty for a declaration with none. */
export const allowPluginSecret = async (
  secretId: string,
  pluginId: string,
  name: string,
  group: string,
): Promise<void> => {
  await settingsService.allowPluginSecret({ secretId, pluginId, name, group })
}

export const denyPluginSecret = async (
  secretId: string,
  pluginId: string,
  name: string,
  group: string,
): Promise<void> => {
  await settingsService.denyPluginSecret({ secretId, pluginId, name, group })
}
