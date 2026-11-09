/**
 * Settings domain methods for the window core.
 */
import { settingsService } from '@/shared/clients'
import { fetched } from './words'
import type { Configuration, PluginSettingCondition, PluginSettingInfo } from '@/entities/settings'
import type { SettingsPort } from '@/app/ports/settings'

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

const readJson = (text: string): unknown => {
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
    ...(s.operation ? { operation: s.operation } : {}),
    min: s.min,
    max: s.max,
    step: s.step,
    activeOptions: s.activeOptions.map(readOption),
    isShown: Boolean(s.isShown),
    value: readJson(s.valueJson),
  }
}

export const settingsCore: SettingsPort = {
  getSettings: async () => {
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
        presence: fetched[one.presence],
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
    } satisfies Configuration
  },
  updateSettings: async (edits) => {
    await settingsService.writeSettings({
      settings: edits.map((one) => ({ at: [...one.at], value: one.value })),
    })
  },
  watchSettings: async function* (signal) {
    for await (const update of settingsService.watchSettings({}, { signal })) {
      yield update.changed.map((one) => ({ at: [...one.at], value: one.value }))
    }
  },
  choosePluginPath: async (pluginId, key) => {
    const answer = await settingsService.choosePluginPath({ pluginId, key })
    return { path: answer.path, isChosen: answer.isChosen, problem: answer.problem }
  },
  retryPlugin: async (pluginId) => {
    const answer = await settingsService.retryPlugin({ pluginId })
    return answer.failure
  },
  getPluginData: async (pluginId) => {
    const answer = await settingsService.getPluginData({ pluginId })
    return answer.fileCount
  },
  deletePluginData: async (pluginId) => {
    await settingsService.deletePluginData({ pluginId })
  },
}
