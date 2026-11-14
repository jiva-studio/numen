/**
 * The settings of the vault: what is in force, what is written, the changes that follow, and the
 * operations on a plugin that belong to its page. Each answer is mapped into the types of this
 * entity.
 */
import {
  PLUGIN_CONTRIBUTION_KIND,
  asFailure,
  asValue,
  type PluginContributionKindValue,
} from '@numen/core-client'
import type { WindowClients } from '@/shared/api/clients'
import type { JsonValue } from '@/shared/api/json'
import type { Result } from '@/shared/lib/result'
import type {
  Configuration,
  PluginContribution,
  PluginContributionKind,
  PluginSettingCondition,
  PluginSettingInfo,
  SettingEdit,
} from '../lib/configuration'

const readOption = (one: { value: string; title: string; detail: string; when: string }) => ({
  value: one.value,
  title: one.title,
  detail: one.detail,
  when: one.when,
})

const readCondition = (
  when: { setting: string; is: readonly string[] } | undefined,
): PluginSettingCondition | undefined =>
  when ? { setting: when.setting, is: [...when.is] } : undefined

type ServedSettings = Awaited<ReturnType<WindowClients['settingsService']['getSettings']>>
type ServedPlugin = NonNullable<ServedSettings['plugins']>[number]
type ServedSetting = NonNullable<ServedPlugin['settings']>[number]
type ServedContribution = NonNullable<ServedPlugin['contributions']>[number]

type ContributionKinds = Record<PluginContributionKindValue, PluginContributionKind | null>

const CONTRIBUTION_KINDS: Readonly<ContributionKinds> = {
  [PLUGIN_CONTRIBUTION_KIND.UNSPECIFIED]: null,
  [PLUGIN_CONTRIBUTION_KIND.COMMANDS]: 'commands',
  [PLUGIN_CONTRIBUTION_KIND.KEYBINDINGS]: 'keybindings',
  [PLUGIN_CONTRIBUTION_KIND.TAB_KINDS]: 'tabKinds',
  [PLUGIN_CONTRIBUTION_KIND.READERS]: 'readers',
  [PLUGIN_CONTRIBUTION_KIND.HANDLERS]: 'handlers',
  [PLUGIN_CONTRIBUTION_KIND.TRIGGERS]: 'triggers',
  [PLUGIN_CONTRIBUTION_KIND.TEXT_KINDS]: 'textKinds',
  [PLUGIN_CONTRIBUTION_KIND.NOTE_TYPES]: 'noteTypes',
  [PLUGIN_CONTRIBUTION_KIND.THEMES]: 'themes',
  [PLUGIN_CONTRIBUTION_KIND.SLOTS]: 'slots',
  [PLUGIN_CONTRIBUTION_KIND.LISTENERS]: 'listeners',
  [PLUGIN_CONTRIBUTION_KIND.VIEW_CAPABILITIES]: 'viewCapabilities',
  [PLUGIN_CONTRIBUTION_KIND.DROP_HANDLERS]: 'dropHandlers',
  [PLUGIN_CONTRIBUTION_KIND.FRAME_ORIGINS]: 'frameOrigins',
  [PLUGIN_CONTRIBUTION_KIND.CONNECTION_TYPES]: 'connectionTypes',
  [PLUGIN_CONTRIBUTION_KIND.OPERATIONS]: 'operations',
  [PLUGIN_CONTRIBUTION_KIND.PERMISSIONS]: 'permissions',
  [PLUGIN_CONTRIBUTION_KIND.DEPENDENCIES]: 'dependencies',
  [PLUGIN_CONTRIBUTION_KIND.SERVICES]: 'services',
  [PLUGIN_CONTRIBUTION_KIND.CAPABILITIES]: 'capabilities',
  [PLUGIN_CONTRIBUTION_KIND.ACTIVATION_EVENTS]: 'activationEvents',
}

const readContributionItem = (item: ServedContribution['items'][number]) => ({
  text: item.text,
  detail: item.detail,
  keys: item.keys,
})

const readContributions = (served: readonly ServedContribution[]): readonly PluginContribution[] =>
  served.flatMap((one) => {
    const kind = CONTRIBUTION_KINDS[one.kind]
    return kind === null ? [] : [{ kind, items: one.items.map(readContributionItem) }]
  })

const readJson = (text: string): JsonValue | undefined => {
  // Stryker disable next-line ConditionalExpression: empty text fails JSON.parse and leaves the
  // value undefined
  if (!text) return undefined
  /* Stryker disable BlockStatement: the value is already undefined when JSON.parse throws */
  try {
    return JSON.parse(text)
  } catch {
    return undefined
  }
  /* Stryker restore BlockStatement */
}

