/**
 * The settings core of a test, in the window's own types. Every call the window makes reaches it
 * through the real client over the router transport: this file encodes the answers the way the
 * core sends them and decodes the requests the way the core reads them.
 */
import { MODEL_PRESENCE, type ModelPresenceCode } from '@numen/core-client'
import { installCoreAnswers, type CoreAnswers } from '@numen/core-client/testing'
import type {
  Configuration,
  Model,
  PluginInfo,
  PluginSettingCondition,
  PluginSettingInfo,
  PluginSettingOption,
  Presence,
  SecretMenu,
  SecretSection,
  SettingEdit,
} from '@/entities/settings'

const toWireOption = (option: PluginSettingOption) => ({
  value: option.value,
  title: option.title,
  detail: option.detail,
  ...(option.when !== undefined ? { when: option.when } : {}),
})

const toWireCondition = (when: PluginSettingCondition) => ({
  setting: when.setting,
  is: [...when.is],
})

/** A declared setting as the core sends it: its default and its value are JSON texts. */
export const toWireSetting = (setting: PluginSettingInfo) => ({
  key: setting.key,
  type: setting.type,
  title: setting.title,
  description: setting.description,
  defaultJson: JSON.stringify(setting.default) ?? '',
  palette: Boolean(setting.palette),
  icon: setting.icon ?? '',
  preview: Boolean(setting.preview),
  multiple: Boolean(setting.multiple),
  accept: [...(setting.accept ?? [])],
  optionsFrom: setting.optionsFrom ?? '',
  options: (setting.options ?? []).map(toWireOption),
  ...(setting.optionsBy
    ? {
        optionsBy: {
          setting: setting.optionsBy.setting,
          options: Object.fromEntries(
            Object.entries(setting.optionsBy.options).map(([value, options]) => [
              value,
              { options: options.map(toWireOption) },
            ]),
          ),
        },
      }
    : {}),
  ...(setting.when ? { when: toWireCondition(setting.when) } : {}),
  activeOptions: (setting.activeOptions ?? []).map(toWireOption),
  ...(setting.min !== undefined ? { min: setting.min } : {}),
  ...(setting.max !== undefined ? { max: setting.max } : {}),
  ...(setting.step !== undefined ? { step: setting.step } : {}),
  isShown: setting.isShown ?? true,
  valueJson: JSON.stringify(setting.value) ?? '',
})

const PRESENCE: Record<Presence, ModelPresenceCode> = {
  present: MODEL_PRESENCE.PRESENT,
  'not fetched': MODEL_PRESENCE.NOT_FETCHED,
  'nothing to fetch': MODEL_PRESENCE.NOTHING_TO_FETCH,
}

const toWireModel = (model: Model) => ({
  namedAt: [...model.namedAt],
  name: model.name,
  title: model.title,
  group: model.group,
  isDefault: model.isDefault,
  writes: model.writes.map((one) => ({ at: [...one.at], value: one.value })),
  presence: PRESENCE[model.presence],
  reasoningLevels: model.reasoningLevels ?? [],
  defaultReasoning: model.defaultReasoning ?? '',
})

const toWirePlugin = (plugin: PluginInfo) => ({
  id: plugin.id,
  name: plugin.name,
  version: plugin.version,
  description: plugin.description,
  isActive: plugin.isActive,
  capabilities: [...plugin.capabilities],
  secrets: plugin.secrets.map((one) => ({
    name: one.name,
    title: one.title,
    description: one.description,
    group: one.group,
    ...(one.when ? { when: toWireCondition(one.when) } : {}),
  })),
  hasFrontend: plugin.hasFrontend,
  hasBackend: plugin.hasBackend,
  readers: (plugin.readers ?? []).map((one) => ({ id: one.id, extensions: [...one.extensions] })),
  settingsViews: plugin.settingsViews ?? [],
  ...(plugin.failure
    ? {
        failure: {
          reason: plugin.failure.reason,
          detail: plugin.failure.detail,
          failedAtUnixMs: BigInt(plugin.failure.failedAt),
        },
      }
    : {}),
  settings: plugin.settings.map(toWireSetting),
})

const toWireSections = (sections: readonly SecretSection[] = []) =>
  sections.map((section) => ({
    group: section.group,
    isAsked: section.isAsked,
    secrets: section.secrets.map((one) => ({
      id: one.id,
      name: one.name,
      group: one.group,
      allowed: one.allowed.map((entry) => ({ ...entry })),
    })),
  }))

const toWireMenus = (menus: readonly SecretMenu[] = []) =>
  menus.map((menu) => ({
    pluginId: menu.pluginId,
    name: menu.name,
    group: menu.group,
    secretIds: menu.secrets.map((one) => one.id),
  }))

const toWireConfiguration = (configuration: Configuration) => ({
  written: configuration.written,
  path: configuration.path,
  models: configuration.models.map(toWireModel),
  plugins: (configuration.plugins ?? []).map(toWirePlugin),
  app: (configuration.app ?? []).map(toWireSetting),
  secretSections: toWireSections(configuration.secretSections),
  secretMenus: toWireMenus(configuration.secretMenus),
  credentialsPath: configuration.credentialsPath ?? '',
})

