/**
 * The settings of the vault: what is in force, what is written, the changes that follow, and the
 * operations on a plugin that belong to its page. Each answer is mapped into the types of this
 * entity.
 */
import { MODEL_PRESENCE, type ModelPresenceCode } from '@numen/core-client'
import { settingsService } from '@/shared/api/clients'
import type { JsonValue } from '@/shared/api/json'
import type {
  Configuration,
  PluginSettingCondition,
  PluginSettingInfo,
  Presence,
  SettingEdit,
} from '../lib/configuration'

/**
 * What a model's files are on this machine, in the words the window uses. Keyed by the schema, so
 * a presence added to it has to be given a word here before this compiles.
 */
const PRESENCE: Record<ModelPresenceCode, Presence> = {
  [MODEL_PRESENCE.UNSPECIFIED]: 'nothing to fetch',
  [MODEL_PRESENCE.PRESENT]: 'present',
  [MODEL_PRESENCE.NOT_FETCHED]: 'not fetched',
  [MODEL_PRESENCE.NOTHING_TO_FETCH]: 'nothing to fetch',
}

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

type ServedSettings = Awaited<ReturnType<typeof settingsService.getSettings>>
type ServedPlugin = NonNullable<ServedSettings['plugins']>[number]
type ServedSetting = NonNullable<ServedPlugin['settings']>[number]

const readJson = (text: string): JsonValue | undefined => {
  // Stryker disable next-line ConditionalExpression: empty text fails JSON.parse and leaves the value undefined
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
    min: s.min,
    max: s.max,
    step: s.step,
    activeOptions: s.activeOptions.map(readOption),
    isShown: Boolean(s.isShown),
    value: readJson(s.valueJson),
  }
}

/** Every setting currently, and the models the settings offer. */
export const getSettings = async (): Promise<Configuration> => {
  const answer = await settingsService.getSettings({})
  return {
    written: answer.written,
    path: answer.path,
    models: answer.models.map((one) => ({
      namedAt: one.namedAt,
      name: one.name,
      title: one.title,
      group: one.group,
      isDefault: one.isDefault,
      writes: one.writes.map((w) => ({ at: w.at, value: w.value })),
      presence: PRESENCE[one.presence],
      reasoningLevels: one.reasoningLevels.map((l) => ({ name: l.name, title: l.title })),
      defaultReasoning: one.defaultReasoning,
    })),
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
      ...(p.failure
        ? {
            failure: {
              reason: p.failure.reason,
              detail: p.failure.detail,
              failedAt: Number(p.failure.failedAtUnixMs),
            },
          }
        : {}),
      settings: p.settings.map(readPluginSetting),
    })),
    app: answer.app.map(readPluginSetting),
  }
}

export const updateSettings = async (edits: readonly SettingEdit[]): Promise<void> => {
  await settingsService.writeSettings({
    settings: edits.map((one) => ({ at: [...one.at], value: one.value })),
  })
}

/**
 * The settings that changed, until the signal aborts: a write by this window, another window, a
 * plugin or the palette, and an edit of the file by hand. Each batch holds the settings that
 * changed together, named by their declared path, with the value each now holds as JSON. The
 * stream opens with an empty batch, which the window ignores.
 */
export async function* watchSettings(signal: AbortSignal): AsyncGenerator<readonly SettingEdit[]> {
  for await (const update of settingsService.watchSettings({}, { signal })) {
    yield update.changed.map((one) => ({ at: [...one.at], value: one.value }))
  }
}

/** What the picker of a plugin setting of type file or folder answered. */
interface ChosenPath {
  readonly path: string
  readonly isChosen: boolean
  readonly problem: string
}

/**
 * Opens the system picker for a plugin setting of type file or folder, and writes the selected
 * path.
 */
export const choosePluginPath = async (pluginId: string, key: string): Promise<ChosenPath> => {
  const answer = await settingsService.choosePluginPath({ pluginId, key })
  return { path: answer.path, isChosen: answer.isChosen, problem: answer.problem }
}

/**
 * A plugin whose last start failed, started once more. The answer is why it failed again, and
 * empty where it started.
 */
export const retryPlugin = async (pluginId: string): Promise<string> =>
  (await settingsService.retryPlugin({ pluginId })).failure

/** How many files the folder of a plugin holds in the shown vault, `.numen/<plugin id>/`. */
export const getPluginData = async (pluginId: string): Promise<number> =>
  (await settingsService.getPluginData({ pluginId })).fileCount

/** The folder of a plugin in the shown vault deleted, with the text versions the plugin made there. */
export const deletePluginData = async (pluginId: string): Promise<void> => {
  await settingsService.deletePluginData({ pluginId })
}
