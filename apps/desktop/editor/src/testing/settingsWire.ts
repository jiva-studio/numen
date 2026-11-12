/**
 * The settings core of a test, in the window's own types. Every call the window makes reaches it
 * through the real client over the router transport: this file encodes the answers the way the
 * core sends them and decodes the requests the way the core reads them.
 */
import { PLUGIN_CONTRIBUTION_KIND } from '@numen/core-client'
import { installCoreAnswers, type CoreAnswers } from '@numen/core-client/testing'
import type { PluginSettingCondition } from '@numen/plugin-manager'
import type {
  PluginInfo,
  PluginSettingInfo,
  PluginSettingOption,
  SettingEdit,
} from '@/entities/settings'

/** Every setting in effect, as a test core answers them. */
interface Configuration {
  readonly jsonContent?: string | undefined
  readonly written: string
  readonly path: string
  readonly plugins?: readonly PluginInfo[] | undefined
  readonly app?: readonly PluginSettingInfo[] | undefined
}

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

const toWireOptions = (setting: PluginSettingInfo) => ({
  optionsFrom: setting.optionsFrom ?? '',
  operations: [...(setting.operations ?? [])],
  options: (setting.options ?? []).map(toWireOption),
  activeOptions: (setting.activeOptions ?? []).map(toWireOption),
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
})

const toWireBounds = (setting: PluginSettingInfo) => ({
  ...(setting.min !== undefined ? { min: setting.min } : {}),
  ...(setting.max !== undefined ? { max: setting.max } : {}),
  ...(setting.step !== undefined ? { step: setting.step } : {}),
})

/** A declared setting as the core sends it: its default and its value are JSON texts. */
export const toWireSetting = (setting: PluginSettingInfo) => ({
  key: setting.key,
  type: setting.type,
  title: setting.title,
  description: setting.description,
  defaultJson: JSON.stringify(setting.default),
  palette: Boolean(setting.palette),
  icon: setting.icon ?? '',
  preview: Boolean(setting.preview),
  multiple: Boolean(setting.multiple),
  accept: [...(setting.accept ?? [])],
  ...toWireOptions(setting),
  ...toWireBounds(setting),
  isShown: setting.isShown ?? true,
  valueJson: JSON.stringify(setting.value),
})

const toCamelCase = (constant: string): string =>
  constant.toLowerCase().replace(/_([a-z])/g, (_match, letter: string) => letter.toUpperCase())

const WIRE_CONTRIBUTION_KINDS = new Map(
  Object.entries(PLUGIN_CONTRIBUTION_KIND).flatMap(([name, value]) =>
    typeof value === 'number' ? [[toCamelCase(name), value] as const] : [],
  ),
)

const toWireContributions = (plugin: PluginInfo) =>
  (plugin.contributions ?? []).map((one) => ({
    kind: WIRE_CONTRIBUTION_KINDS.get(one.kind) ?? PLUGIN_CONTRIBUTION_KIND.UNSPECIFIED,
    items: one.items.map((item) => ({ text: item.text, detail: item.detail, keys: item.keys })),
  }))

const toWirePlugin = (plugin: PluginInfo) => ({
  id: plugin.id,
  name: plugin.name,
  version: plugin.version,
  description: plugin.description,
  isActive: plugin.isActive,
  capabilities: [...plugin.capabilities],
  hasFrontend: plugin.hasFrontend,
  hasBackend: plugin.hasBackend,
  readers: (plugin.readers ?? []).map((one) => ({ id: one.id, extensions: [...one.extensions] })),
  settingsViews: plugin.settingsViews ?? [],
  contributions: toWireContributions(plugin),
  settings: plugin.settings.map(toWireSetting),
})

const toWireConfiguration = (answer: Configuration) => ({
  written: answer.written,
  path: answer.path,
  plugins: (answer.plugins ?? []).map(toWirePlugin),
  app: (answer.app ?? []).map(toWireSetting),
})

/** What the core does with the settings, in the window's types. A call left out is unanswered. */
export interface SettingsCore {
  getSettings(): Promise<Configuration>
  updateSettings(edits: readonly SettingEdit[]): Promise<void>
  watchSettings(signal: AbortSignal): AsyncIterable<readonly SettingEdit[]>
  retryPlugin(pluginId: string): Promise<string>
  choosePluginPath(
    pluginId: string,
    key: string,
  ): Promise<{ readonly path: string; readonly isChosen: boolean; readonly problem: string }>
  getPluginData(pluginId: string): Promise<number>
  deletePluginData(pluginId: string): Promise<void>
}

type Answers = Record<string, (request: never, context: never) => object | AsyncIterable<object>>

const answerSettings = (core: Partial<SettingsCore>): Answers => {
  const { getSettings, updateSettings, watchSettings } = core
  return {
    ...(getSettings ? { getSettings: async () => toWireConfiguration(await getSettings()) } : {}),
    ...(updateSettings
      ? {
          writeSettings: async (request: { settings: readonly SettingEdit[] }) => {
            await updateSettings(
              request.settings.map((one) => ({ at: [...one.at], value: one.value })),
            )
            return {}
          },
        }
      : {}),
    ...(watchSettings
      ? {
          watchSettings: async function* (_request: object, context: { signal: AbortSignal }) {
            for await (const batch of watchSettings(context.signal)) yield { changed: batch }
          },
        }
      : {}),
  }
}

const answerPlugins = (core: Partial<SettingsCore>): Answers => {
  const { retryPlugin, choosePluginPath, getPluginData, deletePluginData } = core
  return {
    ...(retryPlugin
      ? {
          retryPlugin: async (r: { pluginId: string }) => ({
            failure: await retryPlugin(r.pluginId),
          }),
        }
      : {}),
    ...(choosePluginPath
      ? {
          choosePluginPath: (r: { pluginId: string; key: string }) =>
            choosePluginPath(r.pluginId, r.key),
        }
      : {}),
    ...(getPluginData
      ? {
          getPluginData: async (r: { pluginId: string }) => ({
            fileCount: await getPluginData(r.pluginId),
          }),
        }
      : {}),
    ...(deletePluginData
      ? {
          deletePluginData: async (r: { pluginId: string }) => {
            await deletePluginData(r.pluginId)
            return {}
          },
        }
      : {}),
  }
}

/**
 * Answers the settings calls of every client from `core`, through the real clients. The returned
 * function restores the transport installed before. `others` answers the calls of the other
 * clients; the plugin catalog lists nothing unless it says otherwise.
 */
export const installSettingsCore = (
  core: Partial<SettingsCore>,
  others: CoreAnswers = {},
): (() => void) =>
  installCoreAnswers({
    pluginCatalog: { listCatalog: () => ({ entries: [], sourceErrors: [] }) },
    ...others,
    settingsService: {
      ...answerSettings(core),
      ...answerPlugins(core),
    },
  })