/** What the core does with the settings, in the window's types. A call left out is unanswered. */
export interface SettingsCore {
  getSettings(): Promise<Configuration>
  updateSettings(edits: readonly SettingEdit[]): Promise<void>
  watchSettings(signal: AbortSignal): AsyncIterable<readonly SettingEdit[]>
  createSecret(group: string, name: string, value: string): Promise<string>
  renameSecret(id: string, name: string): Promise<void>
  writeSecret(id: string, value: string): Promise<void>
  deleteSecret(id: string): Promise<void>
  deleteSecretGroup(group: string): Promise<void>
  listSecretStates(): Promise<ReadonlyMap<string, boolean>>
  moveSecret(id: string, beforeId: string): Promise<void>
  renameSecretGroup(group: string, name: string): Promise<void>
  allowPluginSecret(secretId: string, pluginId: string, name: string, group: string): Promise<void>
  denyPluginSecret(secretId: string, pluginId: string, name: string, group: string): Promise<void>
  retryPlugin(pluginId: string): Promise<string>
  choosePluginPath(
    pluginId: string,
    key: string,
  ): Promise<{ readonly path: string; readonly isChosen: boolean; readonly problem: string }>
  getPluginData(pluginId: string): Promise<number>
  deletePluginData(pluginId: string): Promise<void>
}

/**
 * Answers the settings calls of every client from `core`, through the real clients. The returned
 * function restores the transport installed before. `others` answers the calls of the other
 * clients; the plugin catalog lists nothing unless it says otherwise.
 */
export const installSettingsCore = (
  core: Partial<SettingsCore>,
  others: CoreAnswers = {},
): (() => void) => {
  const answers: Record<string, unknown> = {}
  const { getSettings, updateSettings, watchSettings, ...secrets } = core
  if (getSettings) {
    answers['getSettings'] = async () => toWireConfiguration(await getSettings())
  }
  if (updateSettings) {
    answers['writeSettings'] = async (request: { settings: readonly SettingEdit[] }) => {
      await updateSettings(request.settings.map((one) => ({ at: [...one.at], value: one.value })))
      return {}
    }
  }
  if (watchSettings) {
    answers['watchSettings'] = async function* (
      _request: unknown,
      context: { signal: AbortSignal },
    ) {
      for await (const batch of watchSettings(context.signal)) yield { changed: batch }
    }
  }
  if (secrets.createSecret) {
    const createSecret = secrets.createSecret
    answers['createSecret'] = async (r: { group: string; name: string; value: string }) => ({
      id: await createSecret(r.group, r.name, r.value),
    })
  }
  if (secrets.renameSecret) {
    const renameSecret = secrets.renameSecret
    answers['renameSecret'] = async (r: { id: string; name: string }) => {
      await renameSecret(r.id, r.name)
      return {}
    }
  }
  if (secrets.writeSecret) {
    const writeSecret = secrets.writeSecret
    answers['writeSecret'] = async (r: { id: string; value: string }) => {
      await writeSecret(r.id, r.value)
      return {}
    }
  }
  if (secrets.deleteSecret) {
    const deleteSecret = secrets.deleteSecret
    answers['deleteSecret'] = async (r: { id: string }) => {
      await deleteSecret(r.id)
      return {}
    }
  }
  if (secrets.deleteSecretGroup) {
    const deleteSecretGroup = secrets.deleteSecretGroup
    answers['deleteSecretGroup'] = async (r: { group: string }) => {
      await deleteSecretGroup(r.group)
      return {}
    }
  }
  if (secrets.listSecretStates) {
    const listSecretStates = secrets.listSecretStates
    answers['listSecretStates'] = async () => ({
      states: [...(await listSecretStates())].map(([id, hasValue]) => ({ id, hasValue })),
    })
  }
  if (secrets.moveSecret) {
    const moveSecret = secrets.moveSecret
    answers['moveSecret'] = async (r: { id: string; beforeId: string }) => {
      await moveSecret(r.id, r.beforeId)
      return {}
    }
  }
  if (secrets.renameSecretGroup) {
    const renameSecretGroup = secrets.renameSecretGroup
    answers['renameSecretGroup'] = async (r: { group: string; name: string }) => {
      await renameSecretGroup(r.group, r.name)
      return {}
    }
  }
  if (secrets.allowPluginSecret) {
    const allowPluginSecret = secrets.allowPluginSecret
    answers['allowPluginSecret'] = async (r: {
      secretId: string
      pluginId: string
      name: string
      group: string
    }) => {
      await allowPluginSecret(r.secretId, r.pluginId, r.name, r.group)
      return {}
    }
  }
  if (secrets.denyPluginSecret) {
    const denyPluginSecret = secrets.denyPluginSecret
    answers['denyPluginSecret'] = async (r: {
      secretId: string
      pluginId: string
      name: string
      group: string
    }) => {
      await denyPluginSecret(r.secretId, r.pluginId, r.name, r.group)
      return {}
    }
  }
  if (secrets.retryPlugin) {
    const retryPlugin = secrets.retryPlugin
    answers['retryPlugin'] = async (r: { pluginId: string }) => ({
      failure: await retryPlugin(r.pluginId),
    })
  }
  if (secrets.choosePluginPath) {
    const choosePluginPath = secrets.choosePluginPath
    answers['choosePluginPath'] = (r: { pluginId: string; key: string }) =>
      choosePluginPath(r.pluginId, r.key)
  }
  if (secrets.getPluginData) {
    const getPluginData = secrets.getPluginData
    answers['getPluginData'] = async (r: { pluginId: string }) => ({
      fileCount: await getPluginData(r.pluginId),
    })
  }
  if (secrets.deletePluginData) {
    const deletePluginData = secrets.deletePluginData
    answers['deletePluginData'] = async (r: { pluginId: string }) => {
      await deletePluginData(r.pluginId)
      return {}
    }
  }
  return installCoreAnswers({
    pluginCatalog: { listCatalog: () => ({ entries: [], sourceErrors: [] }) },
    ...others,
    settingsService: answers as never,
  })
}