const readPluginSetting = (s: ServedSetting): PluginSettingInfo => {
  const defaultValue = readJson(s.defaultJson)
  return {
    key: s.key,
    type: s.type as PluginSettingInfo['type'],
    title: s.title,
    description: s.description,
    ...(s.details ? { details: s.details } : {}),
    default: defaultValue,
    palette: s.palette,
    ...(s.icon ? { icon: s.icon } : {}),
    options: s.options.length > 0 ? s.options.map(readOption) : undefined,
    optionsBy: s.optionsBy
      ? {
          setting: s.optionsBy.setting,
          options: Object.fromEntries(
            Object.entries(s.optionsBy.options).map(([value, list]) => [
              value,
              list.options.map(readOption),
            ]),
          ),
        }
      : undefined,
    when: readCondition(s.when),
    accept: s.accept,
    ...(s.multiple ? { multiple: true } : {}),
    ...(s.preview ? { preview: true } : {}),
    ...(s.optionsFrom ? { optionsFrom: s.optionsFrom } : {}),
    ...(s.operations.length > 0 ? { operations: s.operations } : {}),
    ...(s.discover ? { discover: true } : {}),
    min: s.min,
    max: s.max,
    step: s.step,
    activeOptions: s.activeOptions.map(readOption),
    isShown: Boolean(s.isShown),
    value: readJson(s.valueJson),
  }
}

/** What the picker of a plugin setting of type file or folder answered. */
interface ChosenPath {
  readonly path: string
  readonly isChosen: boolean
  readonly problem: string
}

/** The settings of the vault over `settingsService`. */
export const createSettingsApi = (settingsService: WindowClients['settingsService']) => {
  /**
   * The settings that changed, until the signal aborts: a write by this window, another window, a
   * plugin or the palette, and an edit of the file by hand. Each batch holds the settings that
   * changed together, named by their declared path, with the value each now holds as JSON. The
   * stream opens with an empty batch, which the window ignores.
   */
  async function* watchSettings(signal: AbortSignal): AsyncGenerator<readonly SettingEdit[]> {
    for await (const update of settingsService.watchSettings({}, { signal })) {
      yield update.changed.map((one) => ({ at: [...one.at], value: one.value }))
    }
  }

  return {
    watchSettings,

    /** Every setting currently. */
    getSettings: async (): Promise<Configuration> => {
      const answer = await settingsService.getSettings({})
      return {
        written: answer.written,
        path: answer.path,
        plugins: answer.plugins.map((p) => ({
          id: p.id,
          name: p.name,
          version: p.version,
          description: p.description,
          isActive: p.isActive,
          capabilities: p.capabilities,
          hasFrontend: p.hasFrontend,
          hasBackend: p.hasBackend,
          readers: p.readers.map((one) => ({ id: one.id, extensions: one.extensions })),
          settingsViews: p.settingsViews.map((one) => ({ id: one.id, title: one.title })),
          contributions: readContributions(p.contributions),
          isOffByDefault: p.isOffByDefault,
          settings: p.settings.map(readPluginSetting),
        })),
        app: answer.app.map(readPluginSetting),
      }
    },

    updateSettings: async (edits: readonly SettingEdit[]): Promise<void> => {
      await settingsService.writeSettings({
        settings: edits.map((one) => ({ at: [...one.at], value: one.value })),
      })
    },

    /**
     * Opens the system picker for a plugin setting of type file or folder, and writes the selected
     * path.
     */
    choosePluginPath: async (pluginId: string, key: string): Promise<ChosenPath> => {
      const answer = await settingsService.choosePluginPath({ pluginId, key })
      return { path: answer.path, isChosen: answer.isChosen, problem: answer.problem }
    },

    /**
     * A plugin whose last start failed, started once more. A failure is why it failed again.
     */
    retryPlugin: async (pluginId: string): Promise<Result<null, string>> => {
      const { failure } = await settingsService.retryPlugin({ pluginId })
      return failure === '' ? asValue(null) : asFailure(failure)
    },

    /** How many files the folder of a plugin holds in the shown vault, `.numen/<plugin id>/`. */
    getPluginData: async (pluginId: string): Promise<number> =>
      (await settingsService.getPluginData({ pluginId })).fileCount,

    /**
     * The folder of a plugin in the shown vault deleted, with the text versions the plugin made
     * there.
     */
    deletePluginData: async (pluginId: string): Promise<void> => {
      await settingsService.deletePluginData({ pluginId })
    },
  }
}

export type SettingsApi = ReturnType<typeof createSettingsApi>
